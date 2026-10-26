package armazenamento_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ThiagoCristovao/serverless-vending-validation/sistema-central-simulado/internal/armazenamento"
	"github.com/ThiagoCristovao/serverless-vending-validation/sistema-central-simulado/internal/eventos"
)

func item(id, tipo, maq string) armazenamento.Item {
	agora := time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)
	return armazenamento.Item{
		IdValidacao: id, Tipo: tipo, IdMaquina: maq, IdMensagem: "m-" + id + tipo,
		PublicadaEm: agora, RecebidaEm: agora, Tentativa: 1,
		Evento: eventos.Evento{Tipo: tipo, Versao: 1, IdValidacao: id, Maquina: eventos.Maquina{Id: maq}},
		Corpo:  json.RawMessage(`{"tipo":"` + tipo + `"}`),
	}
}

func TestRegistrarDetectaDuplicatasEPersiste(t *testing.T) {
	caminho := filepath.Join(t.TempDir(), "dados", "eventos.jsonl")
	r, err := armazenamento.Abrir(caminho)
	if err != nil {
		t.Fatal(err)
	}
	const id = "01J8ZK3V9Q7XW2N4M6P8R0T2Y4"
	if novo, err := r.Registrar(item(id, eventos.TipoValidacaoIniciada, "VM-1")); err != nil || !novo {
		t.Fatalf("primeira: novo=%v err=%v", novo, err)
	}
	if novo, err := r.Registrar(item(id, eventos.TipoValidacaoIniciada, "VM-1")); err != nil || novo {
		t.Fatalf("duplicata: novo=%v err=%v", novo, err)
	}
	if novo, err := r.Registrar(item(id, eventos.TipoValidacaoConcluida, "VM-1")); err != nil || !novo {
		t.Fatalf("outro tipo é novo: novo=%v err=%v", novo, err)
	}
	if r.Total() != 2 {
		t.Fatalf("total %d", r.Total())
	}
	if err := r.Fechar(); err != nil {
		t.Fatal(err)
	}

	// Reabrir reconstrói o índice: a duplicata continua sendo reconhecida.
	r2, err := armazenamento.Abrir(caminho)
	if err != nil {
		t.Fatal(err)
	}
	defer r2.Fechar()
	if r2.Total() != 2 {
		t.Fatalf("após reabrir: total %d", r2.Total())
	}
	if novo, _ := r2.Registrar(item(id, eventos.TipoValidacaoConcluida, "VM-1")); novo {
		t.Fatal("duplicata aceita após reinício")
	}
	if it, ok := r2.Obter(eventos.Chave{IdValidacao: id, Tipo: eventos.TipoValidacaoIniciada}); !ok || it.IdMaquina != "VM-1" || string(it.Corpo) != `{"tipo":"validacao.iniciada"}` {
		t.Fatalf("obter: %v %+v", ok, it)
	}
	conteudo, _ := os.ReadFile(caminho)
	if linhas := len(splitLinhas(conteudo)); linhas != 2 {
		t.Fatalf("arquivo com %d linhas", linhas)
	}
}

func TestListarComFiltro(t *testing.T) {
	r, _ := armazenamento.Abrir("")
	_, _ = r.Registrar(item("01J8ZK3V9Q7XW2N4M6P8R0T2Y4", eventos.TipoValidacaoIniciada, "VM-1"))
	_, _ = r.Registrar(item("01J8ZK3V9Q7XW2N4M6P8R0T2Y4", eventos.TipoValidacaoConcluida, "VM-1"))
	_, _ = r.Registrar(item("01J8ZK3V9Q7XW2N4M6P8R0T2Y5", eventos.TipoValidacaoIniciada, "VM-2"))
	if n := len(r.Listar(armazenamento.Filtro{})); n != 3 {
		t.Fatalf("sem filtro: %d", n)
	}
	if n := len(r.Listar(armazenamento.Filtro{IdMaquina: "VM-1"})); n != 2 {
		t.Fatalf("por máquina: %d", n)
	}
	if n := len(r.Listar(armazenamento.Filtro{IdValidacao: "01J8ZK3V9Q7XW2N4M6P8R0T2Y4", Tipo: eventos.TipoValidacaoConcluida})); n != 1 {
		t.Fatalf("por validação e tipo: %d", n)
	}
	if n := len(r.Listar(armazenamento.Filtro{IdMaquina: "VM-9"})); n != 0 {
		t.Fatalf("inexistente: %d", n)
	}
}

func TestAbrirRejeitaArquivoCorrompido(t *testing.T) {
	caminho := filepath.Join(t.TempDir(), "eventos.jsonl")
	_ = os.WriteFile(caminho, []byte("{\"idValidacao\":\"x\"}\nisto não é json\n"), 0o644)
	if _, err := armazenamento.Abrir(caminho); err == nil {
		t.Fatal("esperava erro de linha corrompida")
	}
}

func TestRegistrarExigeChave(t *testing.T) {
	r, _ := armazenamento.Abrir("")
	if _, err := r.Registrar(armazenamento.Item{}); err == nil {
		t.Fatal("esperava erro")
	}
}

func splitLinhas(b []byte) [][]byte {
	var linhas [][]byte
	inicio := 0
	for i, c := range b {
		if c == '\n' {
			linhas = append(linhas, b[inicio:i])
			inicio = i + 1
		}
	}
	return linhas
}
