package apihttp_test

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ThiagoCristovao/serverless-vending-validation/servico/internal/adaptadores/apihttp"
	"github.com/ThiagoCristovao/serverless-vending-validation/servico/internal/adaptadores/memoria"
	"github.com/ThiagoCristovao/serverless-vending-validation/servico/internal/adaptadores/sistema"
	"github.com/ThiagoCristovao/serverless-vending-validation/servico/internal/aplicacao"
	"github.com/ThiagoCristovao/serverless-vending-validation/servico/internal/dominio"
)

var inicio = time.Date(2026, 9, 17, 13, 5, 12, 0, time.UTC)

const chaveIdempotencia = "6f1c2a4e-3b7d-4e8f-9a0b-1c2d3e4f5a6b"

type banco struct {
	t       *testing.T
	srv     http.Handler
	rel     *memoria.Relogio
	pub     *memoria.Publicador
	privada ed25519.PrivateKey
}

func novoBanco(t *testing.T) *banco {
	t.Helper()
	publica, privada, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	arm := memoria.Novo()
	arm.SemearMaquina(dominio.Maquina{Id: "VM-2047", Modelo: "CN168", Localizacao: dominio.Localizacao{Id: "BLA-T", Nome: "Bloco A - Térreo"}, Ativa: true, ContadorCodigo: 12})
	arm.SemearOperador(dominio.Operador{Uid: "op-1", Nome: "Operador Um", Ativo: true})
	arm.SemearOperador(dominio.Operador{Uid: "op-2", Nome: "Operador Dois", Ativo: true})
	arm.SemearChaveQr(dominio.ChaveQr{Kid: "k1", ChavePublica: publica, Ativa: true})
	rel := memoria.NovoRelogio(inicio)
	pub := &memoria.Publicador{}
	gerador, _ := dominio.NovoGeradorContrassenha([]byte("0123456789abcdef0123456789abcdef"))
	servico, err := aplicacao.Novo(aplicacao.Dependencias{
		Maquinas: arm, Operadores: arm, Chaves: arm, Validacoes: arm,
		Publicador: pub, Relogio: rel, Ids: sistema.GeradorUlid{Relogio: rel}, Contrassenha: gerador,
		Registrador: slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	if err != nil {
		t.Fatal(err)
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	return &banco{t: t, rel: rel, pub: pub, privada: privada,
		srv: apihttp.NovoManipulador(servico, apihttp.Opcoes{Registrador: log, ProjetoId: "svv-teste", CabecalhoOperadorDeTeste: "X-Operador-Teste"})}
}

func (b *banco) payload(exp string) json.RawMessage {
	p := dominio.PayloadQr{V: 1, Maq: "VM-2047", Mod: "CN168", Loc: "BLA-T", Exp: exp, Kid: "k1"}
	p.Assinar(b.privada)
	bruto, _ := json.Marshal(p)
	return bruto
}

func userinfo(uid string) string {
	claims, _ := json.Marshal(map[string]any{"sub": uid, "email": uid + "@exemplo.invalid", "iss": "https://securetoken.google.com/svv-teste"})
	return base64.RawURLEncoding.EncodeToString(claims)
}

type resposta struct {
	status int
	tipo   string
	corpo  map[string]any
}

func (b *banco) chamar(metodo, caminho string, corpo any, cabecalhos map[string]string) resposta {
	b.t.Helper()
	var leitor io.Reader
	if corpo != nil {
		dados, _ := json.Marshal(corpo)
		leitor = bytes.NewReader(dados)
	}
	req := httptest.NewRequest(metodo, caminho, leitor)
	req.Header.Set("Content-Type", "application/json")
	for k, v := range cabecalhos {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	b.srv.ServeHTTP(rec, req)
	var decodificado map[string]any
	if rec.Body.Len() > 0 {
		if err := json.Unmarshal(rec.Body.Bytes(), &decodificado); err != nil {
			b.t.Fatalf("resposta não é JSON: %s", rec.Body.String())
		}
	}
	return resposta{status: rec.Code, tipo: rec.Header().Get("Content-Type"), corpo: decodificado}
}

func (b *banco) iniciar(uid string, chave string) resposta {
	cab := map[string]string{apihttp.CabecalhoUserinfo: userinfo(uid)}
	if chave != "" {
		cab["Idempotency-Key"] = chave
	}
	return b.chamar(http.MethodPost, "/v1/validacoes", map[string]any{"qr": b.payload("2027-12-31"), "dispositivo": map[string]string{"plataforma": "android", "versaoApp": "0.1.0"}}, cab)
}

func esperaProblema(t *testing.T, r resposta, status int, codigo string) {
	t.Helper()
	if r.status != status || !strings.HasPrefix(r.tipo, "application/problem+json") || r.corpo["codigo"] != codigo {
		t.Fatalf("status=%d tipo=%s corpo=%v; esperava %d %s", r.status, r.tipo, r.corpo, status, codigo)
	}
	if r.corpo["type"] != "urn:svv:erro:"+codigo || r.corpo["status"] != float64(status) {
		t.Fatalf("problem details inconsistente: %v", r.corpo)
	}
}

func TestFluxoPeloContrato(t *testing.T) {
	b := novoBanco(t)

	r := b.iniciar("op-1", chaveIdempotencia)
	if r.status != http.StatusCreated || !strings.HasPrefix(r.tipo, "application/json") {
		t.Fatalf("iniciar: status=%d tipo=%s corpo=%v", r.status, r.tipo, r.corpo)
	}
	id, _ := r.corpo["idValidacao"].(string)
	contrassenha, _ := r.corpo["contrassenha"].(string)
	if !dominio.UlidValido(id) || len(contrassenha) != 4 || r.corpo["status"] != "aberta" {
		t.Fatalf("corpo inesperado: %v", r.corpo)
	}
	if _, temNulo := r.corpo["concluidaEm"]; !temNulo || r.corpo["concluidaEm"] != nil {
		t.Fatalf("concluidaEm deve estar presente e nulo: %v", r.corpo)
	}
	maquina := r.corpo["maquina"].(map[string]any)
	if maquina["id"] != "VM-2047" || maquina["localizacao"].(map[string]any)["nome"] != "Bloco A - Térreo" {
		t.Fatalf("máquina inesperada: %v", maquina)
	}

	repetida := b.iniciar("op-1", chaveIdempotencia)
	if repetida.status != http.StatusCreated || repetida.corpo["idValidacao"] != id {
		t.Fatalf("reenvio idempotente: %d %v", repetida.status, repetida.corpo)
	}

	b.rel.Avancar(5 * time.Minute)
	c := b.chamar(http.MethodPost, "/v1/validacoes/"+id+"/conclusao",
		map[string]any{"leituras": map[string]int{"medidor": 14832, "unidadesVendidas": 137}, "codigoRotacionado": true},
		map[string]string{apihttp.CabecalhoUserinfo: userinfo("op-1")})
	if c.status != http.StatusOK || c.corpo["status"] != "concluida" || c.corpo["codigoRotacionado"] != true {
		t.Fatalf("conclusão: %d %v", c.status, c.corpo)
	}
	if avisos, ok := c.corpo["avisos"].([]any); !ok || len(avisos) != 0 {
		t.Fatalf("avisos deve ser uma lista vazia: %v", c.corpo["avisos"])
	}

	q := b.chamar(http.MethodGet, "/v1/validacoes/"+id, nil, map[string]string{apihttp.CabecalhoUserinfo: userinfo("op-1")})
	if q.status != http.StatusOK || q.corpo["status"] != "concluida" {
		t.Fatalf("consulta: %d %v", q.status, q.corpo)
	}
	if _, expoe := q.corpo["contrassenha"]; expoe {
		t.Fatal("consulta de validação concluída não deve expor a contrassenha")
	}
	if len(b.pub.Publicados()) != 2 {
		t.Fatalf("eventos publicados: %d", len(b.pub.Publicados()))
	}
}

func TestErrosPeloContrato(t *testing.T) {
	b := novoBanco(t)

	esperaProblema(t, b.chamar(http.MethodPost, "/v1/validacoes", map[string]any{"qr": b.payload("2027-12-31")},
		map[string]string{"Idempotency-Key": chaveIdempotencia}), http.StatusUnauthorized, "operador_nao_autorizado")
	esperaProblema(t, b.iniciar("op-1", ""), http.StatusBadRequest, "payload_invalido")
	esperaProblema(t, b.iniciar("op-1", "nao-e-uuid"), http.StatusBadRequest, "payload_invalido")
	esperaProblema(t, b.iniciar("desconhecido", chaveIdempotencia), http.StatusForbidden, "operador_nao_autorizado")

	vencido := b.chamar(http.MethodPost, "/v1/validacoes", map[string]any{"qr": b.payload("2025-12-31")},
		map[string]string{apihttp.CabecalhoUserinfo: userinfo("op-1"), "Idempotency-Key": "11111111-2222-4333-8444-555555555555"})
	esperaProblema(t, vencido, http.StatusGone, "qr_expirado")

	campoExtra := b.chamar(http.MethodPost, "/v1/validacoes", map[string]any{"qr": b.payload("2027-12-31"), "extra": 1},
		map[string]string{apihttp.CabecalhoUserinfo: userinfo("op-1"), "Idempotency-Key": "11111111-2222-4333-8444-555555555556"})
	esperaProblema(t, campoExtra, http.StatusBadRequest, "payload_invalido")

	aberta := b.iniciar("op-1", chaveIdempotencia)
	conflito := b.iniciar("op-2", "22222222-2222-4333-8444-555555555555")
	esperaProblema(t, conflito, http.StatusConflict, "validacao_em_andamento")
	if conflito.corpo["idValidacao"] != aberta.corpo["idValidacao"] {
		t.Fatalf("conflito deve apontar a validação aberta: %v", conflito.corpo)
	}

	esperaProblema(t, b.chamar(http.MethodGet, "/v1/validacoes/01J8ZK3V9Q7XW2N4M6P8R0T2Y4", nil,
		map[string]string{apihttp.CabecalhoUserinfo: userinfo("op-1")}), http.StatusNotFound, "validacao_nao_encontrada")

	semCampos := b.chamar(http.MethodPost, "/v1/validacoes/"+aberta.corpo["idValidacao"].(string)+"/conclusao",
		map[string]any{"leituras": map[string]int{"medidor": 1, "unidadesVendidas": 1}},
		map[string]string{apihttp.CabecalhoUserinfo: userinfo("op-1")})
	esperaProblema(t, semCampos, http.StatusBadRequest, "payload_invalido")
}

func TestIdentidade(t *testing.T) {
	b := novoBanco(t)

	comTeste := b.chamar(http.MethodPost, "/v1/validacoes", map[string]any{"qr": b.payload("2027-12-31")},
		map[string]string{"X-Operador-Teste": "op-1", "Idempotency-Key": chaveIdempotencia})
	if comTeste.status != http.StatusCreated {
		t.Fatalf("cabeçalho de teste deveria identificar o operador: %d %v", comTeste.status, comTeste.corpo)
	}

	claims, _ := json.Marshal(map[string]any{"user_id": "op-2"})
	comPadding := base64.URLEncoding.EncodeToString(claims)
	r := b.chamar(http.MethodGet, "/v1/validacoes/"+comTeste.corpo["idValidacao"].(string), nil,
		map[string]string{apihttp.CabecalhoUserinfo: comPadding})
	esperaProblema(t, r, http.StatusForbidden, "operador_nao_autorizado")

	ilegivel := b.chamar(http.MethodGet, "/saude", nil, map[string]string{apihttp.CabecalhoUserinfo: "%%%"})
	if ilegivel.status != http.StatusOK {
		t.Fatalf("/saude não exige identidade: %d", ilegivel.status)
	}
}

func TestReconciliacaoPeloEndpointInterno(t *testing.T) {
	b := novoBanco(t)
	b.pub.Falha = io.ErrClosedPipe
	r := b.iniciar("op-1", chaveIdempotencia)
	if r.status != http.StatusCreated {
		t.Fatalf("iniciar com publicador fora: %d %v", r.status, r.corpo)
	}
	b.pub.Falha = nil
	b.rel.Avancar(3 * time.Minute)

	rec := b.chamar(http.MethodPost, "/interno/reconciliar-publicacoes", nil, nil)
	if rec.status != http.StatusOK || rec.corpo["publicadas"] != float64(1) || rec.corpo["examinadas"] != float64(1) {
		t.Fatalf("reconciliação: %d %v", rec.status, rec.corpo)
	}
	if len(b.pub.Publicados()) != 1 {
		t.Fatalf("evento deveria ter sido republicado: %d", len(b.pub.Publicados()))
	}
	de_novo := b.chamar(http.MethodPost, "/interno/reconciliar-publicacoes", nil, nil)
	if de_novo.corpo["examinadas"] != float64(0) {
		t.Fatalf("segunda reconciliação não deveria achar pendências: %v", de_novo.corpo)
	}
}
