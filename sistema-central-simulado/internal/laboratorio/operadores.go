package laboratorio

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// Entrar obtém um ID token de um operador pela API REST do Identity Toolkit.
func Entrar(ctx context.Context, apiKey, email, senha string) (Operador, error) {
	return identity(ctx, "signInWithPassword", apiKey, email, senha)
}

// CriarOperador cria a conta no Firebase Authentication (ou entra, se já
// existir com a mesma senha). Usada pelos testes de carga para os operadores
// virtuais (Sprint 10).
func CriarOperador(ctx context.Context, apiKey, email, senha string) (Operador, error) {
	op, err := identity(ctx, "signUp", apiKey, email, senha)
	if err != nil && strings.Contains(err.Error(), "EMAIL_EXISTS") {
		return Entrar(ctx, apiKey, email, senha)
	}
	return op, err
}

func identity(ctx context.Context, operacao, apiKey, email, senha string) (Operador, error) {
	corpo, _ := json.Marshal(map[string]any{"email": email, "password": senha, "returnSecureToken": true})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, "https://identitytoolkit.googleapis.com/v1/accounts:"+operacao+"?key="+apiKey, bytes.NewReader(corpo))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return Operador{}, err
	}
	defer resp.Body.Close()
	var dados struct {
		IdToken string `json:"idToken"`
		LocalId string `json:"localId"`
		Error   struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&dados); err != nil {
		return Operador{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return Operador{}, fmt.Errorf("identitytoolkit %s %d: %s", operacao, resp.StatusCode, dados.Error.Message)
	}
	return Operador{Token: dados.IdToken, Uid: dados.LocalId, Email: email}, nil
}
