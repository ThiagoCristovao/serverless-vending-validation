import 'dart:async';

import 'package:firebase_auth/firebase_auth.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../compartilhado/erros.dart';

/// Acesso ao Firebase Authentication; substituível nos testes.
final firebaseAuthProvider = Provider<FirebaseAuth>(
  (ref) => FirebaseAuth.instance,
);

/// Operador autenticado (ou null). Fonte da verdade para o roteador.
final sessaoProvider = StreamProvider<User?>(
  (ref) => ref.watch(firebaseAuthProvider).authStateChanges(),
);

/// Entrada e saída do operador (RF-01). O estado carrega o erro da última
/// tentativa, já traduzido para um código do aplicativo.
class AutenticacaoControlador extends AsyncNotifier<void> {
  // Síncrono: o estado inicial já é AsyncData, sem passar por carregando.
  @override
  FutureOr<void> build() {}

  Future<void> entrar({required String email, required String senha}) async {
    state = const AsyncLoading();
    state = await AsyncValue.guard(() async {
      try {
        await ref
            .read(firebaseAuthProvider)
            .signInWithEmailAndPassword(email: email.trim(), password: senha);
      } on FirebaseAuthException catch (e) {
        throw traduzir(e);
      }
    });
  }

  Future<void> sair() async {
    await ref.read(firebaseAuthProvider).signOut();
    state = const AsyncData(null);
  }

  /// Converte os códigos do Firebase nos códigos do aplicativo
  /// (docs/arquitetura.md, CE-04 e CE-05).
  static ErroDominio traduzir(FirebaseAuthException e) => switch (e.code) {
    'invalid-credential' ||
    'wrong-password' ||
    'user-not-found' ||
    'invalid-email' ||
    'INVALID_LOGIN_CREDENTIALS' => ErroDominio(
      CodigoErro.credenciaisInvalidas,
      e.message ?? e.code,
    ),
    'user-disabled' => ErroDominio(
      CodigoErro.operadorNaoAutorizado,
      e.message ?? e.code,
    ),
    'network-request-failed' => ErroDominio(
      CodigoErro.semConexao,
      e.message ?? e.code,
    ),
    _ => ErroDominio(
      CodigoErro.autenticacaoIndisponivel,
      '${e.code}: ${e.message}',
    ),
  };
}

final autenticacaoProvider =
    AsyncNotifierProvider<AutenticacaoControlador, void>(
      AutenticacaoControlador.new,
    );
