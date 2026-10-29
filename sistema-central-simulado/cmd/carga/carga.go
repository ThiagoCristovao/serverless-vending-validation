package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/ThiagoCristovao/serverless-vending-validation/sistema-central-simulado/internal/laboratorio"
)

// carga coloca N operadores virtuais em laço fechado (abrir validação,
// concluir, pausar) contra o gateway e, ao final, cruza as medições do cliente
// com as métricas do Monitoring (gateway, função, instâncias, partidas).
func (x *avaliacao) carga(ctx context.Context) error {
	lote, err := x.carregarLote()
	if err != nil {
		return err
	}
	if len(lote.Operadores) < x.operadores || len(lote.Maquinas) < x.operadores {
		return fmt.Errorf("lote tem %d operadores e %d máquinas; rode make carga-preparar OPERADORES=%d", len(lote.Operadores), len(lote.Maquinas), x.operadores)
	}
	x.rel.Parametro("projeto", x.cfg.Projeto)
	x.rel.Parametro("gateway", x.cfg.Gateway)
	x.rel.Parametro("configuração da função", x.configuracao)
	x.rel.Parametro("operadores simultâneos", x.operadores)
	x.rel.Parametro("rampa", x.rampa)
	x.rel.Parametro("duração em regime", x.duracao)
	x.rel.Parametro("pausa entre validações", x.pausa)
	x.rel.Parametro("fluxo de cada operador", "POST /validacoes (chave nova) → POST /validacoes/{id}/conclusao → pausa")

	// As máquinas de carga ficam na localização do lote; o adesivo precisa
	// coincidir com o cadastro (o serviço devolve 422 assinatura_invalida se não).
	cfgCarga := x.cfg
	cfgCarga.Loc = locCarga
	adesivo, err := laboratorio.NovoAdesivo(cfgCarga)
	if err != nil {
		return err
	}
	emails := make([]string, x.operadores)
	for i := range emails {
		emails[i] = lote.Operadores[i].Email
	}
	x.rel.Evento("obtendo tokens de %d operadores", x.operadores)
	tokens, err := x.tokens(ctx, emails, x.rampa+x.duracao+5*time.Minute)
	if err != nil {
		return err
	}

	cliente := &http.Client{Timeout: 30 * time.Second, Transport: &http.Transport{MaxIdleConns: 2 * x.operadores, MaxIdleConnsPerHost: 2 * x.operadores, IdleConnTimeout: 90 * time.Second}}
	col := &coletor{}
	inicio := time.Now()
	fimRampa := inicio.Add(x.rampa)
	fim := fimRampa.Add(x.duracao)
	prazo, cancelar := context.WithDeadline(ctx, fim)
	defer cancelar()

	x.rel.Evento("rampa de %s; regime até %s", x.rampa, fim.Format("15:04:05"))
	var wg sync.WaitGroup
	for i := 0; i < x.operadores; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			atraso := time.Duration(int64(x.rampa) * int64(i) / int64(x.operadores))
			select {
			case <-time.After(atraso):
			case <-prazo.Done():
				return
			}
			g := &laboratorio.Gateway{Base: x.cfg.Gateway, Token: tokens[emails[i]].Token, Cliente: cliente}
			x.operadorVirtual(prazo, i, g, adesivo, lote.Maquinas[i].Id, col)
		}(i)
	}
	wg.Wait()
	x.rel.Evento("carga encerrada")

	todas := col.todas()
	if len(todas) == 0 {
		return errors.New("nenhuma requisição registrada")
	}
	regime := filtrarDesde(todas, fimRampa)
	x.rel.Tabela("Latência medida no cliente (todas as requisições, inclusive a rampa)",
		[]string{"Requisição", "N", "Erros", "Média", "p50", "p95", "p99", "Máx."},
		[][]string{resumir(todas).linha("todas"), resumir(filtrar(todas, "abertura")).linha("abertura"), resumir(filtrar(todas, "conclusao")).linha("conclusão")})
	rr := resumir(regime)
	x.rel.Tabela(fmt.Sprintf("Latência medida no cliente em regime (%s após a rampa)", x.duracao),
		[]string{"Requisição", "N", "Erros", "Média", "p50", "p95", "p99", "Máx."},
		[][]string{rr.linha("todas"), resumir(filtrar(regime, "abertura")).linha("abertura"), resumir(filtrar(regime, "conclusao")).linha("conclusão")})
	vazao := float64(len(regime)) / x.duracao.Seconds()
	x.rel.Secao("Vazão em regime", fmt.Sprintf("%d requisições em %s: **%.1f req/s** (%.1f validações/min).", len(regime), x.duracao, vazao, vazao*60/2))

	var linhas [][]string
	for i, grupo := range porIntervalo(todas, inicio, 10*time.Second) {
		r := resumir(grupo)
		linhas = append(linhas, []string{fmt.Sprintf("%ds–%ds", i*10, (i+1)*10), fmt.Sprint(r.N), fmt.Sprint(r.Erros), ms(r.P50), ms(r.P95), ms(r.Max)})
	}
	x.rel.Tabela("Linha do tempo no cliente (janelas de 10 s desde o início da rampa)", []string{"Janela", "N", "Erros", "p50", "p95", "Máx."}, linhas)

	var errosLinhas [][]string
	for _, a := range todas {
		if a.Erro && len(errosLinhas) < 20 {
			errosLinhas = append(errosLinhas, []string{a.Quando.Format("15:04:05.000"), fmt.Sprint(a.Operador), a.Tipo, fmt.Sprint(a.Status), ms(a.Duracao)})
		}
	}
	if len(errosLinhas) > 0 {
		x.rel.Tabela("Requisições com erro (até 20)", []string{"Hora", "Operador", "Tipo", "Status", "Duração"}, errosLinhas)
	}

	// As métricas do Cloud Run chegam com 3 a 4 min de atraso; as do gateway, antes.
	x.rel.Evento("aguardando 4 min pela ingestão das métricas do Monitoring")
	select {
	case <-time.After(4 * time.Minute):
	case <-ctx.Done():
		return ctx.Err()
	}
	janela := laboratorio.Janela{Inicio: inicio, Fim: fim}
	gw := x.latenciasGateway(ctx, janela)
	fn := x.latenciasFuncao(ctx, janela)
	x.rel.Tabela("Latência no provedor (Monitoring, janela da rampa ao fim)",
		[]string{"Medida", "p50", "p95", "p99"},
		[][]string{
			{"gateway, total (`serviceruntime…/request_latencies`)", msf(gw.total.valor(50)), msf(gw.total.valor(95)), msf(gw.total.valor(99))},
			{"gateway, parcela do back-end (`…request_latencies_backend`)", msf(gw.backend.valor(50)), msf(gw.backend.valor(95)), msf(gw.backend.valor(99))},
			{"gateway, sobrecarga própria (`…request_latencies_overhead`)", msf(gw.overhead.valor(50)), msf(gw.overhead.valor(95)), msf(gw.overhead.valor(99))},
			{"função (`run.googleapis.com/request_latencies`)", msf(fn.valor(50)), msf(fn.valor(95)), msf(fn.valor(99))},
		})

	instancias, err := x.nuvem.InstanciasFuncao(ctx, x.cfg.NomeFuncao(), janela)
	if err != nil {
		x.rel.Evento("instâncias: %v", err)
	}
	var maxInst float64
	var instLinhas [][]string
	for _, a := range instancias {
		if a.Valor > maxInst {
			maxInst = a.Valor
		}
		instLinhas = append(instLinhas, []string{a.Quando.Local().Format("15:04"), fmt.Sprintf("%.0f", a.Valor)})
	}
	x.rel.Tabela("Instâncias da função por minuto (ativas + ociosas, máximo no minuto)", []string{"Minuto", "Instâncias"}, instLinhas)
	reqs, _ := x.nuvem.RequisicoesFuncao(ctx, x.cfg.NomeFuncao(), janela)
	partidas, mediaPartida, _ := x.nuvem.PartidasFrias(ctx, x.cfg.NomeFuncao(), janela)
	fnMin95, _ := x.nuvem.LatenciaFuncaoPorMinuto(ctx, x.cfg.NomeFuncao(), janela, 95)
	var minLinhas [][]string
	for _, a := range fnMin95 {
		minLinhas = append(minLinhas, []string{a.Quando.Local().Format("15:04"), fmt.Sprintf("%.0f ms", a.Valor)})
	}
	x.rel.Tabela("p95 da função por minuto (Monitoring)", []string{"Minuto", "p95"}, minLinhas)
	x.rel.Secao("Escala e partidas", fmt.Sprintf("Requisições contadas pela função na janela: **%d**. Máximo de instâncias alocadas: **%.0f**. Partidas de instância (`startup_latencies`): **%d**, média **%.0f ms**.", reqs, maxInst, partidas, mediaPartida))

	// Critérios (docs/requisitos.md).
	x.rel.Verificar(rr.Erros == 0, "nenhuma requisição com erro em regime (%d de %d)", rr.Erros, rr.N)
	if v, ok := gw.total[95]; ok {
		x.rel.Verificar(v <= 500, "RNF-05: p95 no gateway com %d operadores = %.0f ms (alvo ≤ 500 ms)", x.operadores, v)
	} else {
		x.rel.Verificar(false, "RNF-05: p95 do gateway indisponível no Monitoring")
	}
	if x.operadores <= 10 {
		p50, ok50 := fn[50]
		p95, ok95 := fn[95]
		x.rel.Verificar(ok50 && ok95 && p50 <= 200 && p95 <= 500, "RNF-06 (regime contínuo): função p50 = %s, p95 = %s (alvos ≤ 200 ms e ≤ 500 ms)", msf(p50, ok50), msf(p95, ok95))
	}
	x.rel.Evento("cliente em regime: p50 %s, p95 %s, p99 %s; %.1f req/s", ms(rr.P50), ms(rr.P95), ms(rr.P99), vazao)
	return nil
}

// operadorVirtual repete o fluxo de validação até o prazo vencer.
func (x *avaliacao) operadorVirtual(ctx context.Context, i int, g *laboratorio.Gateway, adesivo *laboratorio.Adesivo, maq string, col *coletor) {
	for ctx.Err() == nil {
		ab, err := g.IniciarValidacao(ctx, adesivo, maq)
		if err != nil && ctx.Err() != nil {
			return // requisição interrompida pelo fim da janela: não conta
		}
		col.registrar(amostra{Quando: time.Now(), Operador: i, Tipo: "abertura", Status: ab.Resposta.Status, Duracao: ab.Resposta.Duracao, Erro: err != nil})
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s operador %d: %v\n", laboratorio.Hora(), i, resumirErro(err))
			x.esperar(ctx, x.pausa)
			continue
		}
		resp, err := g.ConcluirValidacao(ctx, ab.Id)
		if err != nil && ctx.Err() != nil {
			return
		}
		col.registrar(amostra{Quando: time.Now(), Operador: i, Tipo: "conclusao", Status: resp.Status, Duracao: resp.Duracao, Erro: err != nil})
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s operador %d: %v\n", laboratorio.Hora(), i, resumirErro(err))
		}
		x.esperar(ctx, x.pausa)
	}
}

func (x *avaliacao) esperar(ctx context.Context, d time.Duration) {
	select {
	case <-time.After(d):
	case <-ctx.Done():
	}
}

func resumirErro(err error) string {
	s := err.Error()
	if len(s) > 160 {
		return s[:160] + "…"
	}
	return s
}

func filtrarDesde(as []amostra, t time.Time) []amostra {
	var saida []amostra
	for _, a := range as {
		if !a.Quando.Before(t) {
			saida = append(saida, a)
		}
	}
	return saida
}

type percentis map[int]float64

type latenciasGateway struct{ total, backend, overhead percentis }

// latenciasGateway lê p50/p95/p99 das três métricas de latência do gateway.
func (x *avaliacao) latenciasGateway(ctx context.Context, j laboratorio.Janela) latenciasGateway {
	ler := func(metrica string) percentis {
		p := percentis{}
		for _, q := range []int{50, 95, 99} {
			v, ok, err := x.nuvem.LatenciaGateway(ctx, x.cfg.Prefixo, metrica, "", j, q)
			if err != nil {
				x.rel.Evento("gateway %s p%d: %v", metrica, q, err)
				continue
			}
			if ok {
				p[q] = v
			}
		}
		return p
	}
	lg := latenciasGateway{total: ler("request_latencies"), backend: ler("request_latencies_backend"), overhead: ler("request_latencies_overhead")}
	for _, q := range []int{50, 95, 99} {
		if _, ok := lg.backend[q]; !ok {
			if t, okT := lg.total[q]; okT {
				if o, okO := lg.overhead[q]; okO {
					lg.backend[q] = t - o // estimativa: a métrica do back-end não foi publicada
				}
			}
		}
	}
	return lg
}

func (x *avaliacao) latenciasFuncao(ctx context.Context, j laboratorio.Janela) percentis {
	p := percentis{}
	for _, q := range []int{50, 95, 99} {
		v, ok, err := x.nuvem.LatenciaFuncao(ctx, x.cfg.NomeFuncao(), j, q)
		if err != nil {
			x.rel.Evento("função p%d: %v", q, err)
			continue
		}
		if ok {
			p[q] = v
		}
	}
	return p
}

// msf aceita o par (valor, presente) de um mapa de percentis.
func (p percentis) valor(q int) (float64, bool) { v, ok := p[q]; return v, ok }
