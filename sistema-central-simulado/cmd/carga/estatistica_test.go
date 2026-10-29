package main

import (
	"testing"
	"time"
)

func TestPercentilPostoMaisProximo(t *testing.T) {
	var ds []time.Duration
	for i := 1; i <= 100; i++ {
		ds = append(ds, time.Duration(i)*time.Millisecond)
	}
	casos := map[int]time.Duration{50: 50 * time.Millisecond, 95: 95 * time.Millisecond, 99: 99 * time.Millisecond, 100: 100 * time.Millisecond}
	for p, esperado := range casos {
		if v := percentil(ds, p); v != esperado {
			t.Errorf("p%d = %s, esperado %s", p, v, esperado)
		}
	}
	if percentil(nil, 95) != 0 {
		t.Error("percentil de lista vazia deve ser 0")
	}
	if v := percentil([]time.Duration{7 * time.Millisecond}, 99); v != 7*time.Millisecond {
		t.Errorf("p99 de um elemento = %s", v)
	}
}

func TestResumirContaErrosEMaximo(t *testing.T) {
	as := []amostra{
		{Duracao: 100 * time.Millisecond}, {Duracao: 300 * time.Millisecond, Erro: true}, {Duracao: 200 * time.Millisecond},
	}
	r := resumir(as)
	if r.N != 3 || r.Erros != 1 || r.Max != 300*time.Millisecond || r.Media != 200*time.Millisecond || r.P50 != 200*time.Millisecond {
		t.Errorf("resumo inesperado: %+v", r)
	}
	if v := resumir(nil); v.N != 0 || v.P95 != 0 {
		t.Errorf("resumo vazio inesperado: %+v", v)
	}
}

func TestPorIntervaloAgrupaDesdeOInicio(t *testing.T) {
	inicio := time.Date(2026, 10, 29, 10, 0, 0, 0, time.UTC)
	as := []amostra{
		{Quando: inicio.Add(1 * time.Second)}, {Quando: inicio.Add(9 * time.Second)},
		{Quando: inicio.Add(25 * time.Second)}, {Quando: inicio.Add(-time.Second)},
	}
	grupos := porIntervalo(as, inicio, 10*time.Second)
	if len(grupos) != 3 || len(grupos[0]) != 3 || len(grupos[1]) != 0 || len(grupos[2]) != 1 {
		t.Errorf("grupos inesperados: %d (%d, %d, %d)", len(grupos), len(grupos[0]), len(grupos[1]), len(grupos[2]))
	}
}
