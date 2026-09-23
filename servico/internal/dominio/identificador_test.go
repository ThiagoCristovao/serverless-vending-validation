package dominio

import (
	"crypto/rand"
	"errors"
	"strings"
	"testing"
	"testing/iotest"
	"time"
)

func TestNovoUlid(t *testing.T) {
	t1 := time.Date(2026, 9, 17, 13, 5, 12, 0, time.UTC)

	a, err := NovoUlid(t1, rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := NovoUlid(t1, rand.Reader)
	c, _ := NovoUlid(t1.Add(time.Second), rand.Reader)

	if !UlidValido(a) || !UlidValido(b) || !UlidValido(c) {
		t.Fatalf("identificadores fora do formato: %s %s %s", a, b, c)
	}
	if a == b {
		t.Fatal("dois identificadores no mesmo instante devem diferir pela entropia")
	}
	if a[:10] != b[:10] {
		t.Fatal("o prefixo de tempo deve ser igual para o mesmo instante")
	}
	if !(a < c) {
		t.Fatalf("ordenação por tempo falhou: %s deveria vir antes de %s", a, c)
	}
	if _, err := NovoUlid(t1, iotest.ErrReader(errors.New("sem entropia"))); err == nil {
		t.Fatal("falha na fonte de entropia deve ser propagada")
	}
}

func TestCodificarUlid_VetoresConhecidos(t *testing.T) {
	if got := codificarUlid([16]byte{}); got != strings.Repeat("0", 26) {
		t.Fatalf("zeros = %s", got)
	}
	var maximo [16]byte
	for i := range maximo {
		maximo[i] = 0xFF
	}
	if got := codificarUlid(maximo); got != "7ZZZZZZZZZZZZZZZZZZZZZZZZZ" {
		t.Fatalf("máximo = %s", got)
	}
}

func TestUlidValido(t *testing.T) {
	casos := map[string]bool{
		"01J8ZK3V9Q7XW2N4M6P8R0T2Y4": true,
		"abc":                        false,
		"01J8ZK3V9Q7XW2N4M6P8R0T2YI": false, // I não pertence ao alfabeto
		"01J8ZK3V9Q7XW2N4M6P8R0T2Y":  false,
	}
	for s, esperado := range casos {
		if got := UlidValido(s); got != esperado {
			t.Errorf("UlidValido(%q) = %v, esperava %v", s, got, esperado)
		}
	}
}
