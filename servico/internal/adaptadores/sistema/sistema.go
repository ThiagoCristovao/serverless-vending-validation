// Package sistema implementa as portas que dependem só do sistema operacional:
// relógio real e gerador de identificadores com entropia criptográfica.
package sistema

import (
	"crypto/rand"
	"time"

	"github.com/ThiagoCristovao/serverless-vending-validation/servico/internal/dominio"
	"github.com/ThiagoCristovao/serverless-vending-validation/servico/internal/portas"
)

// Relogio devolve o instante atual em UTC.
type Relogio struct{}

// Agora implementa portas.Relogio.
func (Relogio) Agora() time.Time { return time.Now().UTC() }

// GeradorUlid gera ULIDs com o instante do relógio e entropia criptográfica.
type GeradorUlid struct {
	Relogio portas.Relogio
}

// NovoId implementa portas.GeradorIdentificador.
func (g GeradorUlid) NovoId() (string, error) {
	rel := g.Relogio
	if rel == nil {
		rel = Relogio{}
	}
	return dominio.NovoUlid(rel.Agora(), rand.Reader)
}
