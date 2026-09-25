package aplicacao

import (
	"context"

	"github.com/ThiagoCristovao/serverless-vending-validation/servico/internal/dominio"
)

// EntradaConcluirValidacao é o pedido de POST /v1/validacoes/{id}/conclusao.
type EntradaConcluirValidacao struct {
	Uid               string
	IdValidacao       string
	Leituras          dominio.Leituras
	CodigoRotacionado bool
	Observacoes       string
}

// ResultadoConclusao é a resposta da conclusão.
type ResultadoConclusao struct {
	Validacao *dominio.Validacao
	Maquina   *dominio.Maquina
	// Reproduzida indica reenvio de uma conclusão já registrada e idêntica.
	Reproduzida bool
}

// ConcluirValidacao executa o fluxo 8 dos requisitos: fecha a validação,
// rotaciona o contador quando o novo código foi programado e publica
// validacao.concluida. É idempotente: repetir a mesma conclusão devolve o mesmo
// resultado; uma conclusão diferente sobre validação já concluída é conflito.
func (s *Servico) ConcluirValidacao(ctx context.Context, e EntradaConcluirValidacao) (*ResultadoConclusao, error) {
	operador, err := s.operadorAtivo(ctx, e.Uid)
	if err != nil {
		return nil, err
	}
	v, err := s.validacaoDoOperador(ctx, e.IdValidacao, e.Uid)
	if err != nil {
		return nil, err
	}
	agora := s.dep.Relogio.Agora()

	if v.Status == dominio.StatusConcluida {
		if v.MesmaConclusao(e.Leituras, e.CodigoRotacionado) {
			m, err := s.dep.Maquinas.ObterMaquina(ctx, v.IdMaquina)
			if err != nil {
				return nil, indisponivel(err)
			}
			return &ResultadoConclusao{Validacao: v, Maquina: m, Reproduzida: true}, nil
		}
		return nil, dominio.NovoErro(dominio.CodigoValidacaoJaConcluida,
			"a validação já foi concluída com outros valores").ComValidacao(v.Id)
	}
	if v.Status == dominio.StatusExpirada || v.Expirou(agora) {
		if v.Status != dominio.StatusExpirada {
			v.MarcarExpirada()
			if err := s.dep.Validacoes.MarcarExpirada(ctx, v.Id); err != nil {
				s.dep.Registrador.WarnContext(ctx, "falha ao marcar validação expirada", "idValidacao", v.Id, "erro", err)
			}
		}
		return nil, dominio.NovoErro(dominio.CodigoValidacaoExpirada,
			"a validação expirou; leia o QR novamente").ComValidacao(v.Id)
	}

	// O histórico da máquina alimenta os avisos da RN-07 antes da gravação.
	historico, err := s.dep.Maquinas.ObterMaquina(ctx, v.IdMaquina)
	if err != nil {
		return nil, indisponivel(err)
	}
	if err := v.Concluir(e.Leituras, e.CodigoRotacionado, e.Observacoes, agora); err != nil {
		return nil, err
	}
	v.RegistrarAvisos(historico)
	maquina, err := s.dep.Validacoes.Concluir(ctx, v, agora)
	if err != nil {
		return nil, indisponivel(err)
	}

	s.publicar(ctx, dominio.NovoEventoConcluida(v, maquina, operador), v, agora)
	return &ResultadoConclusao{Validacao: v, Maquina: maquina}, nil
}
