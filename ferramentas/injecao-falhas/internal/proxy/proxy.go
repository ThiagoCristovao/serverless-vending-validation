// Package proxy é um proxy HTTP reverso que injeta falhas de rede entre o
// aplicativo e o API Gateway (Sprint 9, cenário "falhas intermitentes de rede
// no dispositivo"). O celular aponta para ele por --dart-define=SVV_API_BASE.
package proxy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/rand/v2"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Modo de falha.
type Modo string

const (
	// ModoNormal encaminha tudo.
	ModoNormal Modo = "normal"
	// ModoCortar fecha a conexão sem responder (queda de rede antes do envio).
	ModoCortar Modo = "cortar"
	// ModoAtrasar segura a resposta por Atraso (rede lenta; estoura o tempo
	// limite de 20 s do aplicativo quando Atraso é maior).
	ModoAtrasar Modo = "atrasar"
	// ModoIndisponivel responde 503 dependencia_indisponivel sem encaminhar.
	ModoIndisponivel Modo = "indisponivel"
	// ModoCortarResposta encaminha ao gateway, espera a resposta e fecha a
	// conexão sem devolvê-la: o serviço processou, o aplicativo não soube (CE-03).
	ModoCortarResposta Modo = "cortar-resposta"
	// ModoIntermitente alterna: uma requisição cortada, uma normal.
	ModoIntermitente Modo = "intermitente"
)

// Configuracao corrente do proxy.
type Configuracao struct {
	Modo Modo `json:"modo"`
	// Proporcao das requisições afetadas (0 a 1) nos modos cortar, atrasar,
	// indisponivel e cortar-resposta.
	Proporcao float64 `json:"proporcao"`
	// Atraso do modo atrasar.
	Atraso time.Duration `json:"-"`
}

type configuracaoJson struct {
	Modo      Modo    `json:"modo"`
	Proporcao float64 `json:"proporcao"`
	Atraso    string  `json:"atraso,omitempty"`
}

// MarshalJSON representa o atraso como texto ("25s").
func (c Configuracao) MarshalJSON() ([]byte, error) {
	return json.Marshal(configuracaoJson{Modo: c.Modo, Proporcao: c.Proporcao, Atraso: c.Atraso.String()})
}

// UnmarshalJSON aceita o atraso como texto ("25s").
func (c *Configuracao) UnmarshalJSON(dados []byte) error {
	var j configuracaoJson
	if err := json.Unmarshal(dados, &j); err != nil {
		return err
	}
	c.Modo, c.Proporcao = j.Modo, j.Proporcao
	if j.Atraso != "" {
		d, err := time.ParseDuration(j.Atraso)
		if err != nil {
			return fmt.Errorf("atraso: %w", err)
		}
		c.Atraso = d
	}
	return nil
}

// Validar confere o modo e a proporção.
func (c Configuracao) Validar() error {
	switch c.Modo {
	case ModoNormal, ModoCortar, ModoAtrasar, ModoIndisponivel, ModoCortarResposta, ModoIntermitente:
	default:
		return fmt.Errorf("modo desconhecido %q (use normal, cortar, atrasar, indisponivel, cortar-resposta ou intermitente)", c.Modo)
	}
	if c.Proporcao < 0 || c.Proporcao > 1 {
		return errors.New("proporcao deve estar entre 0 e 1")
	}
	return nil
}

// Proxy encaminha requisições ao destino aplicando o modo configurado.
type Proxy struct {
	destino *url.URL
	cliente *http.Client
	log     *slog.Logger

	mu       sync.Mutex
	cfg      Configuracao
	contador uint64
	sorteio  func() float64
}

// Novo cria o proxy para o destino (ex.: https://host-do-gateway).
func Novo(destino string, cfg Configuracao, log *slog.Logger) (*Proxy, error) {
	u, err := url.Parse(destino)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return nil, fmt.Errorf("destino inválido %q", destino)
	}
	if cfg.Modo == "" {
		cfg.Modo = ModoNormal
	}
	if cfg.Proporcao == 0 {
		cfg.Proporcao = 1
	}
	if err := cfg.Validar(); err != nil {
		return nil, err
	}
	if log == nil {
		log = slog.Default()
	}
	return &Proxy{destino: u, cliente: &http.Client{Timeout: 60 * time.Second}, log: log, cfg: cfg, sorteio: rand.Float64}, nil
}

// Definir substitui a configuração.
func (p *Proxy) Definir(c Configuracao) error {
	if err := c.Validar(); err != nil {
		return err
	}
	p.mu.Lock()
	p.cfg = c
	p.contador = 0
	p.mu.Unlock()
	p.log.Warn("modo do proxy alterado", "modo", c.Modo, "proporcao", c.Proporcao, "atraso", c.Atraso.String())
	return nil
}

// Atual devolve a configuração corrente.
func (p *Proxy) Atual() Configuracao {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.cfg
}

// decidir escolhe o modo aplicado a esta requisição.
func (p *Proxy) decidir() (Modo, Configuracao) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.contador++
	switch p.cfg.Modo {
	case ModoNormal:
		return ModoNormal, p.cfg
	case ModoIntermitente:
		if p.contador%2 == 1 {
			return ModoCortar, p.cfg
		}
		return ModoNormal, p.cfg
	}
	if p.sorteio() < p.cfg.Proporcao {
		return p.cfg.Modo, p.cfg
	}
	return ModoNormal, p.cfg
}

// Handler monta as rotas: /admin/modo (GET/PUT) e o encaminhamento do resto.
func (p *Proxy) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /admin/modo", func(w http.ResponseWriter, _ *http.Request) { responderJson(w, http.StatusOK, p.Atual()) })
	mux.HandleFunc("PUT /admin/modo", func(w http.ResponseWriter, r *http.Request) {
		var c Configuracao
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&c); err != nil {
			responderJson(w, http.StatusBadRequest, map[string]string{"erro": err.Error()})
			return
		}
		if c.Proporcao == 0 && c.Modo != ModoNormal {
			c.Proporcao = 1
		}
		if err := p.Definir(c); err != nil {
			responderJson(w, http.StatusBadRequest, map[string]string{"erro": err.Error()})
			return
		}
		responderJson(w, http.StatusOK, p.Atual())
	})
	mux.HandleFunc("/", p.encaminhar)
	return mux
}

func (p *Proxy) encaminhar(w http.ResponseWriter, r *http.Request) {
	inicio := time.Now()
	modo, cfg := p.decidir()
	campos := []any{"metodo", r.Method, "caminho", r.URL.Path, "modoAplicado", string(modo)}
	if chave := r.Header.Get("Idempotency-Key"); chave != "" {
		// Evidência de que os reenvios do aplicativo repetem a mesma chave (RF-23).
		campos = append(campos, "chaveIdempotencia", chave)
	}

	switch modo {
	case ModoCortar:
		p.cortar(w)
		p.log.Warn("conexão cortada antes do envio", campos...)
		return
	case ModoIndisponivel:
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"type": "urn:svv:erro:dependencia_indisponivel", "title": "Serviço indisponível", "status": 503,
			"detail": "falha injetada pelo proxy de rede", "instance": r.URL.Path, "codigo": "dependencia_indisponivel",
		})
		p.log.Warn("503 injetado", campos...)
		return
	case ModoAtrasar:
		select {
		case <-r.Context().Done():
			p.log.Warn("cliente desistiu durante o atraso", append(campos, "apos", time.Since(inicio).Round(time.Millisecond).String())...)
			return
		case <-time.After(cfg.Atraso):
		}
	}

	resp, err := p.repassar(r)
	if err != nil {
		http.Error(w, "proxy: "+err.Error(), http.StatusBadGateway)
		p.log.Error("falha ao encaminhar", append(campos, "erro", err.Error())...)
		return
	}
	defer resp.Body.Close()
	corpo, _ := io.ReadAll(resp.Body)
	campos = append(campos, "status", resp.StatusCode, "duracao", time.Since(inicio).Round(time.Millisecond).String())

	if modo == ModoCortarResposta {
		p.cortar(w)
		p.log.Warn("resposta do gateway descartada; conexão cortada", campos...)
		return
	}
	for chave, valores := range resp.Header {
		if hopByHop(chave) {
			continue
		}
		for _, v := range valores {
			w.Header().Add(chave, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = w.Write(corpo)
	p.log.Info("requisição encaminhada", campos...)
}

// repassar reenvia a requisição ao destino, preservando método, caminho,
// consulta, corpo e cabeçalhos relevantes.
func (p *Proxy) repassar(r *http.Request) (*http.Response, error) {
	alvo := *p.destino
	alvo.Path = strings.TrimRight(p.destino.Path, "/") + r.URL.Path
	alvo.RawQuery = r.URL.RawQuery
	req, err := http.NewRequestWithContext(context.WithoutCancel(r.Context()), r.Method, alvo.String(), r.Body)
	if err != nil {
		return nil, err
	}
	for chave, valores := range r.Header {
		if hopByHop(chave) || strings.EqualFold(chave, "Host") {
			continue
		}
		req.Header[chave] = valores
	}
	req.Host = p.destino.Host
	return p.cliente.Do(req)
}

func (p *Proxy) cortar(w http.ResponseWriter) {
	h, ok := w.(http.Hijacker)
	if !ok {
		panic("servidor sem suporte a hijack")
	}
	conn, _, err := h.Hijack()
	if err != nil {
		return
	}
	if tcp, ok := conn.(*net.TCPConn); ok {
		_ = tcp.SetLinger(0) // RST em vez de FIN: parece queda de rede
	}
	_ = conn.Close()
}

func hopByHop(chave string) bool {
	switch strings.ToLower(chave) {
	case "connection", "keep-alive", "proxy-authenticate", "proxy-authorization", "te", "trailer", "transfer-encoding", "upgrade", "content-length", "accept-encoding":
		return true
	}
	return false
}

func responderJson(w http.ResponseWriter, status int, corpo any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(corpo)
}
