import 'package:connectivity_plus/connectivity_plus.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

/// Verdadeiro quando há alguma interface de rede ativa. Não garante acesso ao
/// serviço, mas basta para a faixa global de "Sem conexão" (CE-01).
final conectividadeProvider = StreamProvider<bool>((ref) async* {
  final conectividade = Connectivity();
  yield _conectado(await conectividade.checkConnectivity());
  yield* conectividade.onConnectivityChanged.map(_conectado);
});

bool _conectado(List<ConnectivityResult> resultados) =>
    resultados.any((r) => r != ConnectivityResult.none);

/// Faixa persistente no topo do aplicativo enquanto não há rede
/// (docs/arquitetura.md, seção 12, estado global).
class FaixaSemConexao extends ConsumerWidget {
  const FaixaSemConexao({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final conectado = ref.watch(conectividadeProvider).value ?? true;
    if (conectado) return const SizedBox.shrink();
    final esquema = Theme.of(context).colorScheme;
    return Material(
      color: esquema.errorContainer,
      child: SafeArea(
        bottom: false,
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
          child: Row(
            children: [
              Icon(Icons.wifi_off, size: 18, color: esquema.onErrorContainer),
              const SizedBox(width: 8),
              Expanded(
                child: Text(
                  'Sem conexão. A validação precisa de rede; a leitura do QR continua disponível.',
                  style: TextStyle(color: esquema.onErrorContainer),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
