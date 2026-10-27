import 'package:firebase_auth/firebase_auth.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:svv/compartilhado/conectividade.dart';
import 'package:svv/compartilhado/erros.dart';
import 'package:svv/recursos/autenticacao/sessao.dart';

/// Firebase Auth falso: só o login existe, e sempre falha como o SDK falha
/// quando o host do provedor não responde.
class _AuthIndisponivel implements FirebaseAuth {
  @override
  Future<UserCredential> signInWithEmailAndPassword({
    required String email,
    required String password,
  }) async => throw FirebaseAuthException(code: 'network-request-failed');

  @override
  dynamic noSuchMethod(Invocation invocation) => super.noSuchMethod(invocation);
}

void main() {
  group('traduzir', () {
    test('mapeia os códigos do Firebase para os do aplicativo', () {
      expect(
        AutenticacaoControlador.traduzir(
          FirebaseAuthException(code: 'wrong-password'),
        ).codigo,
        CodigoErro.credenciaisInvalidas,
      );
      expect(
        AutenticacaoControlador.traduzir(
          FirebaseAuthException(code: 'network-request-failed'),
        ).codigo,
        CodigoErro.semConexao,
      );
      expect(
        AutenticacaoControlador.traduzir(
          FirebaseAuthException(code: 'internal-error'),
        ).codigo,
        CodigoErro.autenticacaoIndisponivel,
      );
    });
  });

  group('qualificar (CE-05)', () {
    const semConexao = ErroDominio(CodigoErro.semConexao, 'rede');
    test(
      'falha de rede com o aparelho conectado vira autenticação indisponível',
      () {
        expect(
          AutenticacaoControlador.qualificar(
            semConexao,
            conectado: true,
          ).codigo,
          CodigoErro.autenticacaoIndisponivel,
        );
      },
    );
    test('falha de rede sem conexão continua sem conexão', () {
      expect(
        AutenticacaoControlador.qualificar(semConexao, conectado: false).codigo,
        CodigoErro.semConexao,
      );
    });
    test('outros erros passam inalterados', () {
      const credenciais = ErroDominio(CodigoErro.credenciaisInvalidas, 'x');
      expect(
        AutenticacaoControlador.qualificar(credenciais, conectado: true),
        same(credenciais),
      );
    });
  });

  group('entrar', () {
    Future<ProviderContainer> montar({required bool conectado}) async {
      final container = ProviderContainer(
        overrides: [
          firebaseAuthProvider.overrideWithValue(_AuthIndisponivel()),
          conectividadeProvider.overrideWith((ref) => Stream.value(conectado)),
        ],
      );
      addTearDown(container.dispose);
      // Mantém o provedor de conectividade vivo (sem ouvinte ele é descartado)
      // e garante que já emitiu antes do login.
      final assinatura = container.listen(conectividadeProvider, (_, _) {});
      addTearDown(assinatura.close);
      await container.read(conectividadeProvider.future);
      return container;
    }

    test('provedor fora do ar com rede: autenticação indisponível', () async {
      final container = await montar(conectado: true);
      await container
          .read(autenticacaoProvider.notifier)
          .entrar(email: 'operador@exemplo.invalid', senha: 'segredo');
      final estado = container.read(autenticacaoProvider);
      expect(estado.hasError, isTrue);
      expect(
        (estado.error! as ErroDominio).codigo,
        CodigoErro.autenticacaoIndisponivel,
      );
    });

    test('provedor fora do ar sem rede: sem conexão', () async {
      final container = await montar(conectado: false);
      await container
          .read(autenticacaoProvider.notifier)
          .entrar(email: 'operador@exemplo.invalid', senha: 'segredo');
      expect(
        (container.read(autenticacaoProvider).error! as ErroDominio).codigo,
        CodigoErro.semConexao,
      );
    });
  });
}
