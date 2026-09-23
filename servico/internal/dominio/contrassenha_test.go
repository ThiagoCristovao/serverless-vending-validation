package dominio

import (
	"strconv"
	"testing"
)

var chaveDeTeste = []byte("0123456789abcdef0123456789abcdef")

func TestGeradorContrassenha(t *testing.T) {
	g, err := NovoGeradorContrassenha(chaveDeTeste)
	if err != nil {
		t.Fatal(err)
	}

	a := g.Derivar("VM-2047", 12)
	if a != g.Derivar("VM-2047", 12) {
		t.Fatal("a derivação deve ser determinística")
	}
	if len(a) != DigitosContrassenha {
		t.Fatalf("código %q não tem %d dígitos", a, DigitosContrassenha)
	}
	if _, err := strconv.Atoi(a); err != nil {
		t.Fatalf("código %q não é numérico", a)
	}

	distintos := map[string]bool{}
	for c := uint64(0); c < 20; c++ {
		distintos[g.Derivar("VM-2047", c)] = true
	}
	if len(distintos) < 15 {
		t.Fatalf("dispersão baixa: %d códigos distintos em 20 contadores", len(distintos))
	}

	if g.Derivar("VM-0001", 0) == g.Derivar("VM-0002", 0) && g.Derivar("VM-0001", 1) == g.Derivar("VM-0002", 1) {
		t.Fatal("máquinas diferentes não devem compartilhar a sequência de códigos")
	}

	outro, _ := NovoGeradorContrassenha([]byte("fedcba9876543210fedcba9876543210"))
	if outro.Derivar("VM-2047", 0) == g.Derivar("VM-2047", 0) && outro.Derivar("VM-2047", 1) == g.Derivar("VM-2047", 1) {
		t.Fatal("chaves diferentes não devem produzir a mesma sequência")
	}
}

func TestGeradorContrassenha_ZerosAEsquerda(t *testing.T) {
	g, _ := NovoGeradorContrassenha(chaveDeTeste)
	for c := uint64(0); c < 500; c++ {
		codigo := g.Derivar("VM-2047", c)
		if codigo[0] == '0' {
			if len(codigo) != 4 {
				t.Fatalf("código %q perdeu o zero à esquerda", codigo)
			}
			return
		}
	}
	t.Skip("nenhum código iniciado por zero em 500 contadores; improvável, mas não é falha")
}

func TestNovoGeradorContrassenha_ChaveCurta(t *testing.T) {
	if _, err := NovoGeradorContrassenha([]byte("curta")); err == nil {
		t.Fatal("chave curta deve ser rejeitada")
	}
}
