package aplicacao

import (
	"context"
	"errors"
	"time"

	"github.com/ThiagoCristovao/serverless-vending-validation/servico/internal/dominio"
	"github.com/ThiagoCristovao/serverless-vending-validation/servico/internal/portas"
)

// EntradaIniciarValidacao é o pedido de POST /v1/validacoes já autenticado.
type EntradaIniciarValidacao struct {
	// Uid é o operador autenticado (claim do JWT repassada pelo gateway).
	Uid string
	// ChaveIdempotencia é o cabeçalho Idempotency-Key.
	ChaveIdempotencia string
	// PayloadQr é o JSON bruto lido do adesivo.
	PayloadQr []byte
	// LidoEm e Dispositivo são informativos.
	LidoEm      *time.Time
	Dispositivo *dominio.Dispositivo
}

// IniciarValidacao executa o fluxo 4–5 dos requisitos: valida o adesivo, o
// operador e a máquina, abre a validação, deriva as contrassenhas e publica
// validacao.iniciada.
func (s *Servico) IniciarValidacao(ctx context.Context, e EntradaIniciarValidacao) (*ResultadoValidacao, error) {
	if e.Uid == "" {
		return nil, dominio.NovoErro(dominio.CodigoOperadorNaoAutorizado, "operador não identificado")
	}
	if e.ChaveIdempotencia == "" {
		return nil, dominio.NovoErro(dominio.CodigoPayloadInvalido, "o cabeçalho Idempotency-Key é obrigatório")
	}
	payload, err := dominio.DecodificarPayloadQr(e.PayloadQr)
	if err != nil {
		return nil, err
	}
	agora := s.dep.Relogio.Agora()

	// Idempotência (ADR-0009): mesma chave e mesmo conteúdo devolvem a mesma
	// resposta; mesma chave com conteúdo diferente é conflito.
	existente, err := s.dep.Validacoes.ObterPorChaveIdempotencia(ctx, e.ChaveIdempotencia)
	switch {
	case err == nil:
		if existente.HashRequisicao != payload.HashRequisicao() || existente.IdOperador != e.Uid {
			return nil, dominio.NovoErro(dominio.CodigoChaveIdempotenciaConflitante,
				"Idempotency-Key já usada com outro conteúdo").ComValidacao(existente.Id)
		}
		r, err := s.montarResultado(ctx, existente, agora)
		if err != nil {
			return nil, err
		}
		r.Reproduzida = true
		return r, nil
	case !errors.Is(err, portas.ErrNaoEncontrado):
		return nil, indisponivel(err)
	}

	operador, err := s.operadorAtivo(ctx, e.Uid)
	if err != nil {
		return nil, err
	}

	chave, err := s.dep.Chaves.ObterChaveQr(ctx, payload.Kid)
	switch {
	case errors.Is(err, portas.ErrNaoEncontrado), err == nil && !chave.Ativa:
		return nil, dominio.NovoErro(dominio.CodigoChaveQrDesconhecida, "a chave que assinou o adesivo não é reconhecida")
	case err != nil:
		return nil, indisponivel(err)
	}
	if err := payload.VerificarAssinatura(chave.ChavePublica); err != nil {
		return nil, err
	}
	if payload.Vencido(agora) {
		return nil, dominio.NovoErro(dominio.CodigoQrExpirado, "o adesivo expirou em "+payload.Exp+"; solicite a substituição")
	}

	maquina, err := s.dep.Maquinas.ObterMaquina(ctx, payload.Maq)
	switch {
	case errors.Is(err, portas.ErrNaoEncontrado):
		return nil, dominio.NovoErro(dominio.CodigoMaquinaDesconhecida, "nenhuma máquina cadastrada com o identificador "+payload.Maq)
	case err != nil:
		return nil, indisponivel(err)
	case !maquina.Ativa:
		return nil, dominio.NovoErro(dominio.CodigoMaquinaInativa, "a máquina está inativa")
	case maquina.Modelo != payload.Mod || maquina.Localizacao.Id != payload.Loc:
		return nil, dominio.NovoErro(dominio.CodigoAssinaturaInvalida, "o adesivo não corresponde ao cadastro da máquina")
	}

	id, err := s.dep.Ids.NovoId()
	if err != nil {
		return nil, dominio.NovoErro(dominio.CodigoErroInterno, "não foi possível gerar o identificador").ComCausa(err)
	}
	v := dominio.NovaValidacao(id, maquina, operador, payload, e.ChaveIdempotencia, e.Dispositivo, agora, s.dep.PrazoValidacao)
	if err := s.dep.Validacoes.Abrir(ctx, v, agora); err != nil {
		var emAndamento *portas.ErrValidacaoEmAndamento
		if errors.As(err, &emAndamento) {
			return nil, dominio.NovoErro(dominio.CodigoValidacaoEmAndamento,
				"a máquina já possui uma validação em andamento").ComValidacao(emAndamento.IdValidacao)
		}
		return nil, indisponivel(err)
	}

	s.publicar(ctx, dominio.NovoEventoIniciada(v, maquina, operador), v, agora)
	return s.resultado(v, maquina), nil
}
