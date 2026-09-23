// Package dominio contém as entidades e as regras de negócio da validação de
// máquinas de vending. Não depende de nuvem, transporte nem persistência: tudo
// aqui é testável sem rede (critério da Sprint 4).
package dominio

import (
	"errors"
	"fmt"
)

// Codigo identifica um erro de domínio de forma estável. Espelha a enumeração
// `codigo` do contrato api/openapi.yaml (ADR-0009); o aplicativo decide a
// mensagem exibida a partir dele, nunca a partir do texto.
type Codigo string

const (
	CodigoPayloadInvalido              Codigo = "payload_invalido"
	CodigoModeloNaoSuportado           Codigo = "modelo_nao_suportado"
	CodigoQrExpirado                   Codigo = "qr_expirado"
	CodigoChaveQrDesconhecida          Codigo = "chave_qr_desconhecida"
	CodigoAssinaturaInvalida           Codigo = "assinatura_invalida"
	CodigoMaquinaDesconhecida          Codigo = "maquina_desconhecida"
	CodigoMaquinaInativa               Codigo = "maquina_inativa"
	CodigoOperadorNaoAutorizado        Codigo = "operador_nao_autorizado"
	CodigoValidacaoEmAndamento         Codigo = "validacao_em_andamento"
	CodigoValidacaoJaConcluida         Codigo = "validacao_ja_concluida"
	CodigoValidacaoExpirada            Codigo = "validacao_expirada"
	CodigoValidacaoNaoEncontrada       Codigo = "validacao_nao_encontrada"
	CodigoChaveIdempotenciaConflitante Codigo = "chave_idempotencia_conflitante"
	CodigoDependenciaIndisponivel      Codigo = "dependencia_indisponivel"
	CodigoErroInterno                  Codigo = "erro_interno"
)

// Erro é um erro de domínio: código estável, mensagem legível e, quando faz
// sentido, o identificador da validação envolvida.
type Erro struct {
	Codigo      Codigo
	Mensagem    string
	IdValidacao string
	causa       error
}

// NovoErro cria um erro de domínio com o código e a mensagem dados.
func NovoErro(codigo Codigo, mensagem string) *Erro {
	return &Erro{Codigo: codigo, Mensagem: mensagem}
}

func (e *Erro) Error() string {
	if e.causa != nil {
		return fmt.Sprintf("%s: %s: %v", e.Codigo, e.Mensagem, e.causa)
	}
	return fmt.Sprintf("%s: %s", e.Codigo, e.Mensagem)
}

// Unwrap expõe a causa técnica, quando houver.
func (e *Erro) Unwrap() error { return e.causa }

// ComValidacao devolve uma cópia do erro associada à validação indicada.
func (e *Erro) ComValidacao(id string) *Erro {
	c := *e
	c.IdValidacao = id
	return &c
}

// ComCausa devolve uma cópia do erro com a causa técnica anexada.
func (e *Erro) ComCausa(causa error) *Erro {
	c := *e
	c.causa = causa
	return &c
}

// CodigoDe devolve o código de domínio de err, ou erro_interno quando err não
// carrega um *Erro.
func CodigoDe(err error) Codigo {
	if e := ErroDe(err); e != nil {
		return e.Codigo
	}
	return CodigoErroInterno
}

// ErroDe devolve o *Erro embutido em err, ou nil.
func ErroDe(err error) *Erro {
	var e *Erro
	if errors.As(err, &e) {
		return e
	}
	return nil
}
