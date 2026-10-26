//go:build e2e

package e2e

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
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"cloud.google.com/go/pubsub/v2"

	"github.com/ThiagoCristovao/serverless-vending-validation/sistema-central-simulado/internal/admin"
	"github.com/ThiagoCristovao/serverless-vending-validation/sistema-central-simulado/internal/armazenamento"
	"github.com/ThiagoCristovao/serverless-vending-validation/sistema-central-simulado/internal/consumidor"
	"github.com/ThiagoCristovao/serverless-vending-validation/sistema-central-simulado/internal/falhas"
)

// configuracao vem do ambiente (make e2e preenche a partir do Terraform e de chaves/).
type configuracao struct {
	Projeto       string
	Gateway       string // base com /v1, ex.: https://host/v1
	Prefixo       string // svv-dev → svv-dev-validacoes, svv-dev-sistema-central, svv-dev-validacoes-dlq-inspecao
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

func (c configuracao) topico() string        { return c.Prefixo + "-validacoes" }
func (c configuracao) assinatura() string    { return c.Prefixo + "-sistema-central" }
func (c configuracao) assinaturaDlq() string { return c.Prefixo + "-validacoes-dlq-inspecao" }

func valorOu(nome, padrao string) string {
	if v := os.Getenv(nome); v != "" {
		return v
	}
	return padrao
}

func carregarConfiguracao() (configuracao, error) {
	c := configuracao{
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
		return c, errors.New("SVV_E2E_GATEWAY não definido (use make e2e)")
	}
	if c.ApiKey == "" {
		if gs := os.Getenv("SVV_E2E_GOOGLE_SERVICES"); gs != "" {
			chave, err := chaveDoGoogleServices(gs)
			if err != nil {
				return c, err
			}
			c.ApiKey = chave
		}
	}
	var faltando []string
	for nome, valor := range map[string]string{"SVV_E2E_PROJETO": c.Projeto, "SVV_E2E_API_KEY_WEB (ou SVV_E2E_GOOGLE_SERVICES)": c.ApiKey, "SVV_E2E_OPERADOR_EMAIL": c.OperadorEmail, "SVV_E2E_OPERADOR_SENHA": c.OperadorSenha, "SVV_E2E_CHAVE_PRIVADA": c.ChavePrivada} {
		if valor == "" {
			faltando = append(faltando, nome)
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
		return "", fmt.Errorf("%s: %w", caminho, err)
	}
	if len(gs.Client) == 0 || len(gs.Client[0].ApiKey) == 0 {
		return "", fmt.Errorf("%s: sem client[0].api_key[0].current_key", caminho)
	}
	return gs.Client[0].ApiKey[0].CurrentKey, nil
}

func carregarPrivada(valor string) (ed25519.PrivateKey, error) {
	if conteudo, err := os.ReadFile(valor); err == nil {
		valor = strings.TrimSpace(string(conteudo))
	}
	bytes, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(valor))
	if err != nil || len(bytes) != ed25519.PrivateKeySize {
		return nil, errors.New("chave privada deve ser Ed25519 (64 bytes) em base64url")
	}
	return ed25519.PrivateKey(bytes), nil
}

// ambiente reúne o consumidor em processo (com registro em arquivo temporário e
// API administrativa em porta efêmera), o cliente Pub/Sub e a sessão do operador.
type ambiente struct {
	cfg         configuracao
	ctx         context.Context
	cancelar    context.CancelFunc
	http        *http.Client
	token       string
	uid         string
	privada     ed25519.PrivateKey
	cli         *pubsub.Client
	registro    *armazenamento.Registro
	interruptor *falhas.Interruptor
	proc        *consumidor.Processador
	cons        *consumidor.Consumidor
	admin       *httptest.Server
	diretorio   string
}

var (
	amb          *ambiente
	motivoPular  string
	prepararUma  sync.Once
	erroPreparar error
)

func TestMain(m *testing.M) {
	cfg, err := carregarConfiguracao()
	if err != nil {
		motivoPular = err.Error()
	}
	codigo := m.Run()
	if amb != nil {
		amb.encerrar()
	}
	if err == nil && amb != nil {
		est := amb.proc.Estatisticas()
		fmt.Printf("\nresumo do consumidor: recebidas=%d registradas=%d duplicadas=%d rejeitadas=%d invalidas=%d confirmacoesPerdidas=%d\nregistro: %s\n",
			est.Recebidas, est.Registradas, est.Duplicadas, est.Rejeitadas, est.Invalidas, est.ConfirmacoesPerdidas, amb.registro.Caminho())
	}
	_ = cfg
	os.Exit(codigo)
}

// exigirAmbiente prepara o ambiente uma vez; pula o teste se a configuração faltar.
func exigirAmbiente(t *testing.T) *ambiente {
	t.Helper()
	if motivoPular != "" {
		t.Skip("e2e não configurado: " + motivoPular)
	}
	prepararUma.Do(func() {
		cfg, _ := carregarConfiguracao()
		amb, erroPreparar = prepararAmbiente(cfg)
	})
	if erroPreparar != nil {
		t.Fatalf("preparação do ambiente: %v", erroPreparar)
	}
	return amb
}

func prepararAmbiente(cfg configuracao) (*ambiente, error) {
	ctx, cancelar := context.WithCancel(context.Background())
	a := &ambiente{cfg: cfg, ctx: ctx, cancelar: cancelar, http: &http.Client{Timeout: 30 * time.Second}}

	privada, err := carregarPrivada(cfg.ChavePrivada)
	if err != nil {
		return nil, err
	}
	a.privada = privada

	a.token, a.uid, err = obterToken(ctx, cfg.ApiKey, cfg.OperadorEmail, cfg.OperadorSenha)
	if err != nil {
		return nil, fmt.Errorf("token do operador: %w", err)
	}

	a.cli, err = pubsub.NewClient(ctx, cfg.Projeto)
	if err != nil {
		return nil, fmt.Errorf("cliente Pub/Sub: %w", err)
	}

	a.diretorio, err = os.MkdirTemp("", "svv-e2e-")
	if err != nil {
		return nil, err
	}
	a.registro, err = armazenamento.Abrir(filepath.Join(a.diretorio, "eventos.jsonl"))
	if err != nil {
		return nil, err
	}
	log := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	a.interruptor = falhas.Novo()
	a.proc = consumidor.NovoProcessador(a.registro, a.interruptor, log)
	a.cons = consumidor.Novo(a.cli, cfg.assinatura(), a.proc, log)
	if err := a.cons.Iniciar(ctx); err != nil {
		return nil, err
	}
	a.admin = httptest.NewServer(admin.NovoHandler(admin.Dependencias{Contexto: ctx, Registro: a.registro, Processador: a.proc, Interruptor: a.interruptor, Consumidor: a.cons, Log: log}))
	return a, nil
}

func (a *ambiente) encerrar() {
	a.cons.Pausar()
	a.admin.Close()
	_ = a.registro.Fechar()
	_ = a.cli.Close()
	a.cancelar()
}

// --- Firebase Authentication ---------------------------------------------------

func obterToken(ctx context.Context, apiKey, email, senha string) (idToken, uid string, err error) {
	corpo, _ := json.Marshal(map[string]any{"email": email, "password": senha, "returnSecureToken": true})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, "https://identitytoolkit.googleapis.com/v1/accounts:signInWithPassword?key="+apiKey, bytes.NewReader(corpo))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", err
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
		return "", "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("identitytoolkit %d: %s", resp.StatusCode, dados.Error.Message)
	}
	return dados.IdToken, dados.LocalId, nil
}

// --- Gateway -------------------------------------------------------------------

func (a *ambiente) qrAssinado(maq string) map[string]any {
	exp := time.Now().UTC().AddDate(0, 6, 0).Format("2006-01-02")
	canonica := fmt.Sprintf("%d|%s|%s|%s|%s|%s", 1, maq, a.cfg.Modelo, a.cfg.Loc, exp, a.cfg.Kid)
	sig := ed25519.Sign(a.privada, []byte(canonica))
	return map[string]any{"v": 1, "maq": maq, "mod": a.cfg.Modelo, "loc": a.cfg.Loc, "exp": exp, "kid": a.cfg.Kid, "sig": base64.RawURLEncoding.EncodeToString(sig)}
}

func (a *ambiente) chamarGateway(t *testing.T, metodo, caminho string, corpo any, chaveIdempotencia string) (int, map[string]any) {
	t.Helper()
	var leitor io.Reader
	if corpo != nil {
		b, err := json.Marshal(corpo)
		if err != nil {
			t.Fatal(err)
		}
		leitor = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(a.ctx, metodo, a.cfg.Gateway+caminho, leitor)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+a.token)
	if corpo != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if chaveIdempotencia != "" {
		req.Header.Set("Idempotency-Key", chaveIdempotencia)
	}
	inicio := time.Now()
	resp, err := a.http.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", metodo, caminho, err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	var dados map[string]any
	if len(b) > 0 {
		if err := json.Unmarshal(b, &dados); err != nil {
			t.Fatalf("%s %s: resposta %d não é JSON: %s", metodo, caminho, resp.StatusCode, b)
		}
	}
	t.Logf("%s %s %s → %d em %s", hora(), metodo, caminho, resp.StatusCode, time.Since(inicio).Round(time.Millisecond))
	return resp.StatusCode, dados
}

type validacaoAberta struct {
	Id       string
	Resposta map[string]any
	Corpo    map[string]any
	Chave    string
}

// iniciarValidacao abre uma validação pelo gateway. Se a máquina ficou com uma
// validação aberta de uma execução anterior (409), conclui-a e tenta de novo.
func (a *ambiente) iniciarValidacao(t *testing.T, maq string) validacaoAberta {
	t.Helper()
	corpo := map[string]any{
		"qr":          a.qrAssinado(maq),
		"lidoEm":      time.Now().UTC().Format(time.RFC3339),
		"dispositivo": map[string]any{"plataforma": "android", "versaoApp": "e2e"},
	}
	chave := uuid4()
	status, resp := a.chamarGateway(t, http.MethodPost, "/validacoes", corpo, chave)
	if status == http.StatusConflict && resp["codigo"] == "validacao_em_andamento" {
		anterior, _ := resp["idValidacao"].(string)
		t.Logf("máquina %s com validação %s aberta de execução anterior; concluindo para liberar", maq, anterior)
		a.concluirValidacao(t, anterior)
		chave = uuid4()
		status, resp = a.chamarGateway(t, http.MethodPost, "/validacoes", corpo, chave)
	}
	if status != http.StatusCreated {
		t.Fatalf("abrir validação em %s: %d %v", maq, status, resp)
	}
	id, _ := resp["idValidacao"].(string)
	if id == "" || resp["contrassenha"] == "" {
		t.Fatalf("resposta sem idValidacao/contrassenha: %v", resp)
	}
	return validacaoAberta{Id: id, Resposta: resp, Corpo: corpo, Chave: chave}
}

func (a *ambiente) concluirValidacao(t *testing.T, id string) map[string]any {
	t.Helper()
	// Medidor crescente no tempo evita o aviso de RN-07 entre execuções.
	corpo := map[string]any{
		"leituras":          map[string]any{"medidor": time.Now().Unix(), "unidadesVendidas": 7},
		"codigoRotacionado": true,
		"observacoes":       "suíte e2e",
	}
	status, resp := a.chamarGateway(t, http.MethodPost, "/validacoes/"+id+"/conclusao", corpo, "")
	if status != http.StatusOK || resp["status"] != "concluida" {
		t.Fatalf("concluir %s: %d %v", id, status, resp)
	}
	return resp
}

// --- API administrativa do consumidor ------------------------------------------

func (a *ambiente) eventos(t *testing.T, idValidacao string) []armazenamento.Item {
	t.Helper()
	resp, err := a.http.Get(a.admin.URL + "/eventos?idValidacao=" + idValidacao)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var dados struct {
		Itens []armazenamento.Item `json:"itens"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&dados); err != nil {
		t.Fatal(err)
	}
	return dados.Itens
}

func (a *ambiente) estatisticas(t *testing.T) consumidor.Estatisticas {
	t.Helper()
	resp, err := a.http.Get(a.admin.URL + "/estatisticas")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var est consumidor.Estatisticas
	if err := json.NewDecoder(resp.Body).Decode(&est); err != nil {
		t.Fatal(err)
	}
	return est
}

func (a *ambiente) definirFalhas(t *testing.T, cfg falhas.Configuracao) {
	t.Helper()
	corpo, _ := json.Marshal(cfg)
	req, _ := http.NewRequest(http.MethodPut, a.admin.URL+"/admin/falhas", bytes.NewReader(corpo))
	resp, err := a.http.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("definir falhas: %d %s", resp.StatusCode, b)
	}
	t.Logf("%s interruptor de falhas: modo=%s idMaquina=%q", hora(), cfg.Modo, cfg.IdMaquina)
}

// --- Pub/Sub -------------------------------------------------------------------

// publicarCopia reenvia ao tópico uma mensagem já consumida, com o mesmo corpo,
// atributos e chave de ordenação: simula a entrega repetida do at-least-once.
func (a *ambiente) publicarCopia(t *testing.T, item armazenamento.Item) {
	t.Helper()
	pub := a.cli.Publisher(a.cfg.topico())
	pub.EnableMessageOrdering = true
	defer pub.Stop()
	res := pub.Publish(a.ctx, &pubsub.Message{Data: item.Corpo, Attributes: item.Atributos, OrderingKey: item.IdMaquina})
	id, err := res.Get(a.ctx)
	if err != nil {
		t.Fatalf("publicar cópia: %v", err)
	}
	t.Logf("%s cópia de %s/%s publicada (mensagem %s)", hora(), item.IdValidacao, item.Tipo, id)
}

// drenarDlq confirma tudo o que estiver na assinatura de inspeção da DLQ.
func (a *ambiente) drenarDlq(t *testing.T) {
	t.Helper()
	ctx, cancelar := context.WithTimeout(a.ctx, 5*time.Second)
	defer cancelar()
	n := 0
	_ = a.cli.Subscriber(a.cfg.assinaturaDlq()).Receive(ctx, func(_ context.Context, m *pubsub.Message) { m.Ack(); n++ })
	t.Logf("%s DLQ drenada: %d mensagem(ns) antiga(s) confirmada(s)", hora(), n)
}

// esperarNaDlq recebe da assinatura de inspeção até achar a validação.
func (a *ambiente) esperarNaDlq(t *testing.T, idValidacao string, prazo time.Duration) *pubsub.Message {
	t.Helper()
	ctx, cancelar := context.WithTimeout(a.ctx, prazo)
	defer cancelar()
	var achada *pubsub.Message
	_ = a.cli.Subscriber(a.cfg.assinaturaDlq()).Receive(ctx, func(_ context.Context, m *pubsub.Message) {
		m.Ack()
		if m.Attributes["idValidacao"] == idValidacao {
			achada = m
			cancelar()
		}
	})
	if achada == nil {
		t.Fatalf("mensagem de %s não chegou à DLQ em %s", idValidacao, prazo)
	}
	return achada
}

// --- utilidades ----------------------------------------------------------------

func esperar(t *testing.T, descricao string, prazo time.Duration, cond func() bool) {
	t.Helper()
	inicio := time.Now()
	for time.Since(inicio) < prazo {
		if cond() {
			t.Logf("%s %s (após %s)", hora(), descricao, time.Since(inicio).Round(time.Millisecond))
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Fatalf("tempo esgotado (%s) esperando: %s", prazo, descricao)
}

func hora() string { return time.Now().Format("15:04:05.000") }

func uuid4() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
