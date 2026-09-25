package apihttp

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

// CabecalhoUserinfo é o cabeçalho em que o API Gateway repassa as claims do JWT
// já verificado (payload em base64url).
const CabecalhoUserinfo = "X-Apigateway-Api-Userinfo"

var errSemIdentidade = errors.New("requisição sem identidade do operador")

// uidDe extrai o identificador do operador. Em produção só o cabeçalho do
// gateway é aceito; em execução local, um cabeçalho de teste configurado em
// Opcoes pode substituí-lo.
func (m *manipulador) uidDe(r *http.Request) (string, error) {
	if bruto := r.Header.Get(CabecalhoUserinfo); bruto != "" {
		return uidDoUserinfo(bruto)
	}
	if m.o.CabecalhoOperadorDeTeste != "" {
		if uid := strings.TrimSpace(r.Header.Get(m.o.CabecalhoOperadorDeTeste)); uid != "" {
			return uid, nil
		}
	}
	return "", errSemIdentidade
}

// uidDoUserinfo decodifica o payload do JWT e devolve a claim `sub` (ou
// `user_id`, que o Firebase também emite).
func uidDoUserinfo(bruto string) (string, error) {
	dados, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(bruto, "="))
	if err != nil {
		if dados, err = base64.StdEncoding.DecodeString(bruto); err != nil {
			return "", errors.New("cabeçalho de identidade ilegível")
		}
	}
	var claims struct {
		Sub    string `json:"sub"`
		UserID string `json:"user_id"`
	}
	if err := json.Unmarshal(dados, &claims); err != nil {
		return "", errors.New("cabeçalho de identidade ilegível")
	}
	switch {
	case claims.Sub != "":
		return claims.Sub, nil
	case claims.UserID != "":
		return claims.UserID, nil
	}
	return "", errors.New("cabeçalho de identidade sem sub")
}
