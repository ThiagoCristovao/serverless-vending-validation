package admin_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ThiagoCristovao/serverless-vending-validation/sistema-central-simulado/internal/admin"
	"github.com/ThiagoCristovao/serverless-vending-validation/sistema-central-simulado/internal/armazenamento"
	"github.com/ThiagoCristovao/serverless-vending-validation/sistema-central-simulado/internal/consumidor"
	"github.com/ThiagoCristovao/serverless-vending-validation/sistema-central-simulado/internal/eventos"
	"github.com/ThiagoCristovao/serverless-vending-validation/sistema-central-simulado/internal/falhas"
)

type controleFalso struct{ estado consumidor.Estado }

func (c *controleFalso) Iniciar(context.Context) error { c.estado.Consumindo = true; return nil }
func (c *controleFalso) Pausar()                       { c.estado.Consumindo = false }
func (c *controleFalso) Estado() consumidor.Estado     { return c.estado }

func novoServidor(t *testing.T) (*httptest.Server, *armazenamento.Registro, *falhas.Interruptor) {
	t.Helper()
	reg, _ := armazenamento.Abrir("")
	inter := falhas.Novo()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	proc := consumidor.NovoProcessador(reg, inter, log)
	srv := httptest.NewServer(admin.NovoHandler(admin.Dependencias{Registro: reg, Processador: proc, Interruptor: inter, Consumidor: &controleFalso{estado: consumidor.Estado{Assinatura: "s", Consumindo: true}}, Log: log}))
	t.Cleanup(srv.Close)
	return srv, reg, inter
}

func chamar(t *testing.T, metodo, url, corpo string) (int, map[string]any) {
	t.Helper()
	req, _ := http.NewRequest(metodo, url, strings.NewReader(corpo))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var dados map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&dados); err != nil {
		t.Fatal(err)
	}
	return resp.StatusCode, dados
}

func TestEventosEEstatisticas(t *testing.T) {
	srv, reg, _ := novoServidor(t)
	status, dados := chamar(t, "GET", srv.URL+"/eventos", "")
	if status != 200 || dados["total"].(float64) != 0 || dados["itens"] == nil {
		t.Fatalf("vazio: %d %v", status, dados)
	}
	_, _ = reg.Registrar(armazenamento.Item{IdValidacao: "01J8ZK3V9Q7XW2N4M6P8R0T2Y4", Tipo: eventos.TipoValidacaoIniciada, IdMaquina: "VM-1", Corpo: []byte(`{}`)})
	_, _ = reg.Registrar(armazenamento.Item{IdValidacao: "01J8ZK3V9Q7XW2N4M6P8R0T2Y5", Tipo: eventos.TipoValidacaoIniciada, IdMaquina: "VM-2", Corpo: []byte(`{}`)})
	_, dados = chamar(t, "GET", srv.URL+"/eventos?idMaquina=VM-2", "")
	if dados["total"].(float64) != 1 {
		t.Fatalf("filtro: %v", dados)
	}
	status, dados = chamar(t, "GET", srv.URL+"/estatisticas", "")
	if status != 200 || dados["recebidas"].(float64) != 0 {
		t.Fatalf("estatísticas: %d %v", status, dados)
	}
	status, dados = chamar(t, "GET", srv.URL+"/saude", "")
	if status != 200 || dados["status"] != "ok" || dados["eventos"].(float64) != 2 {
		t.Fatalf("saúde: %d %v", status, dados)
	}
}

func TestInterruptorEControle(t *testing.T) {
	srv, _, inter := novoServidor(t)
	status, dados := chamar(t, "PUT", srv.URL+"/admin/falhas", `{"modo":"rejeitar","idMaquina":"VM-2"}`)
	if status != 200 || dados["modo"] != "rejeitar" || !inter.Rejeita("VM-2") {
		t.Fatalf("definir: %d %v", status, dados)
	}
	if status, _ := chamar(t, "PUT", srv.URL+"/admin/falhas", `{"modo":"explodir"}`); status != 400 {
		t.Fatalf("modo inválido: %d", status)
	}
	if status, _ := chamar(t, "PUT", srv.URL+"/admin/falhas", `nao-e-json`); status != 400 {
		t.Fatalf("corpo inválido: %d", status)
	}
	_, dados = chamar(t, "GET", srv.URL+"/admin/falhas", "")
	if dados["idMaquina"] != "VM-2" {
		t.Fatalf("atual: %v", dados)
	}
	_, dados = chamar(t, "POST", srv.URL+"/admin/pausar", "")
	if dados["consumindo"] != false {
		t.Fatalf("pausar: %v", dados)
	}
	_, dados = chamar(t, "POST", srv.URL+"/admin/retomar", "")
	if dados["consumindo"] != true {
		t.Fatalf("retomar: %v", dados)
	}
	if status, _ := chamar(t, "GET", srv.URL+"/admin/estado", ""); status != 200 {
		t.Fatalf("estado: %d", status)
	}
}
