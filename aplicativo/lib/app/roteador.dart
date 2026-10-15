import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../recursos/autenticacao/sessao.dart';
import '../recursos/autenticacao/tela_entrar.dart';
import '../recursos/leitura/tela_leitura.dart';

/// Caminhos das telas.
abstract final class Rotas {
  static const entrar = '/entrar';
  static const leitura = '/leitura';
}

/// Converte um Stream em Listenable para o GoRouter reavaliar os redirects
/// sempre que a sessão mudar.
class _NotificadorDeStream extends ChangeNotifier {
  _NotificadorDeStream(Stream<Object?> stream) {
    _assinatura = stream.asBroadcastStream().listen((_) => notifyListeners());
  }

  late final StreamSubscription<Object?> _assinatura;

  @override
  void dispose() {
    _assinatura.cancel();
    super.dispose();
  }
}

final roteadorProvider = Provider<GoRouter>((ref) {
  final notificador = _NotificadorDeStream(
    ref.watch(firebaseAuthProvider).authStateChanges(),
  );
  ref.onDispose(notificador.dispose);

  return GoRouter(
    initialLocation: Rotas.leitura,
    refreshListenable: notificador,
    redirect: (context, estado) {
      final sessao = ref.read(sessaoProvider);
      if (sessao.isLoading) return null;
      final autenticado = sessao.value != null;
      final indoParaEntrar = estado.matchedLocation == Rotas.entrar;
      if (!autenticado && !indoParaEntrar) return Rotas.entrar;
      if (autenticado && indoParaEntrar) return Rotas.leitura;
      return null;
    },
    routes: [
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
