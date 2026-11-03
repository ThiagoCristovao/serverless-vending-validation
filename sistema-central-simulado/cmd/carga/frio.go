package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ThiagoCristovao/serverless-vending-validation/sistema-central-simulado/internal/laboratorio"
)

// frio mede a primeira requisição após um período de inatividade da função.
// Como o Cloud Scheduler chama a reconciliação a cada 5 min e mantém uma
// instância aquecida, o job é pausado durante a medição e retomado ao final;
// com -agendador manter, nada é pausado e a medição reflete o uso normal, com
// o job de aquecimento do gateway ativo (ADR-0017).
func (x *avaliacao) frio(ctx context.Context) error {
	x.rel.Parametro("projeto", x.cfg.Projeto)
	x.rel.Parametro("gateway", x.cfg.Gateway)
	x.rel.Parametro("configuração da função", x.configuracao)
	x.rel.Parametro("amostras", x.amostras)
	x.rel.Parametro("inatividade antes de cada amostra", x.inatividade)
	x.rel.Parametro("máquina", x.cfg.MaquinaA)
	switch x.agendador {
	case "manter":
		x.rel.Parametro("agendador", "jobs mantidos (reconciliação e aquecimento do gateway ativos)")
		x.rel.Evento("jobs do Scheduler mantidos durante a medição")
	case "pausar":
		// Reconciliação (mantém a função aquecida) e aquecimento do gateway
		// (ADR-0017), quando existir.
		var pausados []string
		for _, job := range []string{x.cfg.Prefixo + "-reconciliar-publicacoes", x.cfg.Prefixo + "-aquecer-gateway"} {
			retomar, err := x.nuvem.PausarAgendamento(ctx, x.cfg.Regiao, job)
			if errors.Is(err, laboratorio.ErrJobInexistente) {
				continue
			}
			if err != nil {
				return err
			}
			pausados = append(pausados, job)
			x.rel.Evento("job %s pausado", job)
			defer func(job string) {
				if err := retomar(context.Background()); err != nil {
					x.rel.Verificar(false, "retomar o job %s: %v (retome manualmente)", job, err)
				} else {
					x.rel.Evento("job %s retomado", job)
				}
			}(job)
		}
		x.rel.Parametro("agendador pausado", strings.Join(pausados, ", "))
	default:
		return fmt.Errorf("-agendador deve ser pausar ou manter, não %q", x.agendador)
	}

	type medida struct {
		quando                time.Time
		instanciasAntes       string
		fria, quente          time.Duration
		statusFrio            int
		funcaoFria, funcaoQ   time.Duration
		partidas              int64
		partidaMedia          float64
		gatewayFria, gatewayQ time.Duration
		agenteFrio, agenteQ   string
	}
	var medidas []medida
	for i := 1; i <= x.amostras; i++ {
		x.rel.Evento("amostra %d/%d: aguardando %s de inatividade", i, x.amostras, x.inatividade)
		select {
		case <-time.After(x.inatividade):
		case <-ctx.Done():
			return ctx.Err()
		}
		antes := "—"
		if inst, err := x.nuvem.InstanciasFuncao(ctx, x.cfg.NomeFuncao(), laboratorio.Janela{Inicio: time.Now().Add(-6 * time.Minute), Fim: time.Now()}); err == nil && len(inst) > 0 {
			u := inst[len(inst)-1]
			antes = fmt.Sprintf("%.0f (%s)", u.Valor, u.Quando.Local().Format("15:04"))
		}
		op, err := laboratorio.EntrarComoOperador(ctx, x.cfg)
		if err != nil {
			return fmt.Errorf("token do operador: %w", err)
		}
		g := laboratorio.NovoGateway(x.cfg, op, func(l string) { fmt.Fprintln(os.Stderr, "   "+l) })
		m := medida{quando: time.Now(), instanciasAntes: antes}
		ab, err := g.IniciarValidacao(ctx, x.adesivo, x.cfg.MaquinaA)
		m.fria, m.statusFrio = ab.Resposta.Duracao, ab.Resposta.Status
		if err != nil {
			x.rel.Evento("amostra %d: abertura falhou: %v", i, resumirErro(err))
			medidas = append(medidas, m)
			continue
		}
		resp, err := g.ConcluirValidacao(ctx, ab.Id)
		m.quente = resp.Duracao
		if err != nil {
			x.rel.Evento("amostra %d: conclusão falhou: %v", i, resumirErro(err))
		}
		x.rel.Evento("amostra %d: fria %s, quente %s (instâncias antes: %s)", i, laboratorio.Dur(m.fria), laboratorio.Dur(m.quente), antes)
		medidas = append(medidas, m)
	}

	x.rel.Evento("aguardando 4 min pela ingestão das métricas e dos logs")
	select {
	case <-time.After(4 * time.Minute):
	case <-ctx.Done():
		return ctx.Err()
	}
	var linhas [][]string
	var frias []time.Duration
	for i := range medidas {
		m := &medidas[i]
		j := laboratorio.Janela{Inicio: m.quando.Add(-time.Minute), Fim: m.quando.Add(2 * time.Minute)}
		if entradas, err := x.nuvem.EntradasDaFuncao(ctx, x.cfg.NomeFuncao(), m.quando.Add(-5*time.Second), `httpRequest.requestUrl:"/v1/validacoes"`); err == nil {
			for _, e := range entradas {
				if e.Quando.After(m.quando.Add(90*time.Second)) || e.Metodo != "POST" {
					continue
				}
				if m.funcaoFria == 0 && eAbertura(e) {
					m.funcaoFria = e.Latencia
				} else if m.funcaoQ == 0 && eConclusao(e) {
					m.funcaoQ = e.Latencia
				}
			}
		}
		m.partidas, m.partidaMedia, _ = x.nuvem.PartidasFrias(ctx, x.cfg.NomeFuncao(), j)
		// Latência medida pelo próprio gateway (log de acesso) e o agente que
		// atendeu: o que falta entre o cliente e este valor é a partida do
		// gateway e a rede; o que falta entre este valor e a função é o gateway.
		if entradas, err := x.nuvem.EntradasDoGateway(ctx, m.quando.Add(-5*time.Second), `httpRequest.requestUrl:"/v1/validacoes"`); err == nil {
			for _, e := range entradas {
				if e.Quando.After(m.quando.Add(90*time.Second)) || e.Metodo != "POST" {
					continue
				}
				if m.gatewayFria == 0 && eAbertura(e) {
					m.gatewayFria, m.agenteFrio = e.Latencia, agenteDe(e)
				} else if m.gatewayQ == 0 && eConclusao(e) {
					m.gatewayQ, m.agenteQ = e.Latencia, agenteDe(e)
				}
			}
		}
		if m.statusFrio == 201 {
			frias = append(frias, m.fria)
		}
		linhas = append(linhas, []string{fmt.Sprint(i + 1), m.quando.Local().Format("15:04:05"), m.instanciasAntes,
			fmt.Sprintf("%d em %s", m.statusFrio, laboratorio.Dur(m.fria)), laboratorio.Dur(m.quente),
			fmt.Sprintf("%s (%s)", laboratorio.Dur(m.gatewayFria), m.agenteFrio), fmt.Sprintf("%s (%s)", laboratorio.Dur(m.gatewayQ), m.agenteQ),
			laboratorio.Dur(m.funcaoFria), laboratorio.Dur(m.funcaoQ), fmt.Sprintf("%d (%.0f ms)", m.partidas, m.partidaMedia)})
	}
	x.rel.Tabela("Amostras", []string{"#", "Hora", "Instâncias antes (minuto)", "1ª requisição (cliente)", "2ª requisição (cliente)", "1ª no gateway (log, agente)", "2ª no gateway (log, agente)", "1ª na função (log)", "2ª na função (log)", "Partidas de instância (média)"}, linhas)

	if len(frias) == 0 {
		x.rel.Verificar(false, "nenhuma amostra válida de cold start")
		return nil
	}
	as := make([]amostra, 0, len(frias))
	for _, d := range frias {
		as = append(as, amostra{Duracao: d})
	}
	r := resumir(as)
	x.rel.Secao("Resumo das primeiras requisições", fmt.Sprintf("%d amostras: média %s, mediana %s, máximo %s.", r.N, ms(r.Media), ms(r.P50), ms(r.Max)))
	x.rel.Verificar(r.P50 <= 1500*time.Millisecond, "RNF-06: mediana da primeira requisição após inatividade = %s (alvo ≤ 1,5 s)", ms(r.P50))
	acima := 0
	for _, d := range frias {
		if d > 1500*time.Millisecond {
			acima++
		}
	}
	x.rel.Verificar(acima == 0, "todas as primeiras requisições ≤ 1,5 s (%d de %d acima)", acima, len(frias))
	return nil
}

// agenteDe resume o service_agent do log do gateway ("ESPv2/2.56.0" → "ESPv2",
// "cloudesf_20261012_00_RC02" → "cloudesf").
func agenteDe(e laboratorio.EntradaLog) string {
	a := e.Campo("service_agent")
	if i := strings.IndexAny(a, "/_"); i > 0 {
		return a[:i]
	}
	return a
}

// eAbertura e eConclusao reconhecem, pelo caminho, a abertura (fria) e a
// conclusão (quente) de uma amostra; o gateway pode registrar a mesma
// requisição mais de uma vez (um agente por entrada), por isso a escolha
// não é pela ordem.
func eAbertura(e laboratorio.EntradaLog) bool  { return strings.HasSuffix(e.Caminho, "/v1/validacoes") }
func eConclusao(e laboratorio.EntradaLog) bool { return strings.HasSuffix(e.Caminho, "/conclusao") }
