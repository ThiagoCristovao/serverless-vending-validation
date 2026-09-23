package dominio

import (
	"time"
	"unicode/utf8"
)

// PrazoValidacaoPadrao é o tempo para concluir uma validação aberta (RN-04).
const PrazoValidacaoPadrao = 15 * time.Minute

// TamanhoMaximoObservacoes limita o texto livre da conclusão.
const TamanhoMaximoObservacoes = 500

// StatusValidacao é o estado de uma validação.
type StatusValidacao string

const (
	StatusAberta    StatusValidacao = "aberta"
	StatusConcluida StatusValidacao = "concluida"
	StatusExpirada  StatusValidacao = "expirada"
)

// Leituras são os valores lidos no modo supervisor da máquina.
type Leituras struct {
	Medidor          int64 `json:"medidor"`
	UnidadesVendidas int64 `json:"unidadesVendidas"`
}

// Validar confere as restrições do contrato (inteiros não negativos).
func (l Leituras) Validar() error {
	if l.Medidor < 0 || l.UnidadesVendidas < 0 {
		return NovoErro(CodigoPayloadInvalido, "leituras devem ser inteiros não negativos")
	}
	return nil
}

// Dispositivo são metadados do aparelho do operador, só para diagnóstico.
type Dispositivo struct {
	Plataforma string
	VersaoApp  string
}

// OrigemQr guarda, para auditoria, os campos do adesivo que originou a validação.
type OrigemQr struct {
	Kid string
	Exp string
	Loc string
	Mod string
}

// EventosPublicados marca quando cada evento foi publicado (padrão outbox):
// nil significa publicação pendente de reconciliação (CE-14).
type EventosPublicados struct {
	IniciadaEm  *time.Time
	ConcluidaEm *time.Time
}

// Validacao é o registro de uma visita do operador a uma máquina, da leitura
// do QR à conclusão (coleção validacoes). Nunca guarda a contrassenha: guarda o
// contador da época, do qual o código é derivável.
type Validacao struct {
	Id         string
	IdMaquina  string
	IdOperador string
	Status     StatusValidacao

	IniciadaEm  time.Time
	ExpiraEm    time.Time
	ConcluidaEm *time.Time

	ContadorCodigo uint64

	ChaveIdempotencia string
	HashRequisicao    string

	Qr          OrigemQr
	Dispositivo *Dispositivo

	Leituras          *Leituras
	CodigoRotacionado *bool
	Observacoes       string

	Eventos EventosPublicados
}

// NovaValidacao monta uma validação aberta para a máquina e o operador dados.
func NovaValidacao(id string, maquina *Maquina, operador *Operador, payload *PayloadQr,
	chaveIdempotencia string, dispositivo *Dispositivo, agora time.Time, prazo time.Duration) *Validacao {
	if prazo <= 0 {
		prazo = PrazoValidacaoPadrao
	}
	return &Validacao{
		Id:                id,
		IdMaquina:         maquina.Id,
		IdOperador:        operador.Uid,
		Status:            StatusAberta,
		IniciadaEm:        agora,
		ExpiraEm:          agora.Add(prazo),
		ContadorCodigo:    maquina.ContadorCodigo,
		ChaveIdempotencia: chaveIdempotencia,
		HashRequisicao:    payload.HashRequisicao(),
		Qr:                OrigemQr{Kid: payload.Kid, Exp: payload.Exp, Loc: payload.Loc, Mod: payload.Mod},
		Dispositivo:       dispositivo,
	}
}

// Aberta informa se a validação ainda aguarda conclusão.
func (v *Validacao) Aberta() bool { return v.Status == StatusAberta }

// Expirou informa se uma validação aberta passou do prazo em `agora`.
func (v *Validacao) Expirou(agora time.Time) bool {
	return v.Status == StatusAberta && !agora.Before(v.ExpiraEm)
}

// MarcarExpirada encerra a validação por prazo (RN-04); o contador não rotaciona.
func (v *Validacao) MarcarExpirada() { v.Status = StatusExpirada }

// Concluir fecha a validação com as leituras informadas. A rotação do contador
// da máquina acontece em Maquina.RegistrarConclusao, quando codigoRotacionado
// é verdadeiro (RN-02).
func (v *Validacao) Concluir(leituras Leituras, codigoRotacionado bool, observacoes string, agora time.Time) error {
	switch {
	case v.Status == StatusConcluida:
		return NovoErro(CodigoValidacaoJaConcluida, "a validação já foi concluída").ComValidacao(v.Id)
	case v.Status == StatusExpirada || v.Expirou(agora):
		return NovoErro(CodigoValidacaoExpirada, "a validação expirou; leia o QR novamente").ComValidacao(v.Id)
	}
	if err := leituras.Validar(); err != nil {
		return err
	}
	if utf8.RuneCountInString(observacoes) > TamanhoMaximoObservacoes {
		return NovoErro(CodigoPayloadInvalido, "observações excedem o tamanho máximo")
	}
	// TODO (RN-07, decidir com o orientador): medidor menor que o da última
	// validação concluída da máquina — erro ou apenas aviso?
	quando := agora
	v.Status = StatusConcluida
	v.ConcluidaEm = &quando
	v.Leituras = &leituras
	v.CodigoRotacionado = &codigoRotacionado
	v.Observacoes = observacoes
	return nil
}

// MesmaConclusao informa se a conclusão já registrada coincide com a
// informada, o que caracteriza um reenvio idempotente.
func (v *Validacao) MesmaConclusao(leituras Leituras, codigoRotacionado bool) bool {
	return v.Status == StatusConcluida &&
		v.Leituras != nil && *v.Leituras == leituras &&
		v.CodigoRotacionado != nil && *v.CodigoRotacionado == codigoRotacionado
}
