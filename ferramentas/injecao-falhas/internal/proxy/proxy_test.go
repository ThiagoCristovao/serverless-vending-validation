package proxy_test

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ThiagoCristovao/serverless-vending-validation/ferramentas/injecao-falhas/internal/proxy"
)

var silencioso = slog.New(slog.NewTextHandler(io.Discard, nil))

type ambiente struct {
	gateway   *httptest.Server
	proxy     *httptest.Server
	p         *proxy.Proxy
	recebidas atomic.Int64
	ultima    atomic.Pointer[http.Request]
}

func novoAmbiente(t *testing.T, cfg proxy.Configuracao) *ambiente {
	t.Helper()
	a := &ambiente{}
	a.gateway = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		a.recebidas.Add(1)
		copia := r.Clone(r.Context())
		a.ultima.Store(copia)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"idValidacao":"01J8ZK3V9Q7XW2N4M6P8R0T2Y4","caminho":"` + r.URL.Path + `"}`))
	}))
	t.Cleanup(a.gateway.Close)
	p, err := proxy.Novo(a.gateway.URL, cfg, silencioso)
	if err != nil {
		t.Fatal(err)
	}
	a.p = p
	a.proxy = httptest.NewServer(p.Handler())
	t.Cleanup(a.proxy.Close)
	return a
}

func chamar(t *testing.T, url string) (*http.Response, error) {
	t.Helper()
	req, _ := http.NewRequest(http.MethodPost, url, strings.NewReader(`{"qr":{}}`))
	req.Header.Set("Authorization", "Bearer token")
	req.Header.Set("Idempotency-Key", "chave-1")
	req.Header.Set("Content-Type", "application/json")
	// Sem keep-alive: o cliente padrão do Go reenvia em silêncio uma requisição
	// cuja conexão reaproveitada foi fechada, o que esconderia o corte.
	cliente := &http.Client{Timeout: 5 * time.Second, Transport: &http.Transport{DisableKeepAlives: true}}
	return cliente.Do(req)
}

func TestNormalEncaminhaComCabecalhos(t *testing.T) {
	a := novoAmbiente(t, proxy.Configuracao{Modo: proxy.ModoNormal})
	resp, err := chamar(t, a.proxy.URL+"/v1/validacoes?x=1")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	corpo, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated || !strings.Contains(string(corpo), `"caminho":"/v1/validacoes"`) || resp.Header.Get("Content-Type") != "application/json" {
		t.Fatalf("resposta: %d %s %v", resp.StatusCode, corpo, resp.Header)
	}
	u := a.ultima.Load()
	if u.Header.Get("Authorization") != "Bearer token" || u.Header.Get("Idempotency-Key") != "chave-1" || u.URL.RawQuery != "x=1" {
		t.Fatalf("cabeçalhos no gateway: %v %s", u.Header, u.URL.RawQuery)
	}
}

func TestIndisponivelRespondeProblemaSemEncaminhar(t *testing.T) {
	a := novoAmbiente(t, proxy.Configuracao{Modo: proxy.ModoIndisponivel, Proporcao: 1})
	resp, err := chamar(t, a.proxy.URL+"/v1/validacoes")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var problema map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&problema)
	if resp.StatusCode != 503 || problema["codigo"] != "dependencia_indisponivel" || resp.Header.Get("Content-Type") != "application/problem+json" {
		t.Fatalf("resposta: %d %v", resp.StatusCode, problema)
	}
	if a.recebidas.Load() != 0 {
		t.Fatal("gateway não deveria ter sido chamado")
	}
}

func TestCortarFechaAConexaoSemEncaminhar(t *testing.T) {
	a := novoAmbiente(t, proxy.Configuracao{Modo: proxy.ModoCortar, Proporcao: 1})
	if _, err := chamar(t, a.proxy.URL+"/v1/validacoes"); err == nil {
		t.Fatal("esperava erro de transporte")
	}
	if a.recebidas.Load() != 0 {
		t.Fatal("gateway não deveria ter sido chamado")
	}
}

func TestCortarRespostaEncaminhaEDescarta(t *testing.T) {
	a := novoAmbiente(t, proxy.Configuracao{Modo: proxy.ModoCortarResposta, Proporcao: 1})
	if _, err := chamar(t, a.proxy.URL+"/v1/validacoes/1/conclusao"); err == nil {
		t.Fatal("esperava erro de transporte")
	}
	if a.recebidas.Load() != 1 {
		t.Fatalf("gateway deveria ter processado 1 requisição, processou %d", a.recebidas.Load())
	}
}

func TestAtrasarSeguraAResposta(t *testing.T) {
	a := novoAmbiente(t, proxy.Configuracao{Modo: proxy.ModoAtrasar, Proporcao: 1, Atraso: 300 * time.Millisecond})
	inicio := time.Now()
	resp, err := chamar(t, a.proxy.URL+"/v1/validacoes")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if d := time.Since(inicio); d < 300*time.Millisecond || resp.StatusCode != http.StatusCreated {
		t.Fatalf("atraso %s, status %d", d, resp.StatusCode)
	}
}

func TestIntermitenteAlterna(t *testing.T) {
	a := novoAmbiente(t, proxy.Configuracao{Modo: proxy.ModoIntermitente})
	var resultados []bool
	for i := 0; i < 4; i++ {
		resp, err := chamar(t, a.proxy.URL+"/v1/validacoes")
		if err == nil {
			resp.Body.Close()
		}
		resultados = append(resultados, err == nil)
	}
	if resultados[0] || !resultados[1] || resultados[2] || !resultados[3] {
		t.Fatalf("esperava cortada, ok, cortada, ok; veio %v", resultados)
	}
	if a.recebidas.Load() != 2 {
		t.Fatalf("gateway recebeu %d", a.recebidas.Load())
	}
}

func TestProporcaoZeroNaoAfeta(t *testing.T) {
	a := novoAmbiente(t, proxy.Configuracao{Modo: proxy.ModoCortar, Proporcao: 0.0000001})
	for i := 0; i < 5; i++ {
		resp, err := chamar(t, a.proxy.URL+"/v1/validacoes")
		if err != nil {
			t.Fatalf("requisição %d afetada com proporção ~0: %v", i, err)
		}
		resp.Body.Close()
	}
}

func TestAdminModo(t *testing.T) {
	a := novoAmbiente(t, proxy.Configuracao{Modo: proxy.ModoNormal})
	req, _ := http.NewRequest(http.MethodPut, a.proxy.URL+"/admin/modo", bytes.NewReader([]byte(`{"modo":"atrasar","atraso":"25s","proporcao":0.5}`)))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	var cfg proxy.Configuracao
	_ = json.NewDecoder(resp.Body).Decode(&cfg)
	resp.Body.Close()
	if resp.StatusCode != 200 || cfg.Modo != proxy.ModoAtrasar || cfg.Atraso != 25*time.Second || cfg.Proporcao != 0.5 {
		t.Fatalf("definir: %d %+v", resp.StatusCode, cfg)
	}
	req, _ = http.NewRequest(http.MethodPut, a.proxy.URL+"/admin/modo", bytes.NewReader([]byte(`{"modo":"explodir"}`)))
	resp, _ = http.DefaultClient.Do(req)
	resp.Body.Close()
	if resp.StatusCode != 400 {
		t.Fatalf("modo inválido aceito: %d", resp.StatusCode)
	}
	resp, _ = http.Get(a.proxy.URL + "/admin/modo")
	_ = json.NewDecoder(resp.Body).Decode(&cfg)
	resp.Body.Close()
	if cfg.Modo != proxy.ModoAtrasar {
		t.Fatalf("atual: %+v", cfg)
	}
}
