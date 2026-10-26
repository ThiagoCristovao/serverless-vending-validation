package consumidor

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"cloud.google.com/go/pubsub/v2"
)

// Estado do consumo da assinatura.
type Estado struct {
	Assinatura string     `json:"assinatura"`
	Consumindo bool       `json:"consumindo"`
	IniciadoEm *time.Time `json:"iniciadoEm,omitempty"`
	PausadoEm  *time.Time `json:"pausadoEm,omitempty"`
	UltimoErro string     `json:"ultimoErro,omitempty"`
}

// Consumidor recebe mensagens da assinatura e aplica a decisão do Processador.
// Pode ser pausado e retomado (indisponibilidade deliberada, CE-15).
type Consumidor struct {
	sub  *pubsub.Subscriber
	proc *Processador
	log  *slog.Logger

	mu        sync.Mutex
	cancelar  context.CancelFunc
	terminado chan struct{}
	estado    Estado
}

// Novo prepara o consumidor da assinatura (nome curto ou completo).
func Novo(cli *pubsub.Client, assinatura string, proc *Processador, log *slog.Logger) *Consumidor {
	if log == nil {
		log = slog.Default()
	}
	sub := cli.Subscriber(assinatura)
	// Uma goroutine de recepção basta para o volume deste trabalho e mantém a
	// ordem por chave simples de observar nos logs.
	sub.ReceiveSettings.NumGoroutines = 1
	sub.ReceiveSettings.MaxOutstandingMessages = 10
	return &Consumidor{sub: sub, proc: proc, log: log, estado: Estado{Assinatura: assinatura}}
}

// Iniciar começa (ou retoma) a recepção em segundo plano. Sem efeito se já
// estiver consumindo.
func (c *Consumidor) Iniciar(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.estado.Consumindo {
		return nil
	}
	ctxRecepcao, cancelar := context.WithCancel(ctx)
	terminado := make(chan struct{})
	c.cancelar, c.terminado = cancelar, terminado
	agora := time.Now().UTC()
	c.estado.Consumindo, c.estado.IniciadoEm, c.estado.PausadoEm, c.estado.UltimoErro = true, &agora, nil, ""
	c.log.Info("consumo iniciado", "assinatura", c.estado.Assinatura)

	go func() {
		defer close(terminado)
		err := c.sub.Receive(ctxRecepcao, c.receber)
		c.mu.Lock()
		defer c.mu.Unlock()
		c.estado.Consumindo = false
		if err != nil && !errors.Is(err, context.Canceled) {
			c.estado.UltimoErro = err.Error()
			c.log.Error("recepção encerrada com erro", "assinatura", c.estado.Assinatura, "erro", err.Error())
		}
	}()
	return nil
}

func (c *Consumidor) receber(_ context.Context, m *pubsub.Message) {
	tentativa := 0
	if m.DeliveryAttempt != nil {
		tentativa = *m.DeliveryAttempt
	}
	d := c.proc.Tratar(Mensagem{
		Id:             m.ID,
		Dados:          m.Data,
		Atributos:      m.Attributes,
		ChaveOrdenacao: m.OrderingKey,
		PublicadaEm:    m.PublishTime,
		Tentativa:      tentativa,
	})
	if d.Confirmar {
		m.Ack()
	} else {
		m.Nack()
	}
}

// Pausar interrompe a recepção e espera as mensagens em andamento terminarem.
// As não confirmadas voltam à assinatura.
func (c *Consumidor) Pausar() {
	c.mu.Lock()
	cancelar, terminado, consumindo := c.cancelar, c.terminado, c.estado.Consumindo
	c.mu.Unlock()
	if !consumindo || cancelar == nil {
		return
	}
	cancelar()
	<-terminado
	c.mu.Lock()
	agora := time.Now().UTC()
	c.estado.PausadoEm = &agora
	c.mu.Unlock()
	c.log.Info("consumo pausado", "assinatura", c.estado.Assinatura)
}

// Estado devolve uma cópia do estado corrente.
func (c *Consumidor) Estado() Estado {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.estado
}
