package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/ThiagoCristovao/serverless-vending-validation/sistema-central-simulado/internal/laboratorio"
)

// frio mede a primeira requisição após um período de inatividade da função.
// Como o Cloud Scheduler chama a reconciliação a cada 5 min e mantém uma
// instância aquecida, o job é pausado durante a medição e retomado ao final.
func (x *avaliacao) frio(ctx context.Context) error {
	x.rel.Parametro("projeto", x.cfg.Projeto)
	x.rel.Parametro("gateway", x.cfg.Gateway)
	x.rel.Parametro("configuração da função", x.configuracao)
	x.rel.Parametro("amostras", x.amostras)
	x.rel.Parametro("inatividade antes de cada amostra", x.inatividade)
	x.rel.Parametro("máquina", x.cfg.MaquinaA)
	job := x.cfg.Prefixo + "-reconciliar-publicacoes"
	x.rel.Parametro("agendador pausado", job)

	retomar, err := x.nuvem.PausarAgendamento(ctx, x.cfg.Regiao, job)
	if err != nil {
		return err
	}
	x.rel.Evento("job %s pausado", job)
	defer func() {
		if err := retomar(context.Background()); err != nil {
			x.rel.Verificar(false, "retomar o job %s: %v (retome manualmente)", job, err)
		} else {
			x.rel.Evento("job %s retomado", job)
		}
	}()

	type medida struct {
		quando                                        time.Time
		instanciasAntes                               string
		fria, quente                                  time.Duration
		statusFrio                                    int
		funcaoFria, funcaoQ                           time.Duration
		partidas                                      int64
		partidaMedia                                  float64
		gatewayTotal, gatewayBackend, gatewayOverhead float64
		gatewayOk                                     bool
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
				if m.funcaoFria == 0 {
					m.funcaoFria = e.Latencia
				} else if m.funcaoQ == 0 {
					m.funcaoQ = e.Latencia
				}
			}
		}
		m.partidas, m.partidaMedia, _ = x.nuvem.PartidasFrias(ctx, x.cfg.NomeFuncao(), j)
		// Máximo (p99) das métricas do gateway na janela da amostra: separa o que
		// foi sobrecarga do próprio gateway do que foi espera pelo back-end.
		if v, ok, err := x.nuvem.LatenciaGateway(ctx, x.cfg.Prefixo, "request_latencies", "", j, 99); err == nil && ok {
			m.gatewayTotal, m.gatewayOk = v, true
			m.gatewayBackend, _, _ = x.nuvem.LatenciaGateway(ctx, x.cfg.Prefixo, "request_latencies_backend", "", j, 99)
			m.gatewayOverhead, _, _ = x.nuvem.LatenciaGateway(ctx, x.cfg.Prefixo, "request_latencies_overhead", "", j, 99)
		}
		if m.statusFrio == 201 {
			frias = append(frias, m.fria)
		}
		gateway := "—"
		if m.gatewayOk {
			gateway = fmt.Sprintf("%.0f / %.0f / %.0f ms", m.gatewayTotal, m.gatewayBackend, m.gatewayOverhead)
		}
		linhas = append(linhas, []string{fmt.Sprint(i + 1), m.quando.Local().Format("15:04:05"), m.instanciasAntes,
			fmt.Sprintf("%d em %s", m.statusFrio, laboratorio.Dur(m.fria)), laboratorio.Dur(m.quente),
			laboratorio.Dur(m.funcaoFria), laboratorio.Dur(m.funcaoQ), fmt.Sprintf("%d (%.0f ms)", m.partidas, m.partidaMedia), gateway})
	}
	x.rel.Tabela("Amostras", []string{"#", "Hora", "Instâncias antes (minuto)", "1ª requisição (cliente)", "2ª requisição (cliente)", "1ª na função (log)", "2ª na função (log)", "Partidas de instância (média)", "Gateway, máx. na janela: total / back-end / sobrecarga"}, linhas)

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
