package laboratorio

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/api/cloudscheduler/v1"
	"google.golang.org/api/monitoring/v3"
)

// Janela de observação de métricas.
type Janela struct {
	Inicio time.Time
	Fim    time.Time
}

// Duracao da janela.
func (j Janela) Duracao() time.Duration { return j.Fim.Sub(j.Inicio) }

// Serie lê uma métrica do Monitoring agregada: alinhador por série, redutor
// entre séries (vazio = nenhum) e período de alinhamento. As amostras vêm em
// ordem cronológica; distribuições reduzidas por percentil chegam como valor.
func (n *Nuvem) Serie(ctx context.Context, filtro string, j Janela, alinhador, redutor string, periodo time.Duration) ([]Amostra, error) {
	chamada := n.monitoramento.Projects.TimeSeries.List("projects/" + n.projeto).
		Filter(filtro).
		IntervalStartTime(j.Inicio.UTC().Format(time.RFC3339Nano)).
		IntervalEndTime(j.Fim.UTC().Format(time.RFC3339Nano)).
		AggregationAlignmentPeriod(fmt.Sprintf("%ds", int(periodo.Seconds()))).
		AggregationPerSeriesAligner(alinhador)
	if redutor != "" {
		chamada = chamada.AggregationCrossSeriesReducer(redutor)
	}
	resp, err := chamada.Context(ctx).Do()
	if err != nil {
		return nil, err
	}
	return amostrasDe(resp), nil
}

// Distribuicao lê uma métrica de distribuição somada na janela inteira e
// devolve a contagem e a média dos valores.
func (n *Nuvem) Distribuicao(ctx context.Context, filtro string, j Janela) (contagem int64, media float64, err error) {
	resp, err := n.monitoramento.Projects.TimeSeries.List("projects/" + n.projeto).
		Filter(filtro).
		IntervalStartTime(j.Inicio.UTC().Format(time.RFC3339Nano)).
		IntervalEndTime(j.Fim.UTC().Format(time.RFC3339Nano)).
		AggregationAlignmentPeriod(fmt.Sprintf("%ds", int(j.Duracao().Seconds())+60)).
		AggregationPerSeriesAligner("ALIGN_DELTA").
		AggregationCrossSeriesReducer("REDUCE_SUM").
		Context(ctx).Do()
	if err != nil {
		return 0, 0, err
	}
	for _, serie := range resp.TimeSeries {
		for _, p := range serie.Points {
			if d := p.Value.DistributionValue; d != nil {
				contagem += d.Count
				if d.Count > 0 {
					media = d.Mean
				}
			}
		}
	}
	return contagem, media, nil
}

// Percentil de uma métrica de distribuição na janela inteira (um só valor).
func (n *Nuvem) Percentil(ctx context.Context, filtro string, j Janela, p int) (float64, bool, error) {
	amostras, err := n.Serie(ctx, filtro, j, "ALIGN_DELTA", fmt.Sprintf("REDUCE_PERCENTILE_%02d", p), j.Duracao()+time.Minute)
	if err != nil || len(amostras) == 0 {
		return 0, false, err
	}
	return amostras[len(amostras)-1].Valor, true, nil
}

// FiltroFuncao seleciona o serviço Cloud Run da função.
func FiltroFuncao(nomeFuncao string) string {
	return fmt.Sprintf(`resource.type="cloud_run_revision" AND resource.labels.service_name="%s"`, nomeFuncao)
}

// FiltroGateway seleciona a API gerenciada do gateway (métricas do Service
// Control, resource.type="api"); metodo vazio soma todas as operações.
func FiltroGateway(prefixo, metodo string) string {
	f := fmt.Sprintf(`resource.type="api" AND resource.labels.service=monitoring.regex.full_match("%s-api-.*\.apigateway\..*")`, prefixo)
	if metodo != "" {
		f += fmt.Sprintf(` AND resource.labels.method=monitoring.regex.full_match(".*\.%s")`, metodo)
	}
	return f
}

// InstanciasFuncao devolve, por minuto, o máximo de instâncias alocadas à
// função (ativas e ociosas somadas: a amostragem do Monitoring é instantânea e,
// com requisições curtas, a instância costuma estar ociosa no instante lido).
func (n *Nuvem) InstanciasFuncao(ctx context.Context, nomeFuncao string, j Janela) ([]Amostra, error) {
	filtro := `metric.type="run.googleapis.com/container/instance_count" AND ` + FiltroFuncao(nomeFuncao)
	return n.Serie(ctx, filtro, j, "ALIGN_MAX", "REDUCE_SUM", time.Minute)
}

// LatenciaFuncao devolve o percentil (ms) de run.googleapis.com/request_latencies na janela.
func (n *Nuvem) LatenciaFuncao(ctx context.Context, nomeFuncao string, j Janela, p int) (float64, bool, error) {
	return n.Percentil(ctx, `metric.type="run.googleapis.com/request_latencies" AND `+FiltroFuncao(nomeFuncao), j, p)
}

// LatenciaFuncaoPorMinuto devolve o percentil (ms) por minuto.
func (n *Nuvem) LatenciaFuncaoPorMinuto(ctx context.Context, nomeFuncao string, j Janela, p int) ([]Amostra, error) {
	return n.Serie(ctx, `metric.type="run.googleapis.com/request_latencies" AND `+FiltroFuncao(nomeFuncao), j, "ALIGN_DELTA", fmt.Sprintf("REDUCE_PERCENTILE_%02d", p), time.Minute)
}

// LatenciaGateway devolve o percentil (ms) de uma métrica de latência do
// gateway: "request_latencies" (total), "request_latencies_backend" ou
// "request_latencies_overhead".
func (n *Nuvem) LatenciaGateway(ctx context.Context, prefixo, metrica, metodo string, j Janela, p int) (float64, bool, error) {
	v, ok, err := n.Percentil(ctx, fmt.Sprintf(`metric.type="serviceruntime.googleapis.com/api/%s" AND `, metrica)+FiltroGateway(prefixo, metodo), j, p)
	return v * 1000, ok, err
}

// RequisicoesFuncao conta as requisições da função na janela (run.googleapis.com/request_count).
func (n *Nuvem) RequisicoesFuncao(ctx context.Context, nomeFuncao string, j Janela) (int64, error) {
	amostras, err := n.Serie(ctx, `metric.type="run.googleapis.com/request_count" AND `+FiltroFuncao(nomeFuncao), j, "ALIGN_DELTA", "REDUCE_SUM", j.Duracao()+time.Minute)
	if err != nil {
		return 0, err
	}
	var total int64
	for _, a := range amostras {
		total += int64(a.Valor)
	}
	return total, nil
}

// PartidasFrias conta as partidas de instância da função na janela e devolve
// a média (ms) de run.googleapis.com/container/startup_latencies.
func (n *Nuvem) PartidasFrias(ctx context.Context, nomeFuncao string, j Janela) (int64, float64, error) {
	return n.Distribuicao(ctx, `metric.type="run.googleapis.com/container/startup_latencies" AND `+FiltroFuncao(nomeFuncao), j)
}

// --- Cloud Scheduler -------------------------------------------------------------

// PausarAgendamento pausa um job do Cloud Scheduler (ex.: a reconciliação, que
// mantém a função aquecida a cada 5 min) e devolve a função que o retoma.
func (n *Nuvem) PausarAgendamento(ctx context.Context, regiao, job string) (func(context.Context) error, error) {
	svc, err := cloudscheduler.NewService(ctx)
	if err != nil {
		return nil, err
	}
	nome := fmt.Sprintf("projects/%s/locations/%s/jobs/%s", n.projeto, regiao, job)
	if _, err := svc.Projects.Locations.Jobs.Pause(nome, &cloudscheduler.PauseJobRequest{}).Context(ctx).Do(); err != nil {
		return nil, fmt.Errorf("pausar %s: %w", job, err)
	}
	return func(ctx context.Context) error {
		_, err := svc.Projects.Locations.Jobs.Resume(nome, &cloudscheduler.ResumeJobRequest{}).Context(ctx).Do()
		return err
	}, nil
}

func amostrasDe(resp *monitoring.ListTimeSeriesResponse) []Amostra {
	var amostras []Amostra
	for _, serie := range resp.TimeSeries {
		for _, p := range serie.Points {
			quando, _ := time.Parse(time.RFC3339Nano, p.Interval.EndTime)
			var v float64
			switch {
			case p.Value.Int64Value != nil:
				v = float64(*p.Value.Int64Value)
			case p.Value.DoubleValue != nil:
				v = *p.Value.DoubleValue
			case p.Value.DistributionValue != nil:
				v = p.Value.DistributionValue.Mean
			}
			amostras = append(amostras, Amostra{Quando: quando, Valor: v})
		}
	}
	ordenar(amostras)
	return amostras
}
