package dominio

import "time"

// Localizacao é o ponto físico onde a máquina está instalada.
type Localizacao struct {
	Id   string `json:"id"`
	Nome string `json:"nome"`
}

// Maquina é uma vending machine cadastrada na rede (coleção maquinas).
type Maquina struct {
	Id          string
	Modelo      string
	Localizacao Localizacao
	Ativa       bool

	// ContadorCodigo é o contador de rotação da contrassenha (ADR-0002).
	ContadorCodigo uint64
	// ValidacaoAbertaId aponta para a validação aberta, se houver (RN-03).
	ValidacaoAbertaId string

	UltimaValidacaoId string
	UltimaValidacaoEm *time.Time
	UltimoMedidor     *int64

	CriadaEm     time.Time
	AtualizadaEm time.Time
}

// Operador é uma pessoa autorizada a validar máquinas (coleção operadores).
// A chave é o uid do Firebase Authentication.
type Operador struct {
	Uid      string
	Nome     string
	Email    string
	Rota     string
	Ativo    bool
	CriadoEm time.Time
}

// RegistrarConclusao aplica na máquina os efeitos de uma validação concluída:
// rotação do contador quando o novo código foi programado (RN-02), última
// validação, último medidor e liberação da trava de validação aberta.
func (m *Maquina) RegistrarConclusao(v *Validacao, agora time.Time) {
	if v.CodigoRotacionado != nil && *v.CodigoRotacionado {
		m.ContadorCodigo++
	}
	m.ValidacaoAbertaId = ""
	m.UltimaValidacaoId = v.Id
	quando := agora
	m.UltimaValidacaoEm = &quando
	if v.Leituras != nil {
		medidor := v.Leituras.Medidor
		m.UltimoMedidor = &medidor
	}
	m.AtualizadaEm = agora
}
