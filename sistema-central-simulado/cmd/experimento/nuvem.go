package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"slices"
	"time"

	"cloud.google.com/go/firestore"
	"cloud.google.com/go/iam/apiv1/iampb"
	"cloud.google.com/go/pubsub/v2"
	"google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/logging/v2"
	"google.golang.org/api/monitoring/v3"
)

// Nuvem agrupa as APIs do provedor usadas para injetar falhas (IAM) e coletar
// evidências (Monitoring, Logging, Firestore). Usa as credenciais ADC.
type Nuvem struct {
	projeto       string
	monitoramento *monitoring.Service
	registros     *logging.Service
	crm           *cloudresourcemanager.Service
	firestore     *firestore.Client
	pubsub        *pubsub.Client
}

// NovaNuvem abre os clientes.
func NovaNuvem(ctx context.Context, projeto string, cli *pubsub.Client) (*Nuvem, error) {
	mon, err := monitoring.NewService(ctx)
	if err != nil {
		return nil, fmt.Errorf("monitoring: %w", err)
	}
	reg, err := logging.NewService(ctx)
	if err != nil {
		return nil, fmt.Errorf("logging: %w", err)
	}
	crm, err := cloudresourcemanager.NewService(ctx)
	if err != nil {
		return nil, fmt.Errorf("cloudresourcemanager: %w", err)
	}
	fs, err := firestore.NewClient(ctx, projeto)
	if err != nil {
		return nil, fmt.Errorf("firestore: %w", err)
	}
	return &Nuvem{projeto: projeto, monitoramento: mon, registros: reg, crm: crm, firestore: fs, pubsub: cli}, nil
}

// Fechar libera os clientes.
func (n *Nuvem) Fechar() { _ = n.firestore.Close() }

// Amostra de uma métrica.
type Amostra struct {
	Quando time.Time
	Valor  float64
}

// Metrica lê os pontos de uma métrica da assinatura desde um instante,
// em ordem cronológica.
func (n *Nuvem) Metrica(ctx context.Context, tipo, assinatura string, desde time.Time) ([]Amostra, error) {
	filtro := fmt.Sprintf(`metric.type="%s" AND resource.type="pubsub_subscription" AND resource.labels.subscription_id="%s"`, tipo, assinatura)
	resp, err := n.monitoramento.Projects.TimeSeries.List("projects/" + n.projeto).
		Filter(filtro).
		IntervalStartTime(desde.UTC().Format(time.RFC3339Nano)).
		IntervalEndTime(time.Now().UTC().Format(time.RFC3339Nano)).
		Context(ctx).Do()
	if err != nil {
		return nil, err
	}
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
			}
			amostras = append(amostras, Amostra{Quando: quando, Valor: v})
		}
	}
	slices.SortFunc(amostras, func(a, b Amostra) int { return a.Quando.Compare(b.Quando) })
	return amostras, nil
}

// Backlog lê num_undelivered_messages da assinatura.
func (n *Nuvem) Backlog(ctx context.Context, assinatura string, desde time.Time) ([]Amostra, error) {
	return n.Metrica(ctx, "pubsub.googleapis.com/subscription/num_undelivered_messages", assinatura, desde)
}

// IdadeMaisAntiga lê oldest_unacked_message_age (segundos) da assinatura.
func (n *Nuvem) IdadeMaisAntiga(ctx context.Context, assinatura string, desde time.Time) ([]Amostra, error) {
	return n.Metrica(ctx, "pubsub.googleapis.com/subscription/oldest_unacked_message_age", assinatura, desde)
}

// EntradaLog é uma linha de log da função.
type EntradaLog struct {
	Quando     time.Time
	Severidade string
	Mensagem   string
	// Requisicao resume o httpRequest do log de acesso do Cloud Run, se houver.
	Requisicao string
	Campos     map[string]any
}

// Campo devolve um campo do jsonPayload como texto ("—" se ausente).
func (e EntradaLog) Campo(nome string) string {
	if v, ok := e.Campos[nome]; ok && v != nil {
		return fmt.Sprint(v)
	}
	return "—"
}

// Rota devolve o caminho da requisição registrado pela função, se houver.
func (e EntradaLog) Rota() string {
	for _, chave := range []string{"rota", "caminho", "path"} {
		if v, ok := e.Campos[chave].(string); ok {
			return v
		}
	}
	return "—"
}

// EntradasDaFuncao lista entradas do serviço Cloud Run da função desde um
// instante que contenham o texto (busca livre do Logging).
func (n *Nuvem) EntradasDaFuncao(ctx context.Context, nomeFuncao string, desde time.Time, texto string) ([]EntradaLog, error) {
	filtro := fmt.Sprintf(`resource.type="cloud_run_revision" AND resource.labels.service_name="%s" AND timestamp>="%s"`, nomeFuncao, desde.UTC().Format(time.RFC3339))
	if texto != "" {
		filtro += fmt.Sprintf(` AND %s`, texto)
	}
	resp, err := n.registros.Entries.List(&logging.ListLogEntriesRequest{
		ResourceNames: []string{"projects/" + n.projeto},
		Filter:        filtro,
		OrderBy:       "timestamp asc",
		PageSize:      500,
	}).Context(ctx).Do()
	if err != nil {
		return nil, err
	}
	var saida []EntradaLog
	for _, e := range resp.Entries {
		quando, _ := time.Parse(time.RFC3339Nano, e.Timestamp)
		entrada := EntradaLog{Quando: quando, Severidade: e.Severity, Mensagem: e.TextPayload}
		if r := e.HttpRequest; r != nil {
			entrada.Requisicao = fmt.Sprintf("%s %s → %d em %s", r.RequestMethod, caminhoDe(r.RequestUrl), r.Status, r.Latency)
		}
		if len(e.JsonPayload) > 0 {
			_ = json.Unmarshal(e.JsonPayload, &entrada.Campos)
			for _, chave := range []string{"message", "msg", "mensagem"} {
				if m, ok := entrada.Campos[chave].(string); ok {
					entrada.Mensagem = m
					break
				}
			}
		}
		saida = append(saida, entrada)
	}
	return saida, nil
}

func caminhoDe(bruto string) string {
	if u, err := url.Parse(bruto); err == nil && u.Path != "" {
		return u.Path
	}
	return bruto
}

// Marcadores de publicação de uma validação (docs/modelo-dados.md).
type Marcadores struct {
	Status      string
	IniciadaEm  *time.Time
	ConcluidaEm *time.Time
}

// MarcadoresPublicacao lê eventos.iniciadaPublicadaEm e concluidaPublicadaEm.
func (n *Nuvem) MarcadoresPublicacao(ctx context.Context, idValidacao string) (Marcadores, error) {
	doc, err := n.firestore.Collection("validacoes").Doc(idValidacao).Get(ctx)
	if err != nil {
		return Marcadores{}, err
	}
	dados := doc.Data()
	m := Marcadores{}
	m.Status, _ = dados["status"].(string)
	eventos, _ := dados["eventos"].(map[string]any)
	if t, ok := eventos["iniciadaPublicadaEm"].(time.Time); ok {
		m.IniciadaEm = &t
	}
	if t, ok := eventos["concluidaPublicadaEm"].(time.Time); ok {
		m.ConcluidaEm = &t
	}
	return m, nil
}

// --- injeção de falhas por IAM --------------------------------------------------

// RemoverPublicador retira o membro do papel pubsub.publisher no tópico e
// devolve a função que restaura o vínculo.
func (n *Nuvem) RemoverPublicador(ctx context.Context, topico, membro string) (func(context.Context) error, error) {
	recurso := fmt.Sprintf("projects/%s/topics/%s", n.projeto, topico)
	politica, err := n.pubsub.TopicAdminClient.GetIamPolicy(ctx, &iampb.GetIamPolicyRequest{Resource: recurso})
	if err != nil {
		return nil, err
	}
	removido := false
	for _, b := range politica.Bindings {
		if b.Role == "roles/pubsub.publisher" && slices.Contains(b.Members, membro) {
			b.Members = slices.DeleteFunc(b.Members, func(m string) bool { return m == membro })
			removido = true
		}
	}
	if !removido {
		return nil, fmt.Errorf("%s não é publicador de %s", membro, topico)
	}
	if _, err := n.pubsub.TopicAdminClient.SetIamPolicy(ctx, &iampb.SetIamPolicyRequest{Resource: recurso, Policy: politica}); err != nil {
		return nil, err
	}
	return func(ctx context.Context) error { return n.GarantirPublicador(ctx, topico, membro) }, nil
}

// GarantirPublicador garante o vínculo pubsub.publisher do membro no tópico.
func (n *Nuvem) GarantirPublicador(ctx context.Context, topico, membro string) error {
	recurso := fmt.Sprintf("projects/%s/topics/%s", n.projeto, topico)
	politica, err := n.pubsub.TopicAdminClient.GetIamPolicy(ctx, &iampb.GetIamPolicyRequest{Resource: recurso})
	if err != nil {
		return err
	}
	for _, b := range politica.Bindings {
		if b.Role == "roles/pubsub.publisher" {
			if slices.Contains(b.Members, membro) {
				return nil
			}
			b.Members = append(b.Members, membro)
			_, err := n.pubsub.TopicAdminClient.SetIamPolicy(ctx, &iampb.SetIamPolicyRequest{Resource: recurso, Policy: politica})
			return err
		}
	}
	politica.Bindings = append(politica.Bindings, &iampb.Binding{Role: "roles/pubsub.publisher", Members: []string{membro}})
	_, err = n.pubsub.TopicAdminClient.SetIamPolicy(ctx, &iampb.SetIamPolicyRequest{Resource: recurso, Policy: politica})
	return err
}

// RemoverPapelProjeto retira o membro de um papel no projeto e devolve a
// função que restaura o vínculo.
func (n *Nuvem) RemoverPapelProjeto(ctx context.Context, papel, membro string) (func(context.Context) error, error) {
	politica, err := n.crm.Projects.GetIamPolicy(n.projeto, &cloudresourcemanager.GetIamPolicyRequest{}).Context(ctx).Do()
	if err != nil {
		return nil, err
	}
	removido := false
	for _, b := range politica.Bindings {
		if b.Role == papel && slices.Contains(b.Members, membro) {
			b.Members = slices.DeleteFunc(b.Members, func(m string) bool { return m == membro })
			removido = true
		}
	}
	if !removido {
		return nil, fmt.Errorf("%s não tem %s no projeto", membro, papel)
	}
	if _, err := n.crm.Projects.SetIamPolicy(n.projeto, &cloudresourcemanager.SetIamPolicyRequest{Policy: politica}).Context(ctx).Do(); err != nil {
		return nil, err
	}
	return func(ctx context.Context) error { return n.GarantirPapelProjeto(ctx, papel, membro) }, nil
}

// GarantirPapelProjeto garante o vínculo do membro ao papel no projeto.
func (n *Nuvem) GarantirPapelProjeto(ctx context.Context, papel, membro string) error {
	politica, err := n.crm.Projects.GetIamPolicy(n.projeto, &cloudresourcemanager.GetIamPolicyRequest{}).Context(ctx).Do()
	if err != nil {
		return err
	}
	for _, b := range politica.Bindings {
		if b.Role == papel {
			if slices.Contains(b.Members, membro) {
				return nil
			}
			b.Members = append(b.Members, membro)
			_, err := n.crm.Projects.SetIamPolicy(n.projeto, &cloudresourcemanager.SetIamPolicyRequest{Policy: politica}).Context(ctx).Do()
			return err
		}
	}
	politica.Bindings = append(politica.Bindings, &cloudresourcemanager.Binding{Role: papel, Members: []string{membro}})
	_, err = n.crm.Projects.SetIamPolicy(n.projeto, &cloudresourcemanager.SetIamPolicyRequest{Policy: politica}).Context(ctx).Do()
	return err
}
