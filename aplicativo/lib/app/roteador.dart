import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../recursos/autenticacao/sessao.dart';
import '../recursos/autenticacao/tela_entrar.dart';
import '../recursos/leitura/tela_leitura.dart';
import 'tela_inicial.dart';

/// Caminhos das telas.
abstract final class Rotas {
  static const inicial = '/';
  static const entrar = '/entrar';
  static const leitura = '/leitura';
}

/// Avisa o GoRouter para reavaliar os redirecionamentos.
class _Notificador extends ChangeNotifier {
  void notificar() => notifyListeners();
}

final roteadorProvider = Provider<GoRouter>((ref) {
  final notificador = _Notificador();
  ref.onDispose(notificador.dispose);

  // Reage ao provedor de sessão, e não ao Firebase diretamente: assim o
  // redirect só roda depois que o estado da sessão já foi atualizado, e o
  // provedor fica vivo enquanto o roteador existir.
  ref.listen(sessaoProvider, (_, _) => notificador.notificar());

  return GoRouter(
    initialLocation: Rotas.inicial,
    refreshListenable: notificador,
    redirect: (context, estado) {
      final sessao = ref.read(sessaoProvider);
      final local = estado.matchedLocation;
      if (sessao.isLoading) {
        return local == Rotas.inicial ? null : Rotas.inicial;
      }
      final autenticado = sessao.value != null;
      if (!autenticado) {
        return local == Rotas.entrar ? null : Rotas.entrar;
      }
      if (local == Rotas.entrar || local == Rotas.inicial) {
        return Rotas.leitura;
      }
      return null;
    },
    routes: [
      GoRoute(
        path: Rotas.inicial,
        builder: (context, estado) => const TelaInicial(),
      ),
      GoRoute(
        path: Rotas.entrar,
        builder: (context, estado) => const TelaEntrar(),
      ),
      GoRoute(
        path: Rotas.leitura,
        builder: (context, estado) => const TelaLeitura(),
      ),
    ],
  );
});
