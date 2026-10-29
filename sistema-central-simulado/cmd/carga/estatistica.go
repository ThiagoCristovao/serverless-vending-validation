package main

import (
	"fmt"
	"slices"
	"sync"
	"time"
)

// amostra é uma requisição feita por um operador virtual.
type amostra struct {
	Quando   time.Time
	Operador int
	Tipo     string // abertura | conclusao
	Status   int
	Duracao  time.Duration
	Erro     bool
}

// coletor acumula amostras de forma segura entre goroutines.
type coletor struct {
	mu       sync.Mutex
	amostras []amostra
}

func (c *coletor) registrar(a amostra) {
	c.mu.Lock()
	c.amostras = append(c.amostras, a)
	c.mu.Unlock()
}

func (c *coletor) todas() []amostra {
	c.mu.Lock()
	defer c.mu.Unlock()
	saida := slices.Clone(c.amostras)
	slices.SortFunc(saida, func(a, b amostra) int { return a.Quando.Compare(b.Quando) })
	return saida
}

// resumo estatístico de um conjunto de durações.
type resumo struct {
	N, Erros             int
	Media, P50, P95, P99 time.Duration
	Max                  time.Duration
}

func resumir(as []amostra) resumo {
	r := resumo{N: len(as)}
	if len(as) == 0 {
		return r
	}
	ds := make([]time.Duration, 0, len(as))
	var soma time.Duration
	for _, a := range as {
		if a.Erro {
			r.Erros++
		}
		ds = append(ds, a.Duracao)
		soma += a.Duracao
	}
	slices.Sort(ds)
	r.Media = soma / time.Duration(len(ds))
	r.P50, r.P95, r.P99 = percentil(ds, 50), percentil(ds, 95), percentil(ds, 99)
	r.Max = ds[len(ds)-1]
	return r
}

// percentil pelo método do posto mais próximo (nearest rank).
func percentil(ordenadas []time.Duration, p int) time.Duration {
	if len(ordenadas) == 0 {
		return 0
	}
	posto := (len(ordenadas)*p + 99) / 100
	if posto < 1 {
		posto = 1
	}
	return ordenadas[posto-1]
}

func (r resumo) linha(rotulo string) []string {
	return []string{rotulo, fmt.Sprint(r.N), fmt.Sprint(r.Erros), ms(r.Media), ms(r.P50), ms(r.P95), ms(r.P99), ms(r.Max)}
}

func ms(d time.Duration) string { return fmt.Sprintf("%d ms", d.Milliseconds()) }

func msf(v float64, ok bool) string {
	if !ok {
		return "—"
	}
	return fmt.Sprintf("%.0f ms", v)
}

func filtrar(as []amostra, tipo string) []amostra {
	var saida []amostra
	for _, a := range as {
		if a.Tipo == tipo {
			saida = append(saida, a)
		}
	}
	return saida
}

// porIntervalo agrupa as amostras em janelas de `passo` a partir de `inicio`.
func porIntervalo(as []amostra, inicio time.Time, passo time.Duration) [][]amostra {
	var grupos [][]amostra
	for _, a := range as {
		i := int(a.Quando.Sub(inicio) / passo)
		if i < 0 {
			i = 0
		}
		for len(grupos) <= i {
			grupos = append(grupos, nil)
		}
		grupos[i] = append(grupos[i], a)
	}
	return grupos
}
