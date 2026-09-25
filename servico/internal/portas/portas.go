// Package portas define as interfaces pelas quais a aplicação fala com o mundo
// externo: persistência, mensageria, relógio e identificadores. As
// implementações reais ficam em internal/adaptadores (Firestore e Pub/Sub, na
// Sprint 5); a implementação em memória serve aos testes.
package portas

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ThiagoCristovao/serverless-vending-validation/servico/internal/dominio"
)

// ErrNaoEncontrado é devolvido quando o registro pedido não existe.
var ErrNaoEncontrado = errors.New("registro não encontrado")

// ErrValidacaoEmAndamento é devolvido por Abrir quando a máquina já tem uma
// validação aberta dentro do prazo (RN-03).
type ErrValidacaoEmAndamento struct {
	IdValidacao string
	IdOperador  string
}

func (e *ErrValidacaoEmAndamento) Error() string {
	return fmt.Sprintf("validação %s em andamento pelo operador %s", e.IdValidacao, e.IdOperador)
}

// RepositorioMaquinas dá acesso ao cadastro de máquinas.
type RepositorioMaquinas interface {
	ObterMaquina(ctx context.Context, id string) (*dominio.Maquina, error)
}

// RepositorioOperadores dá acesso ao cadastro de operadores.
type RepositorioOperadores interface {
	ObterOperador(ctx context.Context, uid string) (*dominio.Operador, error)
}

// ProvedorChavesQr dá acesso às chaves públicas que assinam adesivos.
type ProvedorChavesQr interface {
	ObterChaveQr(ctx context.Context, kid string) (*dominio.ChaveQr, error)
}

// RepositorioValidacoes persiste validações e mantém os invariantes que
// envolvem a máquina, de forma atômica (transações no Firestore).
type RepositorioValidacoes interface {
	ObterValidacao(ctx context.Context, id string) (*dominio.Validacao, error)
	ObterPorChaveIdempotencia(ctx context.Context, chave string) (*dominio.Validacao, error)

	// Abrir grava a validação e a registra como aberta na máquina. Se a máquina
	// já tem validação aberta dentro do prazo, devolve *ErrValidacaoEmAndamento;
	// se a anterior expirou, marca-a expirada e prossegue.
	Abrir(ctx context.Context, v *dominio.Validacao, agora time.Time) error

	// Concluir grava a validação concluída e aplica na máquina os efeitos de
	// Maquina.RegistrarConclusao. Devolve a máquina atualizada.
	Concluir(ctx context.Context, v *dominio.Validacao, agora time.Time) (*dominio.Maquina, error)

	MarcarExpirada(ctx context.Context, id string) error
	MarcarEventoPublicado(ctx context.Context, id string, tipo dominio.TipoEvento, quando time.Time) error

	// PublicacoesPendentes lista validações com evento por publicar (marcador
	// nulo), ocorridas antes de antesDe, até limite itens (ADR-0012).
	PublicacoesPendentes(ctx context.Context, antesDe time.Time, limite int) ([]*dominio.Validacao, error)
}

// PublicadorEventos envia eventos ao sistema central.
type PublicadorEventos interface {
	Publicar(ctx context.Context, evento dominio.Evento) error
}

// Relogio fornece o instante atual; injetado para tornar prazos testáveis.
type Relogio interface {
	Agora() time.Time
}

// GeradorIdentificador cria identificadores de validação (ULID).
type GeradorIdentificador interface {
	NovoId() (string, error)
}
