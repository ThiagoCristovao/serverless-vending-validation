/// Configuração de compilação do aplicativo. Valores podem ser trocados com
/// `--dart-define` sem alterar o código (ex.: outro ambiente do gateway).
abstract final class Configuracao {
  /// Base das chamadas ao serviço: host do API Gateway + caminho /v1.
  static const apiBase = String.fromEnvironment(
    'SVV_API_BASE',
    defaultValue: 'https://svv-dev-gateway-4mjof8z9.ue.gateway.dev/v1',
  );

  static const versaoApp = String.fromEnvironment(
    'SVV_VERSAO_APP',
    defaultValue: '0.1.0',
  );

  static const plataforma = 'android';
}
