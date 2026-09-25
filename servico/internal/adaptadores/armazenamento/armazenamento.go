// Package armazenamento implementa os repositórios sobre o Firestore
// (docs/modelo-dados.md). As operações que envolvem máquina e validação ao
// mesmo tempo rodam em transação, o que garante uma única validação aberta por
// máquina (RN-03) e a rotação atômica do contador (RN-02).
package armazenamento

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/ThiagoCristovao/serverless-vending-validation/servico/internal/dominio"
	"github.com/ThiagoCristovao/serverless-vending-validation/servico/internal/portas"
)

const (
	colecaoMaquinas   = "maquinas"
	colecaoOperadores = "operadores"
	colecaoChavesQr   = "chavesQr"
	colecaoValidacoes = "validacoes"
)

// Repositorio implementa as portas de persistência sobre um cliente Firestore.
type Repositorio struct {
	cli *firestore.Client
}

// Novo cria o repositório.
func Novo(cli *firestore.Client) *Repositorio { return &Repositorio{cli: cli} }

func (r *Repositorio) maquinas() *firestore.CollectionRef { return r.cli.Collection(colecaoMaquinas) }
func (r *Repositorio) operadores() *firestore.CollectionRef {
	return r.cli.Collection(colecaoOperadores)
}
func (r *Repositorio) chaves() *firestore.CollectionRef { return r.cli.Collection(colecaoChavesQr) }
func (r *Repositorio) validacoes() *firestore.CollectionRef {
	return r.cli.Collection(colecaoValidacoes)
}

func naoEncontrado(err error) bool { return status.Code(err) == codes.NotFound }

// ObterMaquina implementa portas.RepositorioMaquinas.
func (r *Repositorio) ObterMaquina(ctx context.Context, id string) (*dominio.Maquina, error) {
	snap, err := r.maquinas().Doc(id).Get(ctx)
	if err != nil {
		if naoEncontrado(err) {
			return nil, portas.ErrNaoEncontrado
		}
		return nil, fmt.Errorf("firestore: obter máquina %s: %w", id, err)
	}
	return maquinaDe(snap)
}

// ObterOperador implementa portas.RepositorioOperadores.
func (r *Repositorio) ObterOperador(ctx context.Context, uid string) (*dominio.Operador, error) {
	snap, err := r.operadores().Doc(uid).Get(ctx)
	if err != nil {
		if naoEncontrado(err) {
			return nil, portas.ErrNaoEncontrado
		}
		return nil, fmt.Errorf("firestore: obter operador %s: %w", uid, err)
	}
	var d docOperador
	if err := snap.DataTo(&d); err != nil {
		return nil, fmt.Errorf("firestore: decodificar operador %s: %w", uid, err)
	}
	return &dominio.Operador{Uid: snap.Ref.ID, Nome: d.Nome, Email: d.Email, Rota: valor(d.Rota), Ativo: d.Ativo, CriadoEm: d.CriadoEm}, nil
}

// ObterChaveQr implementa portas.ProvedorChavesQr.
func (r *Repositorio) ObterChaveQr(ctx context.Context, kid string) (*dominio.ChaveQr, error) {
	snap, err := r.chaves().Doc(kid).Get(ctx)
	if err != nil {
		if naoEncontrado(err) {
			return nil, portas.ErrNaoEncontrado
		}
		return nil, fmt.Errorf("firestore: obter chave %s: %w", kid, err)
	}
	var d docChaveQr
	if err := snap.DataTo(&d); err != nil {
		return nil, fmt.Errorf("firestore: decodificar chave %s: %w", kid, err)
	}
	publica, err := base64.RawURLEncoding.DecodeString(d.ChavePublica)
	if err != nil {
		return nil, fmt.Errorf("firestore: chave pública %s inválida: %w", kid, err)
	}
	return &dominio.ChaveQr{Kid: snap.Ref.ID, ChavePublica: publica, Ativa: d.Ativa, ValidaDe: d.ValidaDe, ValidaAte: d.ValidaAte}, nil
}

// ObterValidacao implementa portas.RepositorioValidacoes.
func (r *Repositorio) ObterValidacao(ctx context.Context, id string) (*dominio.Validacao, error) {
	snap, err := r.validacoes().Doc(id).Get(ctx)
	if err != nil {
		if naoEncontrado(err) {
			return nil, portas.ErrNaoEncontrado
		}
		return nil, fmt.Errorf("firestore: obter validação %s: %w", id, err)
	}
	return validacaoDe(snap)
}

// ObterPorChaveIdempotencia implementa portas.RepositorioValidacoes.
func (r *Repositorio) ObterPorChaveIdempotencia(ctx context.Context, chave string) (*dominio.Validacao, error) {
	it := r.validacoes().Where("chaveIdempotencia", "==", chave).Limit(1).Documents(ctx)
	defer it.Stop()
	snap, err := it.Next()
	if errors.Is(err, iterator.Done) {
		return nil, portas.ErrNaoEncontrado
	}
	if err != nil {
		return nil, fmt.Errorf("firestore: consultar chave de idempotência: %w", err)
	}
	return validacaoDe(snap)
}

// Abrir implementa portas.RepositorioValidacoes em uma transação.
func (r *Repositorio) Abrir(ctx context.Context, v *dominio.Validacao, agora time.Time) error {
	refMaquina := r.maquinas().Doc(v.IdMaquina)
	refNova := r.validacoes().Doc(v.Id)
	return r.cli.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		snapMaquina, err := tx.Get(refMaquina)
		if err != nil {
			if naoEncontrado(err) {
				return portas.ErrNaoEncontrado
			}
			return err
		}
		m, err := maquinaDe(snapMaquina)
		if err != nil {
			return err
		}
		var expirar *firestore.DocumentRef
		if m.ValidacaoAbertaId != "" {
			refAberta := r.validacoes().Doc(m.ValidacaoAbertaId)
			snapAberta, err := tx.Get(refAberta)
			if err != nil && !naoEncontrado(err) {
				return err
			}
			if err == nil {
				aberta, err := validacaoDe(snapAberta)
				if err != nil {
					return err
				}
				if aberta.Aberta() {
					if !aberta.Expirou(agora) {
						return &portas.ErrValidacaoEmAndamento{IdValidacao: aberta.Id, IdOperador: aberta.IdOperador}
					}
					expirar = refAberta
				}
			}
		}
		if expirar != nil {
			if err := tx.Update(expirar, []firestore.Update{{Path: "status", Value: string(dominio.StatusExpirada)}}); err != nil {
				return err
			}
		}
		if err := tx.Set(refNova, docDeValidacao(v)); err != nil {
			return err
		}
		return tx.Update(refMaquina, []firestore.Update{
			{Path: "validacaoAbertaId", Value: v.Id},
			{Path: "atualizadaEm", Value: agora},
		})
	})
}

// Concluir implementa portas.RepositorioValidacoes em uma transação.
func (r *Repositorio) Concluir(ctx context.Context, v *dominio.Validacao, agora time.Time) (*dominio.Maquina, error) {
	refMaquina := r.maquinas().Doc(v.IdMaquina)
	refValidacao := r.validacoes().Doc(v.Id)
	var atualizada *dominio.Maquina
	err := r.cli.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		snap, err := tx.Get(refMaquina)
		if err != nil {
			if naoEncontrado(err) {
				return portas.ErrNaoEncontrado
			}
			return err
		}
		m, err := maquinaDe(snap)
		if err != nil {
			return err
		}
		m.RegistrarConclusao(v, agora)
		if err := tx.Set(refValidacao, docDeValidacao(v)); err != nil {
			return err
		}
		if err := tx.Set(refMaquina, docDeMaquina(m)); err != nil {
			return err
		}
		atualizada = m
		return nil
	})
	if err != nil {
		return nil, err
	}
	return atualizada, nil
}

// MarcarExpirada implementa portas.RepositorioValidacoes.
func (r *Repositorio) MarcarExpirada(ctx context.Context, id string) error {
	refValidacao := r.validacoes().Doc(id)
	return r.cli.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		snap, err := tx.Get(refValidacao)
		if err != nil {
			if naoEncontrado(err) {
				return portas.ErrNaoEncontrado
			}
			return err
		}
		v, err := validacaoDe(snap)
		if err != nil {
			return err
		}
		refMaquina := r.maquinas().Doc(v.IdMaquina)
		snapMaquina, err := tx.Get(refMaquina)
		liberar := false
		if err == nil {
			m, err := maquinaDe(snapMaquina)
			if err != nil {
				return err
			}
			liberar = m.ValidacaoAbertaId == id
		} else if !naoEncontrado(err) {
			return err
		}
		if err := tx.Update(refValidacao, []firestore.Update{{Path: "status", Value: string(dominio.StatusExpirada)}}); err != nil {
			return err
		}
		if liberar {
			return tx.Update(refMaquina, []firestore.Update{{Path: "validacaoAbertaId", Value: nil}})
		}
		return nil
	})
}

// MarcarEventoPublicado implementa portas.RepositorioValidacoes.
func (r *Repositorio) MarcarEventoPublicado(ctx context.Context, id string, tipo dominio.TipoEvento, quando time.Time) error {
	campo := ""
	switch tipo {
	case dominio.EventoValidacaoIniciada:
		campo = "eventos.iniciadaPublicadaEm"
	case dominio.EventoValidacaoConcluida:
		campo = "eventos.concluidaPublicadaEm"
	default:
		return fmt.Errorf("tipo de evento desconhecido: %s", tipo)
	}
	_, err := r.validacoes().Doc(id).Update(ctx, []firestore.Update{{Path: campo, Value: quando}})
	if err != nil {
		if naoEncontrado(err) {
			return portas.ErrNaoEncontrado
		}
		return fmt.Errorf("firestore: marcar evento %s de %s: %w", tipo, id, err)
	}
	return nil
}

// PublicacoesPendentes implementa portas.RepositorioValidacoes com duas
// consultas de igualdade a nulo (índices de campo único, automáticos); o filtro
// de tempo é aplicado em memória para não exigir índice composto adicional.
func (r *Repositorio) PublicacoesPendentes(ctx context.Context, antesDe time.Time, limite int) ([]*dominio.Validacao, error) {
	if limite <= 0 {
		limite = 50
	}
	vistas := map[string]bool{}
	var pendentes []*dominio.Validacao
	consultas := []firestore.Query{
		r.validacoes().Where("eventos.iniciadaPublicadaEm", "==", nil).Limit(limite),
		r.validacoes().Where("status", "==", string(dominio.StatusConcluida)).Where("eventos.concluidaPublicadaEm", "==", nil).Limit(limite),
	}
	for _, q := range consultas {
		it := q.Documents(ctx)
		for {
			snap, err := it.Next()
			if errors.Is(err, iterator.Done) {
				break
			}
			if err != nil {
				it.Stop()
				return nil, fmt.Errorf("firestore: consultar publicações pendentes: %w", err)
			}
			v, err := validacaoDe(snap)
			if err != nil {
				it.Stop()
				return nil, err
			}
			referencia := v.IniciadaEm
			if v.Status == dominio.StatusConcluida && v.ConcluidaEm != nil && v.Eventos.IniciadaEm != nil {
				referencia = *v.ConcluidaEm
			}
			if !referencia.Before(antesDe) || vistas[v.Id] {
				continue
			}
			vistas[v.Id] = true
			pendentes = append(pendentes, v)
			if len(pendentes) >= limite {
				it.Stop()
				return pendentes, nil
			}
		}
		it.Stop()
	}
	return pendentes, nil
}

// Semear grava máquinas, operadores e chaves de exemplo; usado pela ferramenta
// semear-firestore e pelos testes de integração.
func (r *Repositorio) Semear(ctx context.Context, maquinas []dominio.Maquina, operadores []dominio.Operador, chaves []dominio.ChaveQr) error {
	lote := r.cli.BulkWriter(ctx)
	for i := range maquinas {
		if _, err := lote.Set(r.maquinas().Doc(maquinas[i].Id), docDeMaquina(&maquinas[i])); err != nil {
			return err
		}
	}
	for i := range operadores {
		o := operadores[i]
		if _, err := lote.Set(r.operadores().Doc(o.Uid), docOperador{Nome: o.Nome, Email: o.Email, Rota: ponteiroSeNaoVazio(o.Rota), Ativo: o.Ativo, CriadoEm: o.CriadoEm}); err != nil {
			return err
		}
	}
	for i := range chaves {
		c := chaves[i]
		if _, err := lote.Set(r.chaves().Doc(c.Kid), docChaveQr{Algoritmo: "Ed25519", ChavePublica: base64.RawURLEncoding.EncodeToString(c.ChavePublica), Ativa: c.Ativa, ValidaDe: c.ValidaDe, ValidaAte: c.ValidaAte}); err != nil {
			return err
		}
	}
	lote.End()
	return nil
}
