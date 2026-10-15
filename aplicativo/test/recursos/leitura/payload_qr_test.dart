import 'package:flutter_test/flutter_test.dart';
import 'package:svv/compartilhado/erros.dart';
import 'package:svv/recursos/leitura/chaves_qr.dart';
import 'package:svv/recursos/leitura/payload_qr.dart';
import 'package:svv/recursos/leitura/validador_local.dart';

// Vetores gerados pela ferramenta Go `servico/cmd/gerar-qr` com a chave k1 de
// desenvolvimento: garantem que Dart e Go validam o mesmo adesivo do mesmo jeito.
const valido =
    '{"v":1,"maq":"VM-2047","mod":"CN168","loc":"BLA-T","exp":"2027-09-25","kid":"k1","sig":"MsxhS6H59KpgEjML9psz4vihfRHusUTyDuMlAjdTERGRzOMa9QWb-q_7eCGx3aXFx2ieCxO2Gf2DQ8alyDeuDg"}';
const vencido =
    '{"v":1,"maq":"VM-2047","mod":"CN168","loc":"BLA-T","exp":"2025-12-31","kid":"k1","sig":"EcaeFiZCBpxPprDxe06N2hj1SGlMQjtGCLlORwP5MfKvj9qIuOzuHOtaDrcUHtiqn8Xr_Q5l21HogkQtC8GgBA"}';
const outraMaquina =
    '{"v":1,"maq":"VM-0009","mod":"CN168","loc":"BLB-1","exp":"2027-10-15","kid":"k1","sig":"3etsPAATkI3ZVxPmsRYjYW_eggTlh61ugy8g2C8vHP6vtJ9XZXAr1mfh7B9htSZGr6q2_oNLfthtYYLOhUfXBw"}';

final referencia = DateTime.utc(2026, 10, 15, 12);

Matcher lancaCodigo(CodigoErro codigo) =>
    throwsA(isA<ErroDominio>().having((e) => e.codigo, 'codigo', codigo));

void main() {
  group('PayloadQr.decodificar', () {
    test('aceita o adesivo gerado pela ferramenta Go', () {
      final p = PayloadQr.decodificar(valido);
      expect(p.maq, 'VM-2047');
      expect(p.mod, 'CN168');
      expect(p.mensagemCanonica, '1|VM-2047|CN168|BLA-T|2027-09-25|k1');
      expect(p.sig.length, 86);
    });

    test(
      'rejeita JSON inválido, campos extras ou ausentes e formatos errados',
      () {
        expect(
          () => PayloadQr.decodificar('{"v":1,'),
          lancaCodigo(CodigoErro.payloadInvalido),
        );
        expect(
          () => PayloadQr.decodificar('[1,2]'),
          lancaCodigo(CodigoErro.payloadInvalido),
        );
        expect(
          () => PayloadQr.decodificar(valido.replaceFirst('}', ',"extra":1}')),
          lancaCodigo(CodigoErro.payloadInvalido),
        );
        expect(
          () =>
              PayloadQr.decodificar(valido.replaceFirst('"loc":"BLA-T",', '')),
          lancaCodigo(CodigoErro.payloadInvalido),
        );
        expect(
          () => PayloadQr.decodificar(valido.replaceFirst('"v":1', '"v":2')),
          lancaCodigo(CodigoErro.payloadInvalido),
        );
        expect(
          () => PayloadQr.decodificar(valido.replaceFirst('"v":1', '"v":"1"')),
          lancaCodigo(CodigoErro.payloadInvalido),
        );
        expect(
          () => PayloadQr.decodificar(valido.replaceFirst('VM-2047', 'vm-1')),
          lancaCodigo(CodigoErro.payloadInvalido),
        );
        expect(
          () => PayloadQr.decodificar(
            valido.replaceFirst('"kid":"k1"', '"kid":"K1"'),
          ),
          lancaCodigo(CodigoErro.payloadInvalido),
        );
        expect(
          () => PayloadQr.decodificar(
            valido.replaceFirst('2027-09-25', '25/09/2027'),
          ),
          lancaCodigo(CodigoErro.payloadInvalido),
        );
        expect(
          () => PayloadQr.decodificar(
            valido.replaceFirst('2027-09-25', '2027-02-30'),
          ),
          lancaCodigo(CodigoErro.payloadInvalido),
        );
      },
    );

    test('rejeita modelo não suportado', () {
      expect(
        () => PayloadQr.decodificar(valido.replaceFirst('CN168', 'CN167')),
        lancaCodigo(CodigoErro.modeloNaoSuportado),
      );
    });
  });

  group('validade', () {
    final p = PayloadQr.decodificar(valido);
    test('vale até o fim do dia exp em UTC', () {
      expect(p.vencido(DateTime.utc(2027, 9, 25, 23, 59, 59)), isFalse);
      expect(p.vencido(DateTime.utc(2027, 9, 26)), isTrue);
      expect(p.vencido(referencia), isFalse);
    });
  });

  group('assinatura Ed25519', () {
    final chave = ChavesQr.publica('k1')!;
    test('confere a assinatura produzida em Go', () async {
      await PayloadQr.decodificar(valido).verificarAssinatura(chave);
      await PayloadQr.decodificar(outraMaquina).verificarAssinatura(chave);
    });
    test('rejeita campo adulterado e assinatura de outro conteúdo', () async {
      final adulterado = PayloadQr.decodificar(
        valido.replaceFirst('BLA-T', 'BLB-T'),
      );
      await expectLater(
        adulterado.verificarAssinatura(chave),
        lancaCodigo(CodigoErro.assinaturaInvalida),
      );
      final trocada = PayloadQr.decodificar(
        valido.replaceFirst(
          RegExp(r'"sig":"[^"]+"'),
          '"sig":"${PayloadQr.decodificar(vencido).sig}"',
        ),
      );
      await expectLater(
        trocada.verificarAssinatura(chave),
        lancaCodigo(CodigoErro.assinaturaInvalida),
      );
    });
  });

  group('ValidadorLocal', () {
    final validador = ValidadorLocal(agora: () => referencia);
    test('aceita adesivo válido', () async {
      final p = await validador.validar(valido);
      expect(p.maq, 'VM-2047');
    });
    test('rejeita vencido, chave desconhecida e adulterado', () async {
      await expectLater(
        validador.validar(vencido),
        lancaCodigo(CodigoErro.qrExpirado),
      );
      await expectLater(
        validador.validar(valido.replaceFirst('"kid":"k1"', '"kid":"k9"')),
        lancaCodigo(CodigoErro.chaveQrDesconhecida),
      );
      await expectLater(
        validador.validar(valido.replaceFirst('BLA-T', 'BLB-T')),
        lancaCodigo(CodigoErro.assinaturaInvalida),
      );
    });
    test('não decide sobre máquina trocada: isso é do serviço', () async {
      final p = await validador.validar(outraMaquina);
      expect(p.maq, 'VM-0009');
    });
  });
}
