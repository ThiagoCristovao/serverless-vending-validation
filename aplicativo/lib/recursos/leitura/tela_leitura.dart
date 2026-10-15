import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:mobile_scanner/mobile_scanner.dart';

import '../../compartilhado/erros.dart';
import '../autenticacao/sessao.dart';
import 'leitor_controlador.dart';
import 'payload_qr.dart';

/// Tela de leitura do QR (Figura 2a do TCC 1). A câmera ocupa a tela; um
/// painel inferior mostra o estado: aguardando, decodificando, código inválido
/// ou código reconhecido com os dados do adesivo.
class TelaLeitura extends ConsumerStatefulWidget {
  const TelaLeitura({super.key});

  @override
  ConsumerState<TelaLeitura> createState() => _TelaLeituraState();
}

class _TelaLeituraState extends ConsumerState<TelaLeitura> {
  final _controlador = MobileScannerController(
    formats: const [BarcodeFormat.qrCode],
    detectionSpeed: DetectionSpeed.noDuplicates,
  );

  @override
  void dispose() {
    _controlador.dispose();
    super.dispose();
  }

  Future<void> _aoDetectar(BarcodeCapture captura) async {
    final bruto = captura.barcodes.isEmpty
        ? null
        : captura.barcodes.first.rawValue;
    if (bruto == null) return;
    await _controlador.stop();
    await ref.read(leitorProvider.notifier).processar(bruto);
  }

  Future<void> _lerNovamente() async {
    ref.read(leitorProvider.notifier).reiniciar();
    await _controlador.start();
  }

  @override
  Widget build(BuildContext context) {
    final estado = ref.watch(leitorProvider);
    final operador = ref.watch(sessaoProvider).value;

    return Scaffold(
      appBar: AppBar(
        title: const Text('Validar máquina'),
        actions: [
          IconButton(
            tooltip: 'Lanterna',
            icon: const Icon(Icons.flashlight_on_outlined),
            onPressed: () => _controlador.toggleTorch(),
          ),
          IconButton(
            tooltip: 'Sair',
            icon: const Icon(Icons.logout),
            onPressed: () => ref.read(autenticacaoProvider.notifier).sair(),
          ),
        ],
      ),
      body: Column(
        children: [
          Expanded(
            child: Stack(
              fit: StackFit.expand,
              children: [
                MobileScanner(
                  controller: _controlador,
                  onDetect: _aoDetectar,
                  errorBuilder: (context, erro) =>
                      _ErroDeCamera(erro: erro, aoTentar: _lerNovamente),
                ),
                const _Moldura(),
                if (estado is Aguardando)
                  const Positioned(
                    left: 24,
                    right: 24,
                    top: 24,
                    child: _Dica('Aponte a câmera para o QR code da máquina'),
                  ),
              ],
            ),
          ),
          _PainelDeEstado(estado: estado, aoLerNovamente: _lerNovamente),
          _Rodape(operador?.email ?? ''),
        ],
      ),
    );
  }
}

class _PainelDeEstado extends StatelessWidget {
  const _PainelDeEstado({required this.estado, required this.aoLerNovamente});

  final EstadoLeitura estado;
  final Future<void> Function() aoLerNovamente;

  @override
  Widget build(BuildContext context) {
    final esquema = Theme.of(context).colorScheme;
    return AnimatedSize(
      duration: const Duration(milliseconds: 200),
      child: Padding(
        padding: const EdgeInsets.fromLTRB(16, 12, 16, 8),
        child: switch (estado) {
          Aguardando() => const SizedBox(height: 8),
          Decodificando() => const Padding(
            padding: EdgeInsets.all(12),
            child: Row(
              mainAxisAlignment: MainAxisAlignment.center,
              children: [
                SizedBox(
                  height: 20,
                  width: 20,
                  child: CircularProgressIndicator(strokeWidth: 2.5),
                ),
                SizedBox(width: 12),
                Text('Verificando o código…'),
              ],
            ),
          ),
          LeituraInvalida(:final erro) => Card(
            color: esquema.errorContainer,
            child: Padding(
              padding: const EdgeInsets.all(16),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  Row(
                    children: [
                      Icon(
                        Icons.error_outline,
                        color: esquema.onErrorContainer,
                      ),
                      const SizedBox(width: 12),
                      Expanded(
                        child: Text(
                          mensagemParaOperador(erro.codigo),
                          style: TextStyle(color: esquema.onErrorContainer),
                        ),
                      ),
                    ],
                  ),
                  const SizedBox(height: 12),
                  FilledButton.tonal(
                    onPressed: aoLerNovamente,
                    child: const Text('Tentar novamente'),
                  ),
                ],
              ),
            ),
          ),
          LeituraValida(:final payload) => _CartaoDecodificado(
            payload: payload,
            aoLerNovamente: aoLerNovamente,
          ),
        },
      ),
    );
  }
}

class _CartaoDecodificado extends StatelessWidget {
  const _CartaoDecodificado({
    required this.payload,
    required this.aoLerNovamente,
  });

  final PayloadQr payload;
  final Future<void> Function() aoLerNovamente;

  @override
  Widget build(BuildContext context) {
    final esquema = Theme.of(context).colorScheme;
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Row(
              children: [
                Icon(Icons.verified_outlined, color: esquema.primary),
                const SizedBox(width: 8),
                Text(
                  'Código reconhecido',
                  style: Theme.of(context).textTheme.titleMedium,
                ),
              ],
            ),
            const SizedBox(height: 12),
            _Linha('Máquina', payload.maq),
            _Linha('Modelo', payload.mod),
            _Linha('Localização', payload.loc),
            _Linha('Adesivo válido até', payload.exp),
            _Linha('Chave', payload.kid),
            const SizedBox(height: 16),
            // Sprint 7: aqui a requisição ao serviço de validação é disparada.
            FilledButton(
              onPressed: null,
              child: const Text('Validar na nuvem (Sprint 7)'),
            ),
            const SizedBox(height: 8),
            OutlinedButton(
              onPressed: aoLerNovamente,
              child: const Text('Ler outro código'),
            ),
          ],
        ),
      ),
    );
  }
}

class _Linha extends StatelessWidget {
  const _Linha(this.rotulo, this.valor);

  final String rotulo;
  final String valor;

  @override
  Widget build(BuildContext context) {
    final estilo = Theme.of(context).textTheme;
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 3),
      child: Row(
        mainAxisAlignment: MainAxisAlignment.spaceBetween,
        children: [
          Text(
            rotulo,
            style: estilo.bodyMedium?.copyWith(color: estilo.bodySmall?.color),
          ),
          Text(
            valor,
            style: estilo.bodyLarge?.copyWith(fontWeight: FontWeight.w600),
          ),
        ],
      ),
    );
  }
}

class _ErroDeCamera extends StatelessWidget {
  const _ErroDeCamera({required this.erro, required this.aoTentar});

  final MobileScannerException erro;
  final Future<void> Function() aoTentar;

  @override
  Widget build(BuildContext context) {
    final negada = erro.errorCode == MobileScannerErrorCode.permissionDenied;
    return Container(
      color: Theme.of(context).scaffoldBackgroundColor,
      padding: const EdgeInsets.all(24),
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          Icon(
            negada
                ? Icons.no_photography_outlined
                : Icons.videocam_off_outlined,
            size: 56,
          ),
          const SizedBox(height: 16),
          Text(
            negada
                ? mensagemParaOperador(CodigoErro.cameraNegada)
                : 'Não foi possível abrir a câmera (${erro.errorCode.name}).',
            textAlign: TextAlign.center,
          ),
          const SizedBox(height: 16),
          FilledButton.tonal(
            onPressed: aoTentar,
            child: const Text('Tentar novamente'),
          ),
        ],
      ),
    );
  }
}

class _Moldura extends StatelessWidget {
  const _Moldura();

  @override
  Widget build(BuildContext context) {
    final cor = Theme.of(context).colorScheme.primary;
    return IgnorePointer(
      child: Center(
        child: FractionallySizedBox(
          widthFactor: 0.72,
          child: AspectRatio(
            aspectRatio: 1,
            child: CustomPaint(painter: _MolduraPintor(cor)),
          ),
        ),
      ),
    );
  }
}

class _MolduraPintor extends CustomPainter {
  _MolduraPintor(this.cor);

  final Color cor;

  @override
  void paint(Canvas canvas, Size size) {
    final tinta = Paint()
      ..color = cor
      ..strokeWidth = 4
      ..strokeCap = StrokeCap.round
      ..style = PaintingStyle.stroke;
    const canto = 28.0;
    final w = size.width;
    final h = size.height;
    final cantos = [
      [Offset(0, canto), Offset.zero, Offset(canto, 0)],
      [Offset(w - canto, 0), Offset(w, 0), Offset(w, canto)],
      [Offset(w, h - canto), Offset(w, h), Offset(w - canto, h)],
      [Offset(canto, h), Offset(0, h), Offset(0, h - canto)],
    ];
    for (final c in cantos) {
      canvas.drawPath(
        Path()
          ..moveTo(c[0].dx, c[0].dy)
          ..lineTo(c[1].dx, c[1].dy)
          ..lineTo(c[2].dx, c[2].dy),
        tinta,
      );
    }
    canvas.drawLine(
      Offset(12, h / 2),
      Offset(w - 12, h / 2),
      tinta..strokeWidth = 2,
    );
  }

  @override
  bool shouldRepaint(covariant _MolduraPintor antigo) => antigo.cor != cor;
}

class _Dica extends StatelessWidget {
  const _Dica(this.texto);

  final String texto;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
      decoration: BoxDecoration(
        color: Colors.black54,
        borderRadius: BorderRadius.circular(12),
      ),
      child: Text(texto, textAlign: TextAlign.center),
    );
  }
}

class _Rodape extends StatelessWidget {
  const _Rodape(this.operador);

  final String operador;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 0, 16, 12),
      child: Text(
        operador.isEmpty ? '' : 'Operador: $operador',
        style: Theme.of(context).textTheme.bodySmall,
        textAlign: TextAlign.center,
      ),
    );
  }
}
