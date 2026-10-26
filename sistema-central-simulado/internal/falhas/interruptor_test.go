package falhas_test

import (
	"testing"

	"github.com/ThiagoCristovao/serverless-vending-validation/sistema-central-simulado/internal/falhas"
)

func TestInterruptor(t *testing.T) {
	i := falhas.Novo()
	if i.Rejeita("VM-1") || i.PerdeConfirmacao("VM-1") {
		t.Fatal("desligado não deveria falhar")
	}
	if err := i.Definir(falhas.Configuracao{Modo: "explodir"}); err == nil {
		t.Fatal("modo inválido aceito")
	}
	if err := i.Definir(falhas.Configuracao{Modo: falhas.ModoRejeitar, IdMaquina: "VM-2"}); err != nil {
		t.Fatal(err)
	}
	if i.Rejeita("VM-1") || !i.Rejeita("VM-2") || i.PerdeConfirmacao("VM-2") {
		t.Fatalf("filtro por máquina: %+v", i.Atual())
	}
	_ = i.Definir(falhas.Configuracao{Modo: falhas.ModoConfirmacaoPerdida})
	if !i.PerdeConfirmacao("VM-1") || !i.PerdeConfirmacao("VM-2") || i.Rejeita("VM-1") {
		t.Fatalf("sem máquina aplica a todas: %+v", i.Atual())
	}
	_ = i.Definir(falhas.Configuracao{Modo: falhas.ModoNenhum, IdMaquina: "VM-1"})
	if c := i.Atual(); c.Modo != falhas.ModoNenhum || c.IdMaquina != "" {
		t.Fatalf("nenhum limpa a máquina: %+v", c)
	}
}
