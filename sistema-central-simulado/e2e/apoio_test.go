//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"cloud.google.com/go/pubsub/v2"

	"github.com/ThiagoCristovao/serverless-vending-validation/sistema-central-simulado/internal/armazenamento"
	"github.com/ThiagoCristovao/serverless-vending-validation/sistema-central-simulado/internal/consumidor"
	"github.com/ThiagoCristovao/serverless-vending-validation/sistema-central-simulado/internal/falhas"
	"github.com/ThiagoCristovao/serverless-vending-validation/sistema-central-simulado/internal/laboratorio"
)

// ambiente reúne o consumidor em processo (registro em arquivo temporário e
// API administrativa em porta efêmera), o cliente Pub/Sub e a sessão do operador.
type ambiente struct {
	cfg      laboratorio.Configuracao
	ctx      context.Context
	cancelar context.CancelFunc
	http     *http.Client
	gateway  *laboratorio.Gateway
	adesivo  *laboratorio.Adesivo
	uid      string
	cli      *pubsub.Client
	central  *laboratorio.ConsumidorLocal
	admin    *httptest.Server
	registro func(string)
}

var (
	amb          *ambiente
	motivoPular  string
	prepararUma  sync.Once
	erroPreparar error
)

func TestMain(m *testing.M) {
	if _, err := laboratorio.CarregarConfiguracao(); err != nil {
		motivoPular = err.Error()
	}
	codigo := m.Run()
	if amb != nil {
		amb.encerrar()
		est := amb.central.Processador.Estatisticas()
		fmt.Printf("\nresumo do consumidor: recebidas=%d registradas=%d duplicadas=%d rejeitadas=%d invalidas=%d confirmacoesPerdidas=%d\nregistro: %s\n",
			est.Recebidas, est.Registradas, est.Duplicadas, est.Rejeitadas, est.Invalidas, est.ConfirmacoesPerdidas, amb.central.Registro.Caminho())
	}
	os.Exit(codigo)
}

// exigirAmbiente prepara o ambiente uma vez; pula o teste se a configuração faltar.
func exigirAmbiente(t *testing.T) *ambiente {
	t.Helper()
	if motivoPular != "" {
		t.Skip("e2e não configurado: " + motivoPular)
	}
	prepararUma.Do(func() {
		cfg, _ := laboratorio.CarregarConfiguracao()
		amb, erroPreparar = prepararAmbiente(cfg)
	})
	if erroPreparar != nil {
		t.Fatalf("preparação do ambiente: %v", erroPreparar)
	}
	// As linhas do gateway vão para o teste corrente.
	amb.registro = func(l string) { t.Log(l) }
	return amb
}

func prepararAmbiente(cfg laboratorio.Configuracao) (*ambiente, error) {
	ctx, cancelar := context.WithCancel(context.Background())
	a := &ambiente{cfg: cfg, ctx: ctx, cancelar: cancelar, http: &http.Client{Timeout: 30 * time.Second}}

	adesivo, err := laboratorio.NovoAdesivo(cfg)
	if err != nil {
		return nil, err
	}
	a.adesivo = adesivo
	op, err := laboratorio.EntrarComoOperador(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("token do operador: %w", err)
	}
	a.uid = op.Uid
	a.gateway = laboratorio.NovoGateway(cfg, op, func(l string) {
		if a.registro != nil {
			a.registro(l)
		}
	})

	a.cli, err = pubsub.NewClient(ctx, cfg.Projeto)
	if err != nil {
		return nil, fmt.Errorf("cliente Pub/Sub: %w", err)
	}
	log := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	a.central, err = laboratorio.NovoConsumidorLocal(a.cli, cfg.Assinatura(), log)
	if err != nil {
		return nil, err
	}
	if err := a.central.Iniciar(ctx); err != nil {
		return nil, err
	}
	a.admin = httptest.NewServer(a.central.HandlerAdmin(ctx))
	return a, nil
}

func (a *ambiente) encerrar() {
	a.central.Encerrar()
	a.admin.Close()
	_ = a.cli.Close()
	a.cancelar()
}

// --- Gateway -------------------------------------------------------------------

func (a *ambiente) chamarGateway(t *testing.T, metodo, caminho string, corpo any, chaveIdempotencia string) (int, map[string]any) {
	t.Helper()
	resp, err := a.gateway.Chamar(a.ctx, metodo, caminho, corpo, chaveIdempotencia)
	if err != nil {
		t.Fatalf("%s %s: %v", metodo, caminho, err)
	}
	return resp.Status, resp.Corpo
}

type validacaoAberta struct {
	Id       string
	Resposta map[string]any
	Corpo    map[string]any
	Chave    string
}

// iniciarValidacao abre uma validação pelo gateway (libera sobras de execuções anteriores).
func (a *ambiente) iniciarValidacao(t *testing.T, maq string) validacaoAberta {
	t.Helper()
	v, err := a.gateway.IniciarValidacao(a.ctx, a.adesivo, maq)
	if err != nil {
		t.Fatal(err)
	}
	return validacaoAberta{Id: v.Id, Resposta: v.Resposta.Corpo, Corpo: v.Corpo, Chave: v.Chave}
}

func (a *ambiente) concluirValidacao(t *testing.T, id string) map[string]any {
	t.Helper()
	resp, err := a.gateway.ConcluirValidacao(a.ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	return resp.Corpo
}

// --- API administrativa do consumidor ------------------------------------------

func (a *ambiente) eventos(t *testing.T, idValidacao string) []armazenamento.Item {
	t.Helper()
	resp, err := a.http.Get(a.admin.URL + "/eventos?idValidacao=" + idValidacao)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var dados struct {
		Itens []armazenamento.Item `json:"itens"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&dados); err != nil {
		t.Fatal(err)
	}
	return dados.Itens
}

func (a *ambiente) estatisticas(t *testing.T) consumidor.Estatisticas {
	t.Helper()
	resp, err := a.http.Get(a.admin.URL + "/estatisticas")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var est consumidor.Estatisticas
	if err := json.NewDecoder(resp.Body).Decode(&est); err != nil {
		t.Fatal(err)
	}
	return est
}

func (a *ambiente) definirFalhas(t *testing.T, cfg falhas.Configuracao) {
	t.Helper()
	corpo, _ := json.Marshal(cfg)
	req, _ := http.NewRequest(http.MethodPut, a.admin.URL+"/admin/falhas", bytes.NewReader(corpo))
	resp, err := a.http.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("definir falhas: %d %s", resp.StatusCode, b)
	}
	t.Logf("%s interruptor de falhas: modo=%s idMaquina=%q", hora(), cfg.Modo, cfg.IdMaquina)
}

// --- Pub/Sub -------------------------------------------------------------------

// publicarCopia reenvia ao tópico uma mensagem já consumida, com o mesmo corpo,
// atributos e chave de ordenação: simula a entrega repetida do at-least-once.
func (a *ambiente) publicarCopia(t *testing.T, item armazenamento.Item) {
	t.Helper()
	pub := a.cli.Publisher(a.cfg.Topico())
	pub.EnableMessageOrdering = true
	defer pub.Stop()
	res := pub.Publish(a.ctx, &pubsub.Message{Data: item.Corpo, Attributes: item.Atributos, OrderingKey: item.IdMaquina})
	id, err := res.Get(a.ctx)
	if err != nil {
		t.Fatalf("publicar cópia: %v", err)
	}
	t.Logf("%s cópia de %s/%s publicada (mensagem %s)", hora(), item.IdValidacao, item.Tipo, id)
}

// drenarDlq confirma tudo o que estiver na assinatura de inspeção da DLQ.
func (a *ambiente) drenarDlq(t *testing.T) {
	t.Helper()
	ctx, cancelar := context.WithTimeout(a.ctx, 5*time.Second)
	defer cancelar()
	n := 0
	_ = a.cli.Subscriber(a.cfg.AssinaturaDlq()).Receive(ctx, func(_ context.Context, m *pubsub.Message) { m.Ack(); n++ })
	t.Logf("%s DLQ drenada: %d mensagem(ns) antiga(s) confirmada(s)", hora(), n)
}

// esperarNaDlq recebe da assinatura de inspeção até achar a validação.
func (a *ambiente) esperarNaDlq(t *testing.T, idValidacao string, prazo time.Duration) *pubsub.Message {
	t.Helper()
	ctx, cancelar := context.WithTimeout(a.ctx, prazo)
	defer cancelar()
	var achada *pubsub.Message
	_ = a.cli.Subscriber(a.cfg.AssinaturaDlq()).Receive(ctx, func(_ context.Context, m *pubsub.Message) {
		m.Ack()
		if m.Attributes["idValidacao"] == idValidacao {
			achada = m
			cancelar()
		}
	})
	if achada == nil {
		t.Fatalf("mensagem de %s não chegou à DLQ em %s", idValidacao, prazo)
	}
	return achada
}

// --- utilidades ----------------------------------------------------------------

func esperar(t *testing.T, descricao string, prazo time.Duration, cond func() bool) {
	t.Helper()
	levou, ok := laboratorio.Esperar(context.Background(), prazo, 500*time.Millisecond, cond)
	if !ok {
		t.Fatalf("tempo esgotado (%s) esperando: %s", prazo, descricao)
	}
	t.Logf("%s %s (após %s)", hora(), descricao, levou.Round(time.Millisecond))
}

func hora() string { return laboratorio.Hora() }

func uuid4() string { return laboratorio.NovoUuid() }
