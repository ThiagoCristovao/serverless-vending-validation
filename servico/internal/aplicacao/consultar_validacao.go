package aplicacao

import "context"

// ConsultarValidacao atende GET /v1/validacoes/{id}: devolve o estado atual da
// validação do próprio operador, com as contrassenhas apenas enquanto aberta.
// Usado pelo aplicativo para recuperar o estado após falha de rede (CE-03).
func (s *Servico) ConsultarValidacao(ctx context.Context, uid, idValidacao string) (*ResultadoValidacao, error) {
	if _, err := s.operadorAtivo(ctx, uid); err != nil {
		return nil, err
	}
	v, err := s.validacaoDoOperador(ctx, idValidacao, uid)
	if err != nil {
		return nil, err
	}
	return s.montarResultado(ctx, v, s.dep.Relogio.Agora())
}
