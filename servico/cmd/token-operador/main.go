// Comando token-operador obtém um ID token do Firebase Authentication para um
// operador de testes, criando a conta se ela não existir. Usa a API REST do
// Identity Toolkit com a chave de API web do projeto (a mesma do
// google-services.json). O token serve para chamar o gateway com curl.
//
//	go run ./cmd/token-operador -google-services ../aplicativo/android/app/google-services.json \
//	   -email operador1@exemplo.invalid -senha 'senha-forte' -criar
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const identityToolkit = "https://identitytoolkit.googleapis.com/v1/accounts:"

func main() {
	var (
		apiKey         = flag.String("api-key", os.Getenv("SVV_API_KEY_WEB"), "chave de API web do projeto Firebase")
		googleServices = flag.String("google-services", "", "caminho do google-services.json de onde ler a chave de API")
		email          = flag.String("email", "", "e-mail do operador")
		senha          = flag.String("senha", "", "senha do operador")
		criar          = flag.Bool("criar", false, "cria a conta se não existir")
		soToken        = flag.Bool("so-token", false, "imprime apenas o ID token")
	)
	flag.Parse()
	if *googleServices != "" {
		chave, err := chaveDoGoogleServices(*googleServices)
		if err != nil {
			falhar(err)
		}
		*apiKey = chave
	}
	if *apiKey == "" || *email == "" || *senha == "" {
		falhar(errors.New("-api-key (ou -google-services), -email e -senha são obrigatórios"))
	}

	resposta, err := chamar("signInWithPassword", *apiKey, *email, *senha)
	// Com a proteção contra enumeração de e-mails, o Identity Platform responde
	// INVALID_LOGIN_CREDENTIALS tanto para conta inexistente quanto para senha errada.
	if err != nil && *criar && (strings.Contains(err.Error(), "EMAIL_NOT_FOUND") || strings.Contains(err.Error(), "INVALID_LOGIN_CREDENTIALS")) {
		criada, errCriar := chamar("signUp", *apiKey, *email, *senha)
		switch {
		case errCriar == nil:
			resposta, err = criada, nil
			fmt.Fprintf(os.Stderr, "conta criada: %s (uid %s)\n", *email, resposta.LocalId)
		case strings.Contains(errCriar.Error(), "EMAIL_EXISTS"):
			err = fmt.Errorf("a conta %s já existe e a senha informada não confere", *email)
		default:
			err = errCriar
		}
	}
	if err != nil {
		falhar(err)
	}
	if *soToken {
		fmt.Println(resposta.IdToken)
		return
	}
	expira, _ := time.ParseDuration(resposta.ExpiresIn + "s")
	saida, _ := json.MarshalIndent(map[string]any{
		"uid":      resposta.LocalId,
		"email":    resposta.Email,
		"idToken":  resposta.IdToken,
		"expiraEm": time.Now().UTC().Add(expira).Format(time.RFC3339),
	}, "", "  ")
	fmt.Println(string(saida))
}

type respostaIdentity struct {
	LocalId   string `json:"localId"`
	Email     string `json:"email"`
	IdToken   string `json:"idToken"`
	ExpiresIn string `json:"expiresIn"`
}

func chamar(operacao, apiKey, email, senha string) (*respostaIdentity, error) {
	corpo, _ := json.Marshal(map[string]any{"email": email, "password": senha, "returnSecureToken": true})
	resp, err := http.Post(identityToolkit+operacao+"?key="+apiKey, "application/json", bytes.NewReader(corpo))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	dados, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		var e struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		_ = json.Unmarshal(dados, &e)
		return nil, fmt.Errorf("%s: %s", operacao, e.Error.Message)
	}
	var r respostaIdentity
	if err := json.Unmarshal(dados, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

func chaveDoGoogleServices(caminho string) (string, error) {
	conteudo, err := os.ReadFile(caminho)
	if err != nil {
		return "", err
	}
	var gs struct {
		Client []struct {
			ApiKey []struct {
				CurrentKey string `json:"current_key"`
			} `json:"api_key"`
		} `json:"client"`
	}
	if err := json.Unmarshal(conteudo, &gs); err != nil {
		return "", err
	}
	if len(gs.Client) == 0 || len(gs.Client[0].ApiKey) == 0 {
		return "", errors.New("google-services.json sem api_key")
	}
	return gs.Client[0].ApiKey[0].CurrentKey, nil
}

func falhar(err error) {
	fmt.Fprintln(os.Stderr, "erro:", err)
	os.Exit(1)
}
