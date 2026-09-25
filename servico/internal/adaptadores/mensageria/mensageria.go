// Package mensageria publica os eventos de validação no Pub/Sub com chave de
// ordenação por máquina (docs/arquitetura.md, seção 6).
package mensageria

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"cloud.google.com/go/pubsub/v2"

	"github.com/ThiagoCristovao/serverless-vending-validation/servico/internal/dominio"
)

// Publicador implementa portas.PublicadorEventos sobre um tópico.
type Publicador struct {
	pub         *pubsub.Publisher
	tempoLimite time.Duration
}

// Novo prepara o publicador do tópico dado, com ordenação de mensagens ligada.
// tempoLimite curto (2 s por padrão) evita que uma instabilidade do Pub/Sub
// segure a resposta ao operador; a publicação fica pendente para reconciliação.
func Novo(cli *pubsub.Client, topicoID string, tempoLimite time.Duration) *Publicador {
	p := cli.Publisher(topicoID)
	p.EnableMessageOrdering = true
	if tempoLimite <= 0 {
		tempoLimite = 2 * time.Second
	}
	return &Publicador{pub: p, tempoLimite: tempoLimite}
}

// Publicar implementa portas.PublicadorEventos: uma tentativa e uma
// retentativa dentro do tempo limite.
func (p *Publicador) Publicar(ctx context.Context, evento dominio.Evento) error {
	corpo, err := json.Marshal(evento)
	if err != nil {
		return fmt.Errorf("serializar evento: %w", err)
	}
	var ultimo error
	for tentativa := 0; tentativa < 2; tentativa++ {
		ctxTentativa, cancelar := context.WithTimeout(ctx, p.tempoLimite)
		resultado := p.pub.Publish(ctxTentativa, &pubsub.Message{
			Data:        corpo,
			Attributes:  evento.Atributos(),
			OrderingKey: evento.ChaveOrdenacao(),
		})
		_, ultimo = resultado.Get(ctxTentativa)
		cancelar()
		if ultimo == nil {
			return nil
		}
		// Com ordenação ligada, uma falha pausa a chave; é preciso retomá-la.
		p.pub.ResumePublish(evento.ChaveOrdenacao())
	}
	return fmt.Errorf("publicar %s de %s: %w", evento.Tipo, evento.IdValidacao, ultimo)
}

// Encerrar descarrega mensagens pendentes e libera o publicador.
func (p *Publicador) Encerrar() { p.pub.Stop() }
