package aplicacao

import (
	"context"
	"errors"
	"time"

	"github.com/ThiagoCristovao/serverless-vending-validation/servico/internal/dominio"
	"github.com/ThiagoCristovao/serverless-vending-validation/servico/internal/portas"
)

// RelatorioReconciliacao resume uma execução da reconciliação.
type RelatorioReconciliacao struct {
	Examinadas int `json:"examinadas"`
	Publicadas int `json:"publicadas"`
	Falhas     int `json:"falhas"`
}

// ReconciliarPublicacoes republica os eventos cuja publicação ficou pendente
// (CE-14, ADR-0012). É chamada periodicamente pelo Cloud Scheduler. O
// consumidor é idempotente, então uma republicação repetida é inócua.
func (s *Servico) ReconciliarPublicacoes(ctx context.Context) (RelatorioReconciliacao, error) {
	var rel RelatorioReconciliacao
	agora := s.dep.Relogio.Agora()
	pendentes, err := s.dep.Validacoes.PublicacoesPendentes(ctx, agora.Add(-s.dep.AtrasoReconciliacao), s.dep.LoteReconciliacao)
	if err != nil {
		return rel, indisponivel(err)
	}
	for _, v := range pendentes {
		rel.Examinadas++
		m, err := s.dep.Maquinas.ObterMaquina(ctx, v.IdMaquina)
		if err != nil {
			rel.Falhas++
			s.dep.Registrador.WarnContext(ctx, "reconciliação: máquina indisponível", "idValidacao", v.Id, "idMaquina", v.IdMaquina, "erro", err)
			continue
		}
		o, err := s.dep.Operadores.ObterOperador(ctx, v.IdOperador)
		if err != nil {
			if !errors.Is(err, portas.ErrNaoEncontrado) {
				rel.Falhas++
				s.dep.Registrador.WarnContext(ctx, "reconciliação: operador indisponível", "idValidacao", v.Id, "erro", err)
				continue
			}
			o = &dominio.Operador{Uid: v.IdOperador}
		}
		if v.Eventos.IniciadaEm == nil {
			if !s.republicar(ctx, dominio.NovoEventoIniciada(v, m, o), v, agora) {
				rel.Falhas++
				continue
			}
			rel.Publicadas++
		}
		if v.Status == dominio.StatusConcluida && v.Eventos.ConcluidaEm == nil {
			if !s.republicar(ctx, dominio.NovoEventoConcluida(v, m, o), v, agora) {
				rel.Falhas++
				continue
			}
			rel.Publicadas++
		}
	}
	return rel, nil
}

func (s *Servico) republicar(ctx context.Context, evento dominio.Evento, v *dominio.Validacao, agora time.Time) bool {
	if err := s.dep.Publicador.Publicar(ctx, evento); err != nil {
		s.dep.Registrador.WarnContext(ctx, "reconciliação: publicação falhou", "tipo", string(evento.Tipo), "idValidacao", v.Id, "erro", err)
		return false
	}
	if err := s.dep.Validacoes.MarcarEventoPublicado(ctx, v.Id, evento.Tipo, agora); err != nil {
		s.dep.Registrador.WarnContext(ctx, "reconciliação: falha ao gravar marcador", "tipo", string(evento.Tipo), "idValidacao", v.Id, "erro", err)
		return false
	}
	return true
}
