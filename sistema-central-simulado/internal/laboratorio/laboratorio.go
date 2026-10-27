// Package laboratorio reúne o que a suíte e2e e os experimentos de resiliência
// (Sprints 8 e 9) precisam para exercitar o ecossistema implantado: a
// configuração do ambiente, a sessão do operador de testes, adesivos
// assinados, chamadas ao gateway e o consumidor simulado em processo.
package laboratorio

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// Configuracao vem do ambiente (make e2e / make experimento preenchem a partir
// do Terraform e de chaves/). As variáveis têm o prefixo SVV_E2E_.
type Configuracao struct {
	Projeto       string
	Gateway       string // base com /v1, ex.: https://host/v1
	Prefixo       string // svv-dev → svv-dev-validacoes, svv-dev-sistema-central, ...
	ApiKey        string
	OperadorEmail string
	OperadorSenha string
	ChavePrivada  string // caminho (ou base64url) da chave Ed25519 que assina adesivos
	MaquinaA      string
	MaquinaB      string
	Loc           string
	Modelo        string
	Kid           string
	PrazoDlq      time.Duration
}

// Topico principal.
func (c Configuracao) Topico() string { return c.Prefixo + "-validacoes" }

// Assinatura do sistema central.
func (c Configuracao) Assinatura() string { return c.Prefixo + "-sistema-central" }

// AssinaturaDlq de inspeção da fila de mensagens mortas.
func (c Configuracao) AssinaturaDlq() string { return c.Prefixo + "-validacoes-dlq-inspecao" }

// NomeFuncao é o nome do serviço Cloud Run da função de validação.
func (c Configuracao) NomeFuncao() string { return c.Prefixo + "-validacao" }

// ContaFuncao é o e-mail da conta de serviço da função (módulo iam).
func (c Configuracao) ContaFuncao() string {
	return fmt.Sprintf("%s-funcao-validacao@%s.iam.gserviceaccount.com", c.Prefixo, c.Projeto)
}

func valorOu(nome, padrao string) string {
	if v := os.Getenv(nome); v != "" {
		return v
	}
	return padrao
}

// CarregarConfiguracao lê as variáveis SVV_E2E_* e confere as obrigatórias.
func CarregarConfiguracao() (Configuracao, error) {
	c := Configuracao{
		Projeto:       os.Getenv("SVV_E2E_PROJETO"),
		Gateway:       strings.TrimSuffix(os.Getenv("SVV_E2E_GATEWAY"), "/"),
		Prefixo:       valorOu("SVV_E2E_PREFIXO", "svv-dev"),
		ApiKey:        os.Getenv("SVV_E2E_API_KEY_WEB"),
		OperadorEmail: os.Getenv("SVV_E2E_OPERADOR_EMAIL"),
		OperadorSenha: os.Getenv("SVV_E2E_OPERADOR_SENHA"),
		ChavePrivada:  os.Getenv("SVV_E2E_CHAVE_PRIVADA"),
		MaquinaA:      valorOu("SVV_E2E_MAQUINA_A", "VM-E2E1"),
		MaquinaB:      valorOu("SVV_E2E_MAQUINA_B", "VM-E2E2"),
		Loc:           valorOu("SVV_E2E_LOC", "E2E-L"),
		Modelo:        valorOu("SVV_E2E_MODELO", "CN168"),
		Kid:           valorOu("SVV_E2E_KID", "k1"),
		PrazoDlq:      10 * time.Minute,
	}
	if c.Gateway == "" {
		return c, errors.New("SVV_E2E_GATEWAY não definido (use make e2e ou make experimento)")
	}
	if c.ApiKey == "" {
		if gs := os.Getenv("SVV_E2E_GOOGLE_SERVICES"); gs != "" {
			chave, err := ChaveDoGoogleServices(gs)
			if err != nil {
				return c, err
			}
			c.ApiKey = chave
		}
	}
	var faltando []string
	for _, par := range [][2]string{
		{"SVV_E2E_PROJETO", c.Projeto},
		{"SVV_E2E_API_KEY_WEB (ou SVV_E2E_GOOGLE_SERVICES)", c.ApiKey},
		{"SVV_E2E_OPERADOR_EMAIL", c.OperadorEmail},
		{"SVV_E2E_OPERADOR_SENHA", c.OperadorSenha},
		{"SVV_E2E_CHAVE_PRIVADA", c.ChavePrivada},
	} {
		if par[1] == "" {
			faltando = append(faltando, par[0])
		}
	}
	if len(faltando) > 0 {
		return c, fmt.Errorf("variáveis ausentes: %s", strings.Join(faltando, ", "))
	}
	if v := os.Getenv("SVV_E2E_PRAZO_DLQ"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return c, fmt.Errorf("SVV_E2E_PRAZO_DLQ: %w", err)
		}
		c.PrazoDlq = d
	}
	return c, nil
}

// ChaveDoGoogleServices extrai a chave de API web do google-services.json.
func ChaveDoGoogleServices(caminho string) (string, error) {
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
		return "", fmt.Errorf("%s: %w", caminho, err)
	}
	if len(gs.Client) == 0 || len(gs.Client[0].ApiKey) == 0 {
		return "", fmt.Errorf("%s: sem client[0].api_key[0].current_key", caminho)
	}
	return gs.Client[0].ApiKey[0].CurrentKey, nil
}

// --- operador (Firebase Authentication) ------------------------------------------

// Operador autenticado no Firebase.
type Operador struct {
	Token string
	Uid   string
	Email string
}

// EntrarComoOperador obtém um ID token pela API REST do Identity Toolkit.
func EntrarComoOperador(ctx context.Context, cfg Configuracao) (Operador, error) {
	corpo, _ := json.Marshal(map[string]any{"email": cfg.OperadorEmail, "password": cfg.OperadorSenha, "returnSecureToken": true})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, "https://identitytoolkit.googleapis.com/v1/accounts:signInWithPassword?key="+cfg.ApiKey, bytes.NewReader(corpo))
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
		return Operador{}, fmt.Errorf("identitytoolkit %d: %s", resp.StatusCode, dados.Error.Message)
	}
	return Operador{Token: dados.IdToken, Uid: dados.LocalId, Email: cfg.OperadorEmail}, nil
}

// --- adesivos --------------------------------------------------------------------

// Adesivo assina payloads de QR com a chave privada local (docs/payload-qr.md).
type Adesivo struct {
	privada ed25519.PrivateKey
	Modelo  string
	Loc     string
	Kid     string
}

// NovoAdesivo carrega a chave privada (base64url de 64 bytes ou caminho).
func NovoAdesivo(cfg Configuracao) (*Adesivo, error) {
	valor := cfg.ChavePrivada
	if conteudo, err := os.ReadFile(valor); err == nil {
		valor = strings.TrimSpace(string(conteudo))
	}
	bytes, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(valor))
	if err != nil || len(bytes) != ed25519.PrivateKeySize {
		return nil, errors.New("chave privada deve ser Ed25519 (64 bytes) em base64url")
	}
	return &Adesivo{privada: ed25519.PrivateKey(bytes), Modelo: cfg.Modelo, Loc: cfg.Loc, Kid: cfg.Kid}, nil
}

// Qr devolve o payload assinado da máquina, com validade de seis meses.
func (a *Adesivo) Qr(maq string) map[string]any {
	exp := time.Now().UTC().AddDate(0, 6, 0).Format("2006-01-02")
	canonica := fmt.Sprintf("%d|%s|%s|%s|%s|%s", 1, maq, a.Modelo, a.Loc, exp, a.Kid)
	sig := ed25519.Sign(a.privada, []byte(canonica))
	return map[string]any{"v": 1, "maq": maq, "mod": a.Modelo, "loc": a.Loc, "exp": exp, "kid": a.Kid, "sig": base64.RawURLEncoding.EncodeToString(sig)}
}

// CorpoValidacao monta a requisição de POST /validacoes para um QR.
func CorpoValidacao(qr map[string]any) map[string]any {
	return map[string]any{
		"qr":          qr,
		"lidoEm":      time.Now().UTC().Format(time.RFC3339),
		"dispositivo": map[string]any{"plataforma": "android", "versaoApp": "laboratorio"},
	}
}

// CorpoConclusao monta a requisição de conclusão. O medidor cresce com o
// tempo para não disparar o aviso da RN-07 entre execuções.
func CorpoConclusao() map[string]any {
	return map[string]any{
		"leituras":          map[string]any{"medidor": time.Now().Unix(), "unidadesVendidas": 7},
		"codigoRotacionado": true,
		"observacoes":       "laboratório de testes",
	}
}

// --- gateway ---------------------------------------------------------------------

// Resposta de uma chamada ao gateway.
type Resposta struct {
	Status  int
	Corpo   map[string]any
	Bruto   []byte
	Duracao time.Duration
}

// Texto devolve um campo de texto do corpo (vazio se ausente).
func (r Resposta) Texto(campo string) string {
	v, _ := r.Corpo[campo].(string)
	return v
}

// Gateway é o cliente HTTP do serviço de validação, com o token do operador.
type Gateway struct {
	Base    string
	Token   string
	Cliente *http.Client
	// Registrar recebe uma linha por chamada (opcional).
	Registrar func(linha string)
}

// NovoGateway prepara o cliente.
func NovoGateway(cfg Configuracao, op Operador, registrar func(string)) *Gateway {
	return &Gateway{Base: cfg.Gateway, Token: op.Token, Cliente: &http.Client{Timeout: 30 * time.Second}, Registrar: registrar}
}

// Chamar faz uma requisição autenticada; corpo nil omite o JSON.
func (g *Gateway) Chamar(ctx context.Context, metodo, caminho string, corpo any, chaveIdempotencia string) (Resposta, error) {
	var leitor io.Reader
	if corpo != nil {
		b, err := json.Marshal(corpo)
		if err != nil {
			return Resposta{}, err
		}
		leitor = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, metodo, g.Base+caminho, leitor)
	if err != nil {
		return Resposta{}, err
	}
	req.Header.Set("Authorization", "Bearer "+g.Token)
	if corpo != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if chaveIdempotencia != "" {
		req.Header.Set("Idempotency-Key", chaveIdempotencia)
	}
	inicio := time.Now()
	resp, err := g.Cliente.Do(req)
	if err != nil {
		if g.Registrar != nil {
			g.Registrar(fmt.Sprintf("%s %s %s → erro de transporte após %s: %v", Hora(), metodo, caminho, time.Since(inicio).Round(time.Millisecond), err))
		}
		return Resposta{Duracao: time.Since(inicio)}, err
	}
	defer resp.Body.Close()
	bruto, _ := io.ReadAll(resp.Body)
	r := Resposta{Status: resp.StatusCode, Bruto: bruto, Duracao: time.Since(inicio)}
	if len(bruto) > 0 {
		if err := json.Unmarshal(bruto, &r.Corpo); err != nil {
			return r, fmt.Errorf("%s %s: resposta %d não é JSON: %s", metodo, caminho, resp.StatusCode, bruto)
		}
	}
	if g.Registrar != nil {
		g.Registrar(fmt.Sprintf("%s %s %s → %d em %s", Hora(), metodo, caminho, r.Status, r.Duracao.Round(time.Millisecond)))
	}
	return r, nil
}

// ValidacaoAberta é o resultado de um POST /validacoes bem-sucedido.
type ValidacaoAberta struct {
	Id       string
	Maquina  string
	Resposta Resposta
	Corpo    map[string]any
	Chave    string
}

// ErrResposta marca um status diferente do esperado.
var ErrResposta = errors.New("resposta inesperada")

// IniciarValidacao abre uma validação pelo gateway. Se a máquina ficou com
// uma validação aberta de uma execução anterior (409), conclui-a e tenta de
// novo. Uma falha de status devolve ErrResposta com a resposta.
func (g *Gateway) IniciarValidacao(ctx context.Context, adesivo *Adesivo, maq string) (ValidacaoAberta, error) {
	corpo := CorpoValidacao(adesivo.Qr(maq))
	chave := NovoUuid()
	resp, err := g.Chamar(ctx, http.MethodPost, "/validacoes", corpo, chave)
	if err != nil {
		return ValidacaoAberta{Resposta: resp}, err
	}
	if resp.Status == http.StatusConflict && resp.Texto("codigo") == "validacao_em_andamento" {
		anterior := resp.Texto("idValidacao")
		if g.Registrar != nil {
			g.Registrar(fmt.Sprintf("%s máquina %s com validação %s aberta de execução anterior; concluindo para liberar", Hora(), maq, anterior))
		}
		if _, err := g.ConcluirValidacao(ctx, anterior); err != nil {
			return ValidacaoAberta{Resposta: resp}, err
		}
		chave = NovoUuid()
		if resp, err = g.Chamar(ctx, http.MethodPost, "/validacoes", corpo, chave); err != nil {
			return ValidacaoAberta{Resposta: resp}, err
		}
	}
	if resp.Status != http.StatusCreated || resp.Texto("idValidacao") == "" || resp.Texto("contrassenha") == "" {
		return ValidacaoAberta{Resposta: resp}, fmt.Errorf("%w: abrir validação em %s: %d %s", ErrResposta, maq, resp.Status, resp.Bruto)
	}
	return ValidacaoAberta{Id: resp.Texto("idValidacao"), Maquina: maq, Resposta: resp, Corpo: corpo, Chave: chave}, nil
}

// ConcluirValidacao conclui pelo gateway e exige 200 com status concluida.
func (g *Gateway) ConcluirValidacao(ctx context.Context, id string) (Resposta, error) {
	resp, err := g.Chamar(ctx, http.MethodPost, "/validacoes/"+id+"/conclusao", CorpoConclusao(), "")
	if err != nil {
		return resp, err
	}
	if resp.Status != http.StatusOK || resp.Texto("status") != "concluida" {
		return resp, fmt.Errorf("%w: concluir %s: %d %s", ErrResposta, id, resp.Status, resp.Bruto)
	}
	return resp, nil
}

// --- utilidades ------------------------------------------------------------------

// NovoUuid gera um UUID v4 para a Idempotency-Key.
func NovoUuid() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// Hora formata o instante atual para as linhas de registro.
func Hora() string { return time.Now().Format("15:04:05.000") }

// Esperar consulta cond a cada intervalo até ser verdadeira ou o prazo vencer.
// Devolve quanto tempo levou e se a condição foi satisfeita.
func Esperar(ctx context.Context, prazo, intervalo time.Duration, cond func() bool) (time.Duration, bool) {
	inicio := time.Now()
	for {
		if cond() {
			return time.Since(inicio), true
		}
		if time.Since(inicio) >= prazo || ctx.Err() != nil {
			return time.Since(inicio), false
		}
		select {
		case <-ctx.Done():
			return time.Since(inicio), false
		case <-time.After(intervalo):
		}
	}
}
