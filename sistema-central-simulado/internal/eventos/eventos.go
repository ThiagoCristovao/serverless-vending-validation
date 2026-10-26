// Package eventos descreve o contrato dos eventos que o serviço de validação
// publica no Pub/Sub (docs/modelo-dados.md, seção 6), do ponto de vista do
// sistema central. O módulo depende do contrato publicado, não do código do
// serviço (ADR-0013): campos desconhecidos são tolerados, campos obrigatórios
// e a versão são conferidos.
package eventos

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"time"
)

// Tipos de evento conhecidos.
const (
	TipoValidacaoIniciada  = "validacao.iniciada"
	TipoValidacaoConcluida = "validacao.concluida"
)

// VersaoSuportada é a versão do esquema do corpo que este consumidor entende.
const VersaoSuportada = 1

// ErrInvalido marca um corpo que não segue o contrato; a mensagem é rejeitada
// e, após as tentativas configuradas, vai para a fila de mensagens mortas.
var ErrInvalido = errors.New("evento inválido")

var padraoUlid = regexp.MustCompile(`^[0-9A-HJKMNP-TV-Z]{26}$`)

// Evento é a mensagem recebida (event-carried state transfer).
type Evento struct {
	Tipo        string    `json:"tipo"`
	Versao      int       `json:"versao"`
	IdValidacao string    `json:"idValidacao"`
	OcorridoEm  time.Time `json:"ocorridoEm"`
	Maquina     Maquina   `json:"maquina"`
	Operador    Operador  `json:"operador"`
	Validacao   Validacao `json:"validacao"`
}

// Maquina identifica a máquina no evento.
type Maquina struct {
	Id          string      `json:"id"`
	Modelo      string      `json:"modelo"`
	Localizacao Localizacao `json:"localizacao"`
}

// Localizacao é o ponto onde a máquina está instalada.
type Localizacao struct {
	Id   string `json:"id"`
	Nome string `json:"nome"`
}

// Operador identifica quem executou a validação.
type Operador struct {
	Id   string `json:"id"`
	Nome string `json:"nome"`
}

// Validacao traz o estado relevante ao tipo do evento.
type Validacao struct {
	IniciadaEm        time.Time  `json:"iniciadaEm"`
	ExpiraEm          *time.Time `json:"expiraEm,omitempty"`
	ConcluidaEm       *time.Time `json:"concluidaEm,omitempty"`
	Leituras          *Leituras  `json:"leituras,omitempty"`
	CodigoRotacionado *bool      `json:"codigoRotacionado,omitempty"`
	Avisos            []string   `json:"avisos,omitempty"`
}

// Leituras são os valores lidos na máquina ao concluir a visita.
type Leituras struct {
	Medidor          int64 `json:"medidor"`
	UnidadesVendidas int64 `json:"unidadesVendidas"`
}

// Chave identifica um evento para fins de idempotência: (idValidacao, tipo).
type Chave struct {
	IdValidacao string `json:"idValidacao"`
	Tipo        string `json:"tipo"`
}

// Chave devolve a chave de idempotência do evento.
func (e Evento) Chave() Chave { return Chave{IdValidacao: e.IdValidacao, Tipo: e.Tipo} }

// TipoConhecido informa se o tipo faz parte do contrato.
func TipoConhecido(tipo string) bool {
	return tipo == TipoValidacaoIniciada || tipo == TipoValidacaoConcluida
}

// Decodificar converte o corpo JSON da mensagem em Evento e confere o contrato.
func Decodificar(corpo []byte) (Evento, error) {
	var e Evento
	if err := json.Unmarshal(corpo, &e); err != nil {
		return Evento{}, fmt.Errorf("%w: JSON mal formado: %v", ErrInvalido, err)
	}
	if err := e.Validar(); err != nil {
		return Evento{}, err
	}
	return e, nil
}

// Validar confere os campos obrigatórios e a versão.
func (e Evento) Validar() error {
	switch {
	case !TipoConhecido(e.Tipo):
		return fmt.Errorf("%w: tipo desconhecido %q", ErrInvalido, e.Tipo)
	case e.Versao != VersaoSuportada:
		return fmt.Errorf("%w: versão %d não suportada (esperada %d)", ErrInvalido, e.Versao, VersaoSuportada)
	case !padraoUlid.MatchString(e.IdValidacao):
		return fmt.Errorf("%w: idValidacao %q não é um ULID", ErrInvalido, e.IdValidacao)
	case e.Maquina.Id == "":
		return fmt.Errorf("%w: maquina.id ausente", ErrInvalido)
	case e.Operador.Id == "":
		return fmt.Errorf("%w: operador.id ausente", ErrInvalido)
	case e.OcorridoEm.IsZero():
		return fmt.Errorf("%w: ocorridoEm ausente", ErrInvalido)
	case e.Validacao.IniciadaEm.IsZero():
		return fmt.Errorf("%w: validacao.iniciadaEm ausente", ErrInvalido)
	case e.Tipo == TipoValidacaoConcluida && e.Validacao.ConcluidaEm == nil:
		return fmt.Errorf("%w: validacao.concluidaEm ausente em %s", ErrInvalido, e.Tipo)
	case e.Tipo == TipoValidacaoConcluida && e.Validacao.Leituras == nil:
		return fmt.Errorf("%w: validacao.leituras ausente em %s", ErrInvalido, e.Tipo)
	}
	return nil
}
