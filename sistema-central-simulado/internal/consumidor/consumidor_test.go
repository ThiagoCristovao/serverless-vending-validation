package consumidor_test

import (
	"context"
	"testing"
	"time"

	"cloud.google.com/go/pubsub/v2"
	"cloud.google.com/go/pubsub/v2/apiv1/pubsubpb"
	"cloud.google.com/go/pubsub/v2/pstest"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/ThiagoCristovao/serverless-vending-validation/sistema-central-simulado/internal/armazenamento"
	"github.com/ThiagoCristovao/serverless-vending-validation/sistema-central-simulado/internal/consumidor"
	"github.com/ThiagoCristovao/serverless-vending-validation/sistema-central-simulado/internal/eventos"
	"github.com/ThiagoCristovao/serverless-vending-validation/sistema-central-simulado/internal/falhas"
)

// Ambiente com o servidor Pub/Sub de testes em processo (pstest): tópico,
// DLQ e assinatura ordenada com política de mensagens mortas em 5 tentativas,
// como o módulo Terraform `pubsub`.
type ambiente struct {
	cli      *pubsub.Client
	pub      *pubsub.Publisher
	registro *armazenamento.Registro
	inter    *falhas.Interruptor
	proc     *consumidor.Processador
	cons     *consumidor.Consumidor
}

const (
	projeto     = "svv-teste"
	topico      = "projects/svv-teste/topics/validacoes"
	topicoDlq   = "projects/svv-teste/topics/validacoes-dlq"
	assinatura  = "projects/svv-teste/subscriptions/sistema-central"
	dlqInspecao = "projects/svv-teste/subscriptions/validacoes-dlq-inspecao"
)

func novoAmbiente(t *testing.T) *ambiente {
	t.Helper()
	ctx := context.Background()
	srv := pstest.NewServer()
	t.Cleanup(func() { _ = srv.Close() })
	conn, err := grpc.NewClient(srv.Addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	cli, err := pubsub.NewClient(ctx, projeto, option.WithGRPCConn(conn))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cli.Close() })
	for _, nome := range []string{topico, topicoDlq} {
		if _, err := cli.TopicAdminClient.CreateTopic(ctx, &pubsubpb.Topic{Name: nome}); err != nil {
			t.Fatal(err)
		}
	}
	_, err = cli.SubscriptionAdminClient.CreateSubscription(ctx, &pubsubpb.Subscription{
		Name: assinatura, Topic: topico, AckDeadlineSeconds: 10, EnableMessageOrdering: true,
		DeadLetterPolicy: &pubsubpb.DeadLetterPolicy{DeadLetterTopic: topicoDlq, MaxDeliveryAttempts: 5},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cli.SubscriptionAdminClient.CreateSubscription(ctx, &pubsubpb.Subscription{Name: dlqInspecao, Topic: topicoDlq, AckDeadlineSeconds: 10}); err != nil {
		t.Fatal(err)
	}
	pub := cli.Publisher(topico)
	pub.EnableMessageOrdering = true
	t.Cleanup(pub.Stop)

	registro, _ := armazenamento.Abrir("")
	inter := falhas.Novo()
	proc := consumidor.NovoProcessador(registro, inter, silencioso)
	cons := consumidor.Novo(cli, assinatura, proc, silencioso)
	t.Cleanup(cons.Pausar)
	return &ambiente{cli: cli, pub: pub, registro: registro, inter: inter, proc: proc, cons: cons}
}

func (a *ambiente) publicar(t *testing.T, tipo, id, maq string) {
	t.Helper()
	res := a.pub.Publish(context.Background(), &pubsub.Message{Data: corpo(tipo, id, maq), Attributes: map[string]string{"tipo": tipo, "idValidacao": id, "idMaquina": maq, "versao": "1"}, OrderingKey: maq})
	if _, err := res.Get(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func (a *ambiente) publicarBruto(t *testing.T, dados []byte, chave string) {
	t.Helper()
	res := a.pub.Publish(context.Background(), &pubsub.Message{Data: dados, OrderingKey: chave})
	if _, err := res.Get(context.Background()); err != nil {
		t.Fatal(err)
	}
}

// esperarNaDlq recebe da assinatura de inspeção até achar a mensagem com o
// atributo idValidacao (ou corpo) esperado.
func (a *ambiente) esperarNaDlq(t *testing.T, aceita func(*pubsub.Message) bool) *pubsub.Message {
	t.Helper()
	ctx, cancelar := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelar()
	var achada *pubsub.Message
	err := a.cli.Subscriber(dlqInspecao).Receive(ctx, func(_ context.Context, m *pubsub.Message) {
		m.Ack()
		if aceita(m) {
			achada = m
			cancelar()
		}
	})
	if achada == nil {
		t.Fatalf("mensagem não chegou à DLQ (%v)", err)
	}
	return achada
}

func esperar(t *testing.T, descricao string, cond func() bool) {
	t.Helper()
	prazo := time.Now().Add(15 * time.Second)
	for time.Now().Before(prazo) {
		if cond() {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("tempo esgotado esperando: %s", descricao)
}

func TestConsumidorRegistraEmOrdemEIgnoraDuplicata(t *testing.T) {
	a := novoAmbiente(t)
	a.publicar(t, eventos.TipoValidacaoIniciada, idA, "VM-1")
	a.publicar(t, eventos.TipoValidacaoConcluida, idA, "VM-1")
	if err := a.cons.Iniciar(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !a.cons.Estado().Consumindo {
		t.Fatal("deveria estar consumindo")
	}
	esperar(t, "dois eventos registrados", func() bool { return a.registro.Total() == 2 })
	itens := a.registro.Listar(armazenamento.Filtro{IdValidacao: idA})
	if itens[0].Tipo != eventos.TipoValidacaoIniciada || itens[1].Tipo != eventos.TipoValidacaoConcluida || itens[0].IdMensagem == "" || itens[0].Tentativa != 1 {
		t.Fatalf("ordem/metadados: %+v", itens)
	}

	// Entrega repetida (at-least-once): ignorada e confirmada.
	a.publicar(t, eventos.TipoValidacaoIniciada, idA, "VM-1")
	esperar(t, "duplicata contada", func() bool { return a.proc.Estatisticas().Duplicadas == 1 })
	if a.registro.Total() != 2 {
		t.Fatalf("duplicata registrada: total %d", a.registro.Total())
	}

	a.cons.Pausar()
	if e := a.cons.Estado(); e.Consumindo || e.PausadoEm == nil {
		t.Fatalf("estado após pausa: %+v", e)
	}
	// Retomar volta a consumir.
	a.publicar(t, eventos.TipoValidacaoIniciada, idB, "VM-1")
	_ = a.cons.Iniciar(context.Background())
	esperar(t, "evento após retomar", func() bool { return a.registro.Total() == 3 })
}

func TestFalhasSucessivasVaoParaDlqSemBloquearOutrasMaquinas(t *testing.T) {
	a := novoAmbiente(t)
	_ = a.inter.Definir(falhas.Configuracao{Modo: falhas.ModoRejeitar, IdMaquina: "VM-2"})
	a.publicar(t, eventos.TipoValidacaoIniciada, idB, "VM-2")
	a.publicar(t, eventos.TipoValidacaoIniciada, idA, "VM-1")
	_ = a.cons.Iniciar(context.Background())

	esperar(t, "VM-1 registrada enquanto VM-2 falha", func() bool { return a.registro.Total() == 1 })
	m := a.esperarNaDlq(t, func(m *pubsub.Message) bool { return m.Attributes["idValidacao"] == idB })
	if m.Attributes["tipo"] != eventos.TipoValidacaoIniciada {
		t.Fatalf("mensagem na DLQ: %+v", m.Attributes)
	}
	if est := a.proc.Estatisticas(); est.Rejeitadas != 5 || est.Registradas != 1 {
		t.Fatalf("estatísticas: %+v", est)
	}
	if len(a.registro.Listar(armazenamento.Filtro{IdMaquina: "VM-2"})) != 0 {
		t.Fatal("mensagem rejeitada não deveria ter sido persistida")
	}

	// Com o interruptor desligado, o consumo segue. (O pstest não remove da
	// assinatura ordenada a mensagem que encaminhou à DLQ, o que prende a chave
	// VM-2 só no servidor falso; a retomada da mesma máquina após a DLQ é
	// verificada contra o Pub/Sub real na suíte e2e.)
	_ = a.inter.Definir(falhas.Configuracao{Modo: falhas.ModoNenhum})
	a.publicar(t, eventos.TipoValidacaoConcluida, idA, "VM-1")
	esperar(t, "VM-1 segue consumida após a DLQ", func() bool {
		return len(a.registro.Listar(armazenamento.Filtro{IdMaquina: "VM-1"})) == 2
	})
}

func TestMensagemForaDoContratoVaiParaDlq(t *testing.T) {
	a := novoAmbiente(t)
	a.publicarBruto(t, []byte(`{"tipo":"validacao.cancelada"}`), "VM-9")
	_ = a.cons.Iniciar(context.Background())
	m := a.esperarNaDlq(t, func(m *pubsub.Message) bool { return string(m.Data) == `{"tipo":"validacao.cancelada"}` })
	if m == nil || a.proc.Estatisticas().Invalidas != 5 || a.registro.Total() != 0 {
		t.Fatalf("estatísticas: %+v", a.proc.Estatisticas())
	}
}

func TestConfirmacaoPerdidaGeraReentregaReconhecida(t *testing.T) {
	a := novoAmbiente(t)
	_ = a.inter.Definir(falhas.Configuracao{Modo: falhas.ModoConfirmacaoPerdida, IdMaquina: "VM-3"})
	a.publicar(t, eventos.TipoValidacaoIniciada, idA, "VM-3")
	_ = a.cons.Iniciar(context.Background())
	esperar(t, "reentrega reconhecida como duplicata", func() bool {
		est := a.proc.Estatisticas()
		return est.ConfirmacoesPerdidas == 1 && est.Duplicadas == 1
	})
	if a.registro.Total() != 1 {
		t.Fatalf("total %d", a.registro.Total())
	}
}
