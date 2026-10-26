// Package consumidor liga a assinatura do Pub/Sub ao registro idempotente.
// O Processador decide o destino de cada mensagem sem conhecer o cliente do
// Pub/Sub (testável sem rede); o Consumidor faz a recepção e aplica a decisão.
package consumidor

import (
	"encoding/json"
	"log/slog"
	"maps"
	"sync"
	"time"

	"github.com/ThiagoCristovao/serverless-vending-validation/sistema-central-simulado/internal/armazenamento"
	"github.com/ThiagoCristovao/serverless-vending-validation/sistema-central-simulado/internal/eventos"
	"github.com/ThiagoCristovao/serverless-vending-validation/sistema-central-simulado/internal/falhas"
)

// Mensagem é uma entrega do Pub/Sub, desacoplada do cliente.
type Mensagem struct {
	Id             string
	Dados          []byte
	Atributos      map[string]string
	ChaveOrdenacao string
	PublicadaEm    time.Time
	Tentativa      int
}

// Resultado do processamento de uma mensagem.
type Resultado string

const (
	ResultadoRegistrada         Resultado = "registrada"
	ResultadoDuplicada          Resultado = "duplicada"
	ResultadoRejeitada          Resultado = "rejeitada"
	ResultadoInvalida           Resultado = "invalida"
	ResultadoConfirmacaoPerdida Resultado = "confirmacao-perdida"
	ResultadoErroArmazenamento  Resultado = "erro-armazenamento"
)

// Decisao diz se a mensagem deve ser confirmada (ack) ou rejeitada (nack).
type Decisao struct {
	Confirmar bool
	Resultado Resultado
	Evento    *eventos.Evento
	Erro      error
}

// Estatisticas acumuladas desde o início do processo.
type Estatisticas struct {
	Recebidas            int64      `json:"recebidas"`
	Registradas          int64      `json:"registradas"`
	Duplicadas           int64      `json:"duplicadas"`
	Rejeitadas           int64      `json:"rejeitadas"`
	Invalidas            int64      `json:"invalidas"`
	ConfirmacoesPerdidas int64      `json:"confirmacoesPerdidas"`
	ErrosArmazenamento   int64      `json:"errosArmazenamento"`
	UltimaRecebidaEm     *time.Time `json:"ultimaRecebidaEm,omitempty"`
}

// Armazenador é o que o processador precisa do registro.
type Armazenador interface {
	Registrar(item armazenamento.Item) (novo bool, err error)
}

// Processador aplica as regras do consumidor a uma mensagem.
type Processador struct {
	registro    Armazenador
	interruptor *falhas.Interruptor
	log         *slog.Logger
	// Agora permite fixar o relógio nos testes.
	Agora func() time.Time

	mu  sync.Mutex
	est Estatisticas
}

// NovoProcessador cria o processador.
func NovoProcessador(registro Armazenador, interruptor *falhas.Interruptor, log *slog.Logger) *Processador {
	if log == nil {
		log = slog.Default()
	}
	if interruptor == nil {
		interruptor = falhas.Novo()
	}
	return &Processador{registro: registro, interruptor: interruptor, log: log, Agora: time.Now}
}

// Tratar decide o destino de uma mensagem:
//
//  1. corpo fora do contrato → rejeitada (após o limite de tentativas, vai à DLQ);
//  2. interruptor em "rejeitar" para a máquina → rejeitada antes de persistir;
//  3. persistida; se já existia (idValidacao, tipo) → duplicata, confirmada;
//  4. interruptor em "confirmacao-perdida" → persistida mas não confirmada;
//  5. caso normal → confirmada após persistir.
func (p *Processador) Tratar(m Mensagem) Decisao {
	recebida := p.Agora().UTC()
	p.contar(func(e *Estatisticas) { e.Recebidas++; e.UltimaRecebidaEm = &recebida })
	campos := []any{"idMensagem", m.Id, "tentativa", m.Tentativa, "chaveOrdenacao", m.ChaveOrdenacao}

	ev, err := eventos.Decodificar(m.Dados)
	if err != nil {
		p.contar(func(e *Estatisticas) { e.Invalidas++ })
		p.log.Warn("mensagem fora do contrato rejeitada", append(campos, "erro", err.Error())...)
		return Decisao{Resultado: ResultadoInvalida, Erro: err}
	}
	campos = append(campos, "idValidacao", ev.IdValidacao, "tipo", ev.Tipo, "idMaquina", ev.Maquina.Id)

	if p.interruptor.Rejeita(ev.Maquina.Id) {
		p.contar(func(e *Estatisticas) { e.Rejeitadas++ })
		p.log.Warn("mensagem rejeitada pelo interruptor de falhas", campos...)
		return Decisao{Resultado: ResultadoRejeitada, Evento: &ev}
	}

	item := armazenamento.Item{
		IdValidacao: ev.IdValidacao,
		Tipo:        ev.Tipo,
		IdMaquina:   ev.Maquina.Id,
		IdMensagem:  m.Id,
		PublicadaEm: m.PublicadaEm.UTC(),
		RecebidaEm:  recebida,
		Tentativa:   m.Tentativa,
		Atributos:   maps.Clone(m.Atributos),
		Evento:      ev,
		Corpo:       json.RawMessage(append([]byte(nil), m.Dados...)),
	}
	novo, err := p.registro.Registrar(item)
	if err != nil {
		p.contar(func(e *Estatisticas) { e.ErrosArmazenamento++ })
		p.log.Error("falha ao persistir o evento", append(campos, "erro", err.Error())...)
		return Decisao{Resultado: ResultadoErroArmazenamento, Evento: &ev, Erro: err}
	}
	if !novo {
		p.contar(func(e *Estatisticas) { e.Duplicadas++ })
		p.log.Info("evento duplicado ignorado", campos...)
		return Decisao{Confirmar: true, Resultado: ResultadoDuplicada, Evento: &ev}
	}
	p.contar(func(e *Estatisticas) { e.Registradas++ })
	if p.interruptor.PerdeConfirmacao(ev.Maquina.Id) {
		p.contar(func(e *Estatisticas) { e.ConfirmacoesPerdidas++ })
		p.log.Warn("evento registrado sem confirmação (interruptor de falhas)", campos...)
		return Decisao{Resultado: ResultadoConfirmacaoPerdida, Evento: &ev}
	}
	p.log.Info("evento registrado", campos...)
	return Decisao{Confirmar: true, Resultado: ResultadoRegistrada, Evento: &ev}
}

func (p *Processador) contar(f func(*Estatisticas)) {
	p.mu.Lock()
	f(&p.est)
	p.mu.Unlock()
}

// Estatisticas devolve uma cópia dos contadores.
func (p *Processador) Estatisticas() Estatisticas {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.est
}
