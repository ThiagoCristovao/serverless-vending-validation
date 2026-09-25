// Package apihttp expõe os casos de uso pelo contrato api/openapi.yaml:
// decodifica requisições, identifica o operador pelo cabeçalho do API Gateway,
// chama a aplicação e escreve respostas JSON ou erros em Problem Details.
package apihttp

import (
	"encoding/json"
	"net/http"

	"github.com/ThiagoCristovao/serverless-vending-validation/servico/internal/dominio"
)

// tipoProblema é o prefixo do campo `type` da RFC 9457 (urn:svv:erro:<codigo>).
const tipoProblema = "urn:svv:erro:"

// problema é o corpo application/problem+json do contrato.
type problema struct {
	Type        string `json:"type"`
	Title       string `json:"title"`
	Status      int    `json:"status"`
	Detail      string `json:"detail,omitempty"`
	Instance    string `json:"instance,omitempty"`
	Codigo      string `json:"codigo"`
	IdValidacao string `json:"idValidacao,omitempty"`
}

// statusDe mapeia o código de domínio para o status HTTP do contrato.
func statusDe(c dominio.Codigo) int {
	switch c {
	case dominio.CodigoPayloadInvalido:
		return http.StatusBadRequest
	case dominio.CodigoOperadorNaoAutorizado:
		return http.StatusForbidden
	case dominio.CodigoMaquinaDesconhecida, dominio.CodigoValidacaoNaoEncontrada:
		return http.StatusNotFound
	case dominio.CodigoValidacaoEmAndamento, dominio.CodigoValidacaoJaConcluida, dominio.CodigoMaquinaInativa:
		return http.StatusConflict
	case dominio.CodigoQrExpirado, dominio.CodigoValidacaoExpirada:
		return http.StatusGone
	case dominio.CodigoModeloNaoSuportado, dominio.CodigoChaveQrDesconhecida,
		dominio.CodigoAssinaturaInvalida, dominio.CodigoChaveIdempotenciaConflitante:
		return http.StatusUnprocessableEntity
	case dominio.CodigoDependenciaIndisponivel:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

var titulos = map[dominio.Codigo]string{
	dominio.CodigoPayloadInvalido:              "Requisição inválida",
	dominio.CodigoModeloNaoSuportado:           "Modelo não suportado",
	dominio.CodigoQrExpirado:                   "Adesivo vencido",
	dominio.CodigoChaveQrDesconhecida:          "Chave do adesivo desconhecida",
	dominio.CodigoAssinaturaInvalida:           "Assinatura inválida",
	dominio.CodigoMaquinaDesconhecida:          "Máquina desconhecida",
	dominio.CodigoMaquinaInativa:               "Máquina inativa",
	dominio.CodigoOperadorNaoAutorizado:        "Operador não autorizado",
	dominio.CodigoValidacaoEmAndamento:         "Validação em andamento",
	dominio.CodigoValidacaoJaConcluida:         "Validação já concluída",
	dominio.CodigoValidacaoExpirada:            "Validação expirada",
	dominio.CodigoValidacaoNaoEncontrada:       "Validação não encontrada",
	dominio.CodigoChaveIdempotenciaConflitante: "Chave de idempotência conflitante",
	dominio.CodigoDependenciaIndisponivel:      "Serviço temporariamente indisponível",
	dominio.CodigoErroInterno:                  "Erro interno",
}

// escreverErro converte err em Problem Details. Erros que não são de domínio
// viram erro_interno sem expor a causa ao cliente.
func escreverErro(w http.ResponseWriter, r *http.Request, err error) {
	codigo := dominio.CodigoDe(err)
	status := statusDe(codigo)
	p := problema{
		Type:     tipoProblema + string(codigo),
		Title:    titulos[codigo],
		Status:   status,
		Instance: r.URL.Path,
		Codigo:   string(codigo),
	}
	if e := dominio.ErroDe(err); e != nil {
		p.Detail = e.Mensagem
		p.IdValidacao = e.IdValidacao
	} else {
		p.Detail = "Falha inesperada. Tente novamente em instantes."
	}
	escreverProblema(w, r, p, err)
}

// escreverNaoAutenticado responde 401 quando a requisição não traz identidade.
func escreverNaoAutenticado(w http.ResponseWriter, r *http.Request, detalhe string) {
	escreverProblema(w, r, problema{
		Type:     tipoProblema + string(dominio.CodigoOperadorNaoAutorizado),
		Title:    "Não autenticado",
		Status:   http.StatusUnauthorized,
		Detail:   detalhe,
		Instance: r.URL.Path,
		Codigo:   string(dominio.CodigoOperadorNaoAutorizado),
	}, nil)
}

func escreverProblema(w http.ResponseWriter, r *http.Request, p problema, causa error) {
	if reg := registroDe(r.Context()); reg != nil {
		reg.codigo = p.Codigo
		reg.erro = causa
	}
	w.Header().Set("Content-Type", "application/problem+json; charset=utf-8")
	w.WriteHeader(p.Status)
	_ = json.NewEncoder(w).Encode(p)
}

func escreverJSON(w http.ResponseWriter, status int, corpo any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(corpo)
}
