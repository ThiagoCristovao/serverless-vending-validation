// Package falhas implementa o interruptor de indisponibilidade do sistema
// central simulado: faz o consumidor falhar de propósito para exercitar
// retentativa, fila de mensagens mortas (CE-16) e duplicatas por confirmação
// perdida (CE-17). Controlado pela API administrativa e usado pelas Sprints 8 e 9.
package falhas

import (
	"fmt"
	"sync"
)

// Modo de falha.
type Modo string

const (
	// ModoNenhum: processamento normal.
	ModoNenhum Modo = "nenhum"
	// ModoRejeitar: a mensagem é rejeitada (nack) antes de ser persistida; o
	// Pub/Sub reentrega com espera exponencial e, após o limite, encaminha à DLQ.
	ModoRejeitar Modo = "rejeitar"
	// ModoConfirmacaoPerdida: o evento é persistido, mas a confirmação não é
	// enviada (nack após persistir), como se o ack se perdesse na rede. A
	// reentrega deve ser reconhecida como duplicata.
	ModoConfirmacaoPerdida Modo = "confirmacao-perdida"
)

// Configuracao do interruptor. IdMaquina vazio aplica o modo a todas as máquinas.
type Configuracao struct {
	Modo      Modo   `json:"modo"`
	IdMaquina string `json:"idMaquina,omitempty"`
}

// Validar confere o modo.
func (c Configuracao) Validar() error {
	switch c.Modo {
	case ModoNenhum, ModoRejeitar, ModoConfirmacaoPerdida:
		return nil
	}
	return fmt.Errorf("modo desconhecido %q (use nenhum, rejeitar ou confirmacao-perdida)", c.Modo)
}

// Interruptor guarda a configuração corrente de forma segura entre goroutines.
type Interruptor struct {
	mu  sync.RWMutex
	cfg Configuracao
}

// Novo cria um interruptor desligado.
func Novo() *Interruptor { return &Interruptor{cfg: Configuracao{Modo: ModoNenhum}} }

// Definir substitui a configuração.
func (i *Interruptor) Definir(c Configuracao) error {
	if err := c.Validar(); err != nil {
		return err
	}
	if c.Modo == ModoNenhum {
		c.IdMaquina = ""
	}
	i.mu.Lock()
	i.cfg = c
	i.mu.Unlock()
	return nil
}

// Atual devolve a configuração corrente.
func (i *Interruptor) Atual() Configuracao {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.cfg
}

func (i *Interruptor) aplica(idMaquina string, modo Modo) bool {
	c := i.Atual()
	return c.Modo == modo && (c.IdMaquina == "" || c.IdMaquina == idMaquina)
}

// Rejeita informa se a mensagem da máquina deve ser rejeitada antes de persistir.
func (i *Interruptor) Rejeita(idMaquina string) bool { return i.aplica(idMaquina, ModoRejeitar) }

// PerdeConfirmacao informa se a confirmação da máquina deve ser omitida após persistir.
func (i *Interruptor) PerdeConfirmacao(idMaquina string) bool {
	return i.aplica(idMaquina, ModoConfirmacaoPerdida)
}
