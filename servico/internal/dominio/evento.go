package dominio

import (
	"strconv"
	"time"
)

// TipoEvento identifica o evento publicado no Pub/Sub.
type TipoEvento string

const (
	EventoValidacaoIniciada  TipoEvento = "validacao.iniciada"
	EventoValidacaoConcluida TipoEvento = "validacao.concluida"
)

// VersaoEvento é a versão do esquema do corpo do evento.
const VersaoEvento = 1

// Evento é a mensagem enviada ao sistema central (docs/modelo-dados.md, seção 6),
// no estilo event-carried state transfer. A contrassenha nunca trafega aqui.
type Evento struct {
	Tipo        TipoEvento      `json:"tipo"`
	Versao      int             `json:"versao"`
	IdValidacao string          `json:"idValidacao"`
	OcorridoEm  time.Time       `json:"ocorridoEm"`
	Maquina     EventoMaquina   `json:"maquina"`
	Operador    EventoOperador  `json:"operador"`
	Validacao   EventoValidacao `json:"validacao"`
}

// EventoMaquina identifica a máquina no evento.
type EventoMaquina struct {
	Id          string      `json:"id"`
	Modelo      string      `json:"modelo"`
	Localizacao Localizacao `json:"localizacao"`
}

// EventoOperador identifica o operador no evento.
type EventoOperador struct {
	Id   string `json:"id"`
	Nome string `json:"nome"`
}

// EventoValidacao traz o estado da validação relevante ao tipo do evento.
type EventoValidacao struct {
	IniciadaEm        time.Time  `json:"iniciadaEm"`
	ExpiraEm          *time.Time `json:"expiraEm,omitempty"`
	ConcluidaEm       *time.Time `json:"concluidaEm,omitempty"`
	Leituras          *Leituras  `json:"leituras,omitempty"`
	CodigoRotacionado *bool      `json:"codigoRotacionado,omitempty"`
}

// NovoEventoIniciada monta o evento validacao.iniciada.
func NovoEventoIniciada(v *Validacao, m *Maquina, o *Operador) Evento {
	expira := v.ExpiraEm
	return Evento{
		Tipo:        EventoValidacaoIniciada,
		Versao:      VersaoEvento,
		IdValidacao: v.Id,
		OcorridoEm:  v.IniciadaEm,
		Maquina:     EventoMaquina{Id: m.Id, Modelo: m.Modelo, Localizacao: m.Localizacao},
		Operador:    EventoOperador{Id: o.Uid, Nome: o.Nome},
		Validacao:   EventoValidacao{IniciadaEm: v.IniciadaEm, ExpiraEm: &expira},
	}
}

// NovoEventoConcluida monta o evento validacao.concluida.
func NovoEventoConcluida(v *Validacao, m *Maquina, o *Operador) Evento {
	ocorrido := v.IniciadaEm
	if v.ConcluidaEm != nil {
		ocorrido = *v.ConcluidaEm
	}
	return Evento{
		Tipo:        EventoValidacaoConcluida,
		Versao:      VersaoEvento,
		IdValidacao: v.Id,
		OcorridoEm:  ocorrido,
		Maquina:     EventoMaquina{Id: m.Id, Modelo: m.Modelo, Localizacao: m.Localizacao},
		Operador:    EventoOperador{Id: o.Uid, Nome: o.Nome},
		Validacao: EventoValidacao{
			IniciadaEm:        v.IniciadaEm,
			ConcluidaEm:       v.ConcluidaEm,
			Leituras:          v.Leituras,
			CodigoRotacionado: v.CodigoRotacionado,
		},
	}
}

// Atributos devolve os atributos da mensagem Pub/Sub, usados para filtro e
// ordenação sem abrir o corpo.
func (e Evento) Atributos() map[string]string {
	return map[string]string{
		"tipo":        string(e.Tipo),
		"idValidacao": e.IdValidacao,
		"idMaquina":   e.Maquina.Id,
		"versao":      strconv.Itoa(e.Versao),
	}
}

// ChaveOrdenacao garante que eventos da mesma máquina sejam entregues em ordem.
func (e Evento) ChaveOrdenacao() string { return e.Maquina.Id }
