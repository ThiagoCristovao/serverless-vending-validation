package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"google.golang.org/api/option"
	htransport "google.golang.org/api/transport/http"

	"github.com/ThiagoCristovao/serverless-vending-validation/sistema-central-simulado/internal/laboratorio"
)

// loteCarga é o arquivo que liga operadores virtuais a máquinas de carga; o
// mesmo formato é lido por servico/cmd/semear-firestore -lote.
type loteCarga struct {
	Maquinas []struct {
		Id      string `json:"id"`
		Modelo  string `json:"modelo"`
		Loc     string `json:"loc"`
		LocNome string `json:"locNome"`
	} `json:"maquinas"`
	Operadores []struct {
		Uid   string `json:"uid"`
		Nome  string `json:"nome"`
		Email string `json:"email"`
	} `json:"operadores"`
}

const (
	locCarga     = "CARGA-L"
	locCargaNome = "Laboratório de carga"
)

func emailCarga(i int) string   { return fmt.Sprintf("carga-%03d@exemplo.invalid", i) }
func maquinaCarga(i int) string { return fmt.Sprintf("VM-CARGA-%03d", i) }

// preparar cria (ou reaproveita) N contas no Firebase Authentication pela API
// administrativa do Identity Toolkit e grava o lote de operadores e máquinas
// para o semear-firestore.
func (x *avaliacao) preparar(ctx context.Context) error {
	if x.lote == "" || x.senha == "" {
		return errors.New("SVV_CARGA_LOTE e SVV_CARGA_SENHA são obrigatórios (use make carga-preparar)")
	}
	admin, _, err := htransport.NewClient(ctx, option.WithScopes("https://www.googleapis.com/auth/cloud-platform"))
	if err != nil {
		return err
	}
	var lote loteCarga
	if conteudo, err := os.ReadFile(x.lote); err == nil {
		_ = json.Unmarshal(conteudo, &lote)
	}
	existentes := map[string]bool{}
	for _, o := range lote.Operadores {
		existentes[o.Email] = true
	}
	for i := 1; i <= x.operadores; i++ {
		email := emailCarga(i)
		if !existentes[email] {
			uid, err := x.criarConta(ctx, admin, email)
			if err != nil {
				return fmt.Errorf("%s: %w", email, err)
			}
			lote.Operadores = append(lote.Operadores, struct {
				Uid   string `json:"uid"`
				Nome  string `json:"nome"`
				Email string `json:"email"`
			}{Uid: uid, Nome: fmt.Sprintf("Operador de carga %03d", i), Email: email})
			fmt.Fprintf(os.Stderr, "%s conta %s (uid %s)\n", laboratorio.Hora(), email, uid)
		}
		if i > len(lote.Maquinas) {
			lote.Maquinas = append(lote.Maquinas, struct {
				Id      string `json:"id"`
				Modelo  string `json:"modelo"`
				Loc     string `json:"loc"`
				LocNome string `json:"locNome"`
			}{Id: maquinaCarga(i), Modelo: x.cfg.Modelo, Loc: locCarga, LocNome: locCargaNome})
		}
	}
	conteudo, _ := json.MarshalIndent(lote, "", "  ")
	if err := os.MkdirAll(filepath.Dir(x.lote), 0o700); err != nil {
		return err
	}
	if err := os.WriteFile(x.lote, conteudo, 0o600); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "lote gravado em %s: %d operadores, %d máquinas; semeie o Firestore com semear-firestore -lote\n", x.lote, len(lote.Operadores), len(lote.Maquinas))
	return nil
}

// criarConta usa a API administrativa (credenciais ADC), que não está sujeita
// aos limites de cadastro por IP da API de cliente; se a conta existir, entra
// com a senha para obter o uid.
func (x *avaliacao) criarConta(ctx context.Context, admin *http.Client, email string) (string, error) {
	corpo, _ := json.Marshal(map[string]any{"email": email, "password": x.senha, "displayName": "Operador de carga", "emailVerified": true})
	url := fmt.Sprintf("https://identitytoolkit.googleapis.com/v1/projects/%s/accounts", x.cfg.Projeto)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(corpo))
	req.Header.Set("Content-Type", "application/json")
	resp, err := admin.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var dados struct {
		LocalId string `json:"localId"`
		Error   struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&dados)
	switch {
	case resp.StatusCode == http.StatusOK:
		return dados.LocalId, nil
	case strings.Contains(dados.Error.Message, "EMAIL_EXISTS") || strings.Contains(dados.Error.Message, "DUPLICATE_EMAIL"):
		op, err := laboratorio.Entrar(ctx, x.cfg.ApiKey, email, x.senha)
		if err != nil {
			return "", fmt.Errorf("conta existe com outra senha: %w", err)
		}
		return op.Uid, nil
	default:
		return "", fmt.Errorf("identitytoolkit admin %d: %s", resp.StatusCode, dados.Error.Message)
	}
}

// --- tokens dos operadores virtuais (cache de 1 h ao lado do lote) ----------------

type tokenGuardado struct {
	Uid      string    `json:"uid"`
	Token    string    `json:"token"`
	ExpiraEm time.Time `json:"expiraEm"`
}

func (x *avaliacao) caminhoTokens() string {
	return filepath.Join(filepath.Dir(x.lote), "carga-tokens.json")
}

// tokens entra com cada operador (reaproveitando tokens ainda válidos por
// pelo menos `minimo`), com paralelismo baixo para não disparar a proteção
// contra abuso do Identity Toolkit.
func (x *avaliacao) tokens(ctx context.Context, emails []string, minimo time.Duration) (map[string]laboratorio.Operador, error) {
	cache := map[string]tokenGuardado{}
	if conteudo, err := os.ReadFile(x.caminhoTokens()); err == nil {
		_ = json.Unmarshal(conteudo, &cache)
	}
	saida := map[string]laboratorio.Operador{}
	var pendentes []string
	for _, e := range emails {
		if t, ok := cache[e]; ok && time.Until(t.ExpiraEm) > minimo {
			saida[e] = laboratorio.Operador{Token: t.Token, Uid: t.Uid, Email: e}
		} else {
			pendentes = append(pendentes, e)
		}
	}
	type resultado struct {
		email string
		op    laboratorio.Operador
		err   error
	}
	resultados := make(chan resultado)
	fila := make(chan string)
	for w := 0; w < 4; w++ {
		go func() {
			for e := range fila {
				var op laboratorio.Operador
				var err error
				for tentativa := 0; tentativa < 5; tentativa++ {
					if op, err = laboratorio.Entrar(ctx, x.cfg.ApiKey, e, x.senha); err == nil || ctx.Err() != nil {
						break
					}
					time.Sleep(time.Duration(1<<tentativa) * time.Second)
				}
				resultados <- resultado{e, op, err}
			}
		}()
	}
	go func() {
		for _, e := range pendentes {
			fila <- e
		}
		close(fila)
	}()
	for range pendentes {
		r := <-resultados
		if r.err != nil {
			return nil, fmt.Errorf("token de %s: %w", r.email, r.err)
		}
		saida[r.email] = r.op
		cache[r.email] = tokenGuardado{Uid: r.op.Uid, Token: r.op.Token, ExpiraEm: time.Now().Add(55 * time.Minute)}
	}
	if len(pendentes) > 0 {
		conteudo, _ := json.MarshalIndent(cache, "", "  ")
		_ = os.WriteFile(x.caminhoTokens(), conteudo, 0o600)
	}
	return saida, nil
}

func (x *avaliacao) carregarLote() (loteCarga, error) {
	var lote loteCarga
	if x.lote == "" {
		return lote, errors.New("SVV_CARGA_LOTE não definido (use make carga)")
	}
	conteudo, err := os.ReadFile(x.lote)
	if err != nil {
		return lote, fmt.Errorf("lote de carga (rode make carga-preparar): %w", err)
	}
	if err := json.Unmarshal(conteudo, &lote); err != nil {
		return lote, err
	}
	return lote, nil
}
