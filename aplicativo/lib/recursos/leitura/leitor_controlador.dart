import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../compartilhado/erros.dart';
import 'payload_qr.dart';
import 'validador_local.dart';

/// Estados da tela de leitura (docs/arquitetura.md, seção 12).
sealed class EstadoLeitura {
  const EstadoLeitura();
}

/// Câmera aberta, aguardando um QR.
class Aguardando extends EstadoLeitura {
  const Aguardando();
}

/// Um código foi lido e está sendo validado localmente.
class Decodificando extends EstadoLeitura {
  const Decodificando();
}

/// O código lido foi rejeitado pela validação local (CE-06, CE-07, CE-08).
class LeituraInvalida extends EstadoLeitura {
  const LeituraInvalida(this.erro);
  final ErroDominio erro;
}

/// Adesivo válido: dados decodificados prontos para a requisição (Sprint 7).
class LeituraValida extends EstadoLeitura {
  const LeituraValida(this.payload);
  final PayloadQr payload;
}

final validadorLocalProvider = Provider<ValidadorLocal>(
  (ref) => const ValidadorLocal(),
);

/// Controla a leitura: ignora leituras repetidas enquanto processa ou enquanto
/// um resultado está em exibição, e volta a aguardar sob comando do operador.
class LeitorControlador extends Notifier<EstadoLeitura> {
  @override
  EstadoLeitura build() => const Aguardando();

  Future<void> processar(String? bruto) async {
    if (state is! Aguardando || bruto == null || bruto.isEmpty) return;
    state = const Decodificando();
    try {
      final payload = await ref.read(validadorLocalProvider).validar(bruto);
      state = LeituraValida(payload);
    } on ErroDominio catch (e) {
      state = LeituraInvalida(e);
    } catch (e) {
      state = LeituraInvalida(
        ErroDominio(CodigoErro.payloadInvalido, e.toString()),
      );
    }
  }

  void reiniciar() => state = const Aguardando();
}

final leitorProvider = NotifierProvider<LeitorControlador, EstadoLeitura>(
  LeitorControlador.new,
);
