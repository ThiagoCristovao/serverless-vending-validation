// Package aplicacao implementa os casos de uso do serviço de validação
// (iniciar, concluir e consultar), orquestrando o domínio por meio das portas.
package aplicacao

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/ThiagoCristovao/serverless-vending-validation/servico/internal/dominio"
	"github.com/ThiagoCristovao/serverless-vending-validation/servico/internal/portas"
)

// Dependencias reúne as portas e a configuração de que o serviço precisa.
type Dependencias struct {
	Maquinas     portas.RepositorioMaquinas
	Operadores   portas.RepositorioOperadores
	Chaves       portas.ProvedorChavesQr
	Validacoes   portas.RepositorioValidacoes
	Publicador   portas.PublicadorEventos
	Relogio      portas.Relogio
	Ids          portas.GeradorIdentificador
	Contrassenha *dominio.GeradorContrassenha

	// PrazoValidacao é o tempo para concluir uma validação (RN-04). Zero usa o padrão.
	PrazoValidacao time.Duration
	// Registrador recebe avisos operacionais; nil usa slog.Default().
	Registrador *slog.Logger
}

// Servico expõe os casos de uso.
type Servico struct {
	dep Dependencias
}

// Novo valida as dependências obrigatórias e aplica os padrões.
func Novo(dep Dependencias) (*Servico, error) {
	switch {
	case dep.Maquinas == nil, dep.Operadores == nil, dep.Chaves == nil, dep.Validacoes == nil,
		dep.Publicador == nil, dep.Relogio == nil, dep.Ids == nil, dep.Contrassenha == nil:
		return nil, errors.New("aplicacao: todas as portas e o gerador de contrassenha são obrigatórios")
	}
	if dep.PrazoValidacao <= 0 {
		dep.PrazoValidacao = dominio.PrazoValidacaoPadrao
	}
	if dep.Registrador == nil {
		dep.Registrador = slog.Default()
	}
	return &Servico{dep: dep}, nil
}

// ResultadoValidacao é a visão de uma validação devolvida ao aplicativo.
// Contrassenha e ProximaContrassenha só vêm preenchidas com status aberta.
type ResultadoValidacao struct {
	Validacao           *dominio.Validacao
	Maquina             *dominio.Maquina
	Contrassenha        string
	ProximaContrassenha string
	// Reproduzida indica resposta repetida por idempotência, sem efeitos novos.
	Reproduzida bool
}

func (s *Servico) resultado(v *dominio.Validacao, m *dominio.Maquina) *ResultadoValidacao {
	r := &ResultadoValidacao{Validacao: v, Maquina: m}
	if v.Aberta() {
		r.Contrassenha = s.dep.Contrassenha.Derivar(v.IdMaquina, v.ContadorCodigo)
		r.ProximaContrassenha = s.dep.Contrassenha.Derivar(v.IdMaquina, v.ContadorCodigo+1)
	}
	return r
}

// montarResultado carrega a máquina, aplica a expiração preguiçosa (RN-04) e
// monta a resposta.
func (s *Servico) montarResultado(ctx context.Context, v *dominio.Validacao, agora time.Time) (*ResultadoValidacao, error) {
	if v.Expirou(agora) {
		v.MarcarExpirada()
		if err := s.dep.Validacoes.MarcarExpirada(ctx, v.Id); err != nil {
			s.dep.Registrador.WarnContext(ctx, "falha ao marcar validação expirada", "idValidacao", v.Id, "erro", err)
		}
	}
	m, err := s.dep.Maquinas.ObterMaquina(ctx, v.IdMaquina)
	if err != nil {
		return nil, indisponivel(err)
	}
	return s.resultado(v, m), nil
}

// operadorAtivo carrega o operador e exige que esteja ativo (RN-06).
func (s *Servico) operadorAtivo(ctx context.Context, uid string) (*dominio.Operador, error) {
	if uid == "" {
		return nil, dominio.NovoErro(dominio.CodigoOperadorNaoAutorizado, "operador não identificado")
	}
	o, err := s.dep.Operadores.ObterOperador(ctx, uid)
	switch {
	case errors.Is(err, portas.ErrNaoEncontrado):
		return nil, dominio.NovoErro(dominio.CodigoOperadorNaoAutorizado, "operador não cadastrado")
	case err != nil:
		return nil, indisponivel(err)
	case !o.Ativo:
		return nil, dominio.NovoErro(dominio.CodigoOperadorNaoAutorizado, "operador inativo; procure a administração da rede")
	}
	return o, nil
}

// validacaoDoOperador carrega a validação e exige que pertença ao operador.
func (s *Servico) validacaoDoOperador(ctx context.Context, id, uid string) (*dominio.Validacao, error) {
	if !dominio.UlidValido(id) {
		return nil, dominio.NovoErro(dominio.CodigoValidacaoNaoEncontrada, "validação não encontrada")
	}
	v, err := s.dep.Validacoes.ObterValidacao(ctx, id)
	switch {
	case errors.Is(err, portas.ErrNaoEncontrado):
		return nil, dominio.NovoErro(dominio.CodigoValidacaoNaoEncontrada, "validação não encontrada")
	case err != nil:
		return nil, indisponivel(err)
	case v.IdOperador != uid:
		return nil, dominio.NovoErro(dominio.CodigoOperadorNaoAutorizado, "a validação pertence a outro operador").ComValidacao(v.Id)
	}
	return v, nil
}

// publicar envia o evento e registra a publicação. Falhas não interrompem o
// fluxo do operador (RNF-04): ficam como pendência para reconciliação (CE-14).
func (s *Servico) publicar(ctx context.Context, evento dominio.Evento, v *dominio.Validacao, agora time.Time) {
	if err := s.dep.Publicador.Publicar(ctx, evento); err != nil {
		s.dep.Registrador.WarnContext(ctx, "falha ao publicar evento; publicação pendente",
			"tipo", string(evento.Tipo), "idValidacao", v.Id, "idMaquina", v.IdMaquina, "erro", err)
		return
	}
	if err := s.dep.Validacoes.MarcarEventoPublicado(ctx, v.Id, evento.Tipo, agora); err != nil {
		s.dep.Registrador.WarnContext(ctx, "falha ao registrar publicação do evento",
			"tipo", string(evento.Tipo), "idValidacao", v.Id, "erro", err)
		return
	}
	quando := agora
	switch evento.Tipo {
	case dominio.EventoValidacaoIniciada:
		v.Eventos.IniciadaEm = &quando
	case dominio.EventoValidacaoConcluida:
		v.Eventos.ConcluidaEm = &quando
	}
}

func indisponivel(causa error) error {
	return dominio.NovoErro(dominio.CodigoDependenciaIndisponivel, "não foi possível acessar o armazenamento").ComCausa(causa)
}
