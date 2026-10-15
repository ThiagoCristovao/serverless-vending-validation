import 'chaves_qr.dart';
import 'payload_qr.dart';
import '../../compartilhado/erros.dart';

/// Validação local do adesivo (RF-04 e RF-05): estrutura e formato, chave
/// conhecida, assinatura e validade. Tudo sem rede; o serviço repete as mesmas
/// verificações com relógio e cadastro autoritativos.
class ValidadorLocal {
  const ValidadorLocal({this.agora});

  /// Fonte do instante atual; injetável nos testes.
  final DateTime Function()? agora;

  Future<PayloadQr> validar(String bruto) async {
    final payload = PayloadQr.decodificar(bruto);
    final chave = ChavesQr.publica(payload.kid);
    if (chave == null) {
      throw const ErroDominio(
        CodigoErro.chaveQrDesconhecida,
        'a chave que assinou o adesivo não é reconhecida',
      );
    }
    await payload.verificarAssinatura(chave);
    if (payload.vencido((agora ?? DateTime.now)())) {
      throw ErroDominio(
        CodigoErro.qrExpirado,
        'o adesivo expirou em ${payload.exp}',
      );
    }
    return payload;
  }
}
