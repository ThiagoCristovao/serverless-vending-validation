//go:build e2e

// Suíte de ponta a ponta da Sprint 8: exercita o ecossistema implantado em
// svv-dev, do gateway ao consumo pelo sistema central simulado, que roda em
// processo durante a suíte. Cobre o fluxo principal, a idempotência do serviço
// (CE-02, CE-11), mensagens duplicadas (CE-17, RF-24) e falhas sucessivas com
// encaminhamento à fila de mensagens mortas (CE-16, RF-25, RNF-02).
//
//	make e2e            # preenche o ambiente a partir do Terraform e de chaves/
//	go test -tags e2e ./e2e/ -v -timeout 25m
package e2e

import (
	"net/http"
	"testing"
	"time"

	"github.com/ThiagoCristovao/serverless-vending-validation/sistema-central-simulado/internal/eventos"
	"github.com/ThiagoCristovao/serverless-vending-validation/sistema-central-simulado/internal/falhas"
)

const (
	prazoConsumo = 90 * time.Second
	// Reentrega após nack respeita a espera mínima de 10 s da assinatura.
	prazoReentrega = 3 * time.Minute
)

func TestFluxoPrincipalAteOConsumo(t *testing.T) {
	a := exigirAmbiente(t)
	maq := a.cfg.MaquinaA

	v := a.iniciarValidacao(t, maq)
	contrassenha := v.Resposta["contrassenha"]
	t.Logf("validação %s aberta em %s; contrassenha e próximo código recebidos", v.Id, maq)

	// CE-02: reenvio com a mesma Idempotency-Key devolve a mesma validação.
	status, repetida := a.chamarGateway(t, http.MethodPost, "/validacoes", v.Corpo, v.Chave)
	if status != http.StatusCreated || repetida["idValidacao"] != v.Id || repetida["contrassenha"] != contrassenha {
		t.Fatalf("reenvio idempotente: %d %v", status, repetida)
	}

	// CE-11: nova leitura com a validação aberta é recusada com o id existente.
	status, conflito := a.chamarGateway(t, http.MethodPost, "/validacoes", v.Corpo, uuid4())
	if status != http.StatusConflict || conflito["codigo"] != "validacao_em_andamento" || conflito["idValidacao"] != v.Id {
		t.Fatalf("validação em andamento: %d %v", status, conflito)
	}

	conclusao := a.concluirValidacao(t, v.Id)
	if conclusao["codigoRotacionado"] != true {
		t.Fatalf("conclusão: %v", conclusao)
	}

	esperar(t, "iniciada e concluída consumidas", prazoConsumo, func() bool { return len(a.eventos(t, v.Id)) == 2 })
	itens := a.eventos(t, v.Id)
	iniciada, concluida := itens[0], itens[1]
	if iniciada.Tipo != eventos.TipoValidacaoIniciada || concluida.Tipo != eventos.TipoValidacaoConcluida {
		t.Fatalf("ordem dos eventos: %s, %s", iniciada.Tipo, concluida.Tipo)
	}
	if iniciada.Evento.Maquina.Id != maq || iniciada.Evento.Operador.Id != a.uid || iniciada.Evento.Validacao.ExpiraEm == nil {
		t.Fatalf("iniciada: %+v", iniciada.Evento)
	}
	cv := concluida.Evento.Validacao
	if cv.Leituras == nil || cv.Leituras.UnidadesVendidas != 7 || cv.CodigoRotacionado == nil || !*cv.CodigoRotacionado || cv.ConcluidaEm == nil {
		t.Fatalf("concluída: %+v", cv)
	}
	if concluida.RecebidaEm.Before(iniciada.RecebidaEm) {
		t.Fatalf("concluída recebida antes da iniciada: %s < %s", concluida.RecebidaEm, iniciada.RecebidaEm)
	}
	if iniciada.Atributos["idMaquina"] != maq || concluida.Atributos["tipo"] != eventos.TipoValidacaoConcluida {
		t.Fatalf("atributos: %v %v", iniciada.Atributos, concluida.Atributos)
	}
	t.Logf("eventos de %s: iniciada (mensagem %s) e concluída (mensagem %s), cada um uma única vez apesar do reenvio idempotente", v.Id, iniciada.IdMensagem, concluida.IdMensagem)
}

func TestMensagemDuplicadaEhIgnorada(t *testing.T) {
	a := exigirAmbiente(t)
	maq := a.cfg.MaquinaA

	// (a) Entrega repetida: a mesma mensagem publicada de novo no tópico.
	v := a.iniciarValidacao(t, maq)
	a.concluirValidacao(t, v.Id)
	esperar(t, "eventos consumidos", prazoConsumo, func() bool { return len(a.eventos(t, v.Id)) == 2 })
	antes := a.estatisticas(t)
	a.publicarCopia(t, a.eventos(t, v.Id)[1])
	esperar(t, "cópia reconhecida como duplicata", prazoConsumo, func() bool { return a.estatisticas(t).Duplicadas == antes.Duplicadas+1 })
	if n := len(a.eventos(t, v.Id)); n != 2 {
		t.Fatalf("duplicata gerou registro: %d eventos", n)
	}

	// (b) Confirmação perdida: o consumidor persiste, não confirma, e a
	// reentrega do Pub/Sub é reconhecida pela chave (idValidacao, tipo).
	a.definirFalhas(t, falhas.Configuracao{Modo: falhas.ModoConfirmacaoPerdida, IdMaquina: maq})
	t.Cleanup(func() { a.definirFalhas(t, falhas.Configuracao{Modo: falhas.ModoNenhum}) })
	v2 := a.iniciarValidacao(t, maq)
	esperar(t, "iniciada persistida sem confirmação", prazoConsumo, func() bool { return a.estatisticas(t).ConfirmacoesPerdidas == antes.ConfirmacoesPerdidas+1 })
	a.definirFalhas(t, falhas.Configuracao{Modo: falhas.ModoNenhum})
	esperar(t, "reentrega reconhecida como duplicata", prazoReentrega, func() bool { return a.estatisticas(t).Duplicadas == antes.Duplicadas+2 })
	if n := len(a.eventos(t, v2.Id)); n != 1 {
		t.Fatalf("reentrega gerou registro: %d eventos", n)
	}
	a.concluirValidacao(t, v2.Id)
	esperar(t, "concluída consumida após a reentrega", prazoConsumo, func() bool { return len(a.eventos(t, v2.Id)) == 2 })
}

func TestFalhasSucessivasEncaminhamParaDlq(t *testing.T) {
	a := exigirAmbiente(t)
	if testing.Short() {
		t.Skip("cenário de DLQ leva alguns minutos (espera exponencial entre reentregas)")
	}
	maqFalha, maqNormal := a.cfg.MaquinaB, a.cfg.MaquinaA

	a.drenarDlq(t)
	a.definirFalhas(t, falhas.Configuracao{Modo: falhas.ModoRejeitar, IdMaquina: maqFalha})
	t.Cleanup(func() { a.definirFalhas(t, falhas.Configuracao{Modo: falhas.ModoNenhum}) })
	antes := a.estatisticas(t)
	inicio := time.Now()

	vFalha := a.iniciarValidacao(t, maqFalha)
	t.Logf("validação %s em %s: o consumidor rejeitará a mensagem iniciada", vFalha.Id, maqFalha)

	// RNF-02: enquanto uma máquina falha, as demais seguem sendo consumidas.
	vNormal := a.iniciarValidacao(t, maqNormal)
	a.concluirValidacao(t, vNormal.Id)
	esperar(t, "outra máquina consumida durante as falhas", prazoConsumo, func() bool { return len(a.eventos(t, vNormal.Id)) == 2 })
	esperar(t, "primeira rejeição registrada", prazoConsumo, func() bool { return a.estatisticas(t).Rejeitadas > antes.Rejeitadas })

	morta := a.esperarNaDlq(t, vFalha.Id, a.cfg.PrazoDlq)
	tentativas := morta.Attributes["CloudPubSubDeadLetterSourceDeliveryCount"]
	t.Logf("%s mensagem %s/%s na DLQ após %s; tentativas na origem=%s; assinatura de origem=%s",
		hora(), vFalha.Id, morta.Attributes["tipo"], time.Since(inicio).Round(time.Second), tentativas, morta.Attributes["CloudPubSubDeadLetterSourceSubscription"])
	if morta.Attributes["tipo"] != eventos.TipoValidacaoIniciada || morta.OrderingKey != maqFalha {
		t.Fatalf("mensagem na DLQ: atributos=%v chave=%q", morta.Attributes, morta.OrderingKey)
	}
	if rejeitadas := a.estatisticas(t).Rejeitadas - antes.Rejeitadas; rejeitadas < 5 {
		t.Fatalf("esperava ao menos 5 rejeições antes da DLQ, houve %d", rejeitadas)
	}
	if n := len(a.eventos(t, vFalha.Id)); n != 0 {
		t.Fatalf("mensagem rejeitada foi registrada: %d eventos", n)
	}

	// Falha sanada: a máquina volta a ser consumida (a concluída não depende da
	// iniciada perdida, que fica na DLQ para tratamento manual).
	a.definirFalhas(t, falhas.Configuracao{Modo: falhas.ModoNenhum})
	a.concluirValidacao(t, vFalha.Id)
	esperar(t, "concluída consumida após sanar a falha", prazoConsumo, func() bool {
		itens := a.eventos(t, vFalha.Id)
		return len(itens) == 1 && itens[0].Tipo == eventos.TipoValidacaoConcluida
	})
}
