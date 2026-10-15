import 'dart:convert';
import 'dart:typed_data';

/// Chaves públicas Ed25519 que assinam os adesivos, embutidas no aplicativo
/// para a validação local (RF-05). Só chaves públicas: não há segredo aqui.
///
/// A chave `k1` abaixo é a do ambiente de desenvolvimento. Em outro ambiente,
/// passe `--dart-define=SVV_CHAVE_PUBLICA_K1=<base64url>` na compilação.
class ChavesQr {
  ChavesQr._();

  static const _k1 = String.fromEnvironment(
    'SVV_CHAVE_PUBLICA_K1',
    defaultValue: 'Q04S-msOO-YksuYO-T4WOdycr0irQd4usTGi6vd-CYw',
  );

  static const Map<String, String> _embutidas = {'k1': _k1};

  /// Devolve a chave pública (32 bytes) do identificador dado, ou null.
  static Uint8List? publica(String kid) {
    final codificada = _embutidas[kid];
    if (codificada == null || codificada.isEmpty) return null;
    final bytes = base64Url.decode(base64Url.normalize(codificada));
    return bytes.length == 32 ? Uint8List.fromList(bytes) : null;
  }
}
