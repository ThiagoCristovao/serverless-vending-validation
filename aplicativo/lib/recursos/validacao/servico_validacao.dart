import 'modelos.dart';

/// Porta do aplicativo para o serviço de validação (api/openapi.yaml). A
/// implementação real é o cliente HTTP; os testes de interface usam um falso.
abstract interface class ServicoValidacao {
  /// POST /v1/validacoes. `payloadQr` é o JSON bruto lido do adesivo.
  Future<Validacao> iniciar({
    required String payloadQr,
    required String chaveIdempotencia,
  });

  /// GET /v1/validacoes/{id}.
  Future<Validacao> consultar(String idValidacao);

  /// POST /v1/validacoes/{id}/conclusao.
  Future<Conclusao> concluir(
    String idValidacao, {
    required Leituras leituras,
    required bool codigoRotacionado,
    String? observacoes,
  });
}
