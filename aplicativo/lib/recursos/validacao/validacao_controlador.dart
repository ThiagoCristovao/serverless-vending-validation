import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../compartilhado/erros.dart';
import '../../compartilhado/uuid.dart';
import '../autenticacao/sessao.dart';
import '../leitura/payload_qr.dart';
import 'cliente_http.dart';
import 'servico_validacao.dart';
import 'modelos.dart';

/// Estados da validação (docs/arquitetura.md, seção 12, tela Resultado).
sealed class EstadoValidacao {
  const EstadoValidacao();
}

class SemValidacao extends EstadoValidacao {
  const SemValidacao();
}

/// Requisição de abertura (ou retomada) em andamento.
class Enviando extends EstadoValidacao {
  const Enviando({this.payload});
  final PayloadQr? payload;
}

/// Validação aberta: contrassenha em exibição, aguardando leituras e conclusão.
class ValidacaoAberta extends EstadoValidacao {
  const ValidacaoAberta(
    this.validacao, {
    this.concluindo = false,
    this.erroConclusao,
  });
  final Validacao validacao;
  final bool concluindo;
  final ErroDominio? erroConclusao;

  ValidacaoAberta copiar({
    bool? concluindo,
    ErroDominio? erroConclusao,
    bool limparErro = false,
  }) => ValidacaoAberta(
    validacao,
    concluindo: concluindo ?? this.concluindo,
    erroConclusao: limparErro ? null : (erroConclusao ?? this.erroConclusao),
  );
}

class ValidacaoConcluida extends EstadoValidacao {
  const ValidacaoConcluida(this.validacao, this.conclusao);
  final Validacao validacao;
  final Conclusao conclusao;
}

/// A abertura falhou (ou a validação expirou antes da conclusão).
class FalhaValidacao extends EstadoValidacao {
  const FalhaValidacao(this.erro, {this.payload});
  final ErroDominio erro;
  final PayloadQr? payload;

  /// Erros que valem a pena repetir com o mesmo pedido.
  bool get podeTentarNovamente =>
      erro.codigo == CodigoErro.semConexao ||
      erro.codigo == CodigoErro.dependenciaIndisponivel;
}

/// Orquestra o fluxo 4–8 dos requisitos no aplicativo: uma chave de
/// idempotência por leitura, abertura, conclusão com consulta antes de
/// reenviar (CE-03) e tratamento dos códigos do serviço.
class ValidacaoControlador extends Notifier<EstadoValidacao> {
  PayloadQr? _payload;
  String? _chaveIdempotencia;

  @override
  EstadoValidacao build() => const SemValidacao();

  ServicoValidacao get _servico => ref.read(servicoValidacaoProvider);

  /// Abre a validação para o adesivo lido. Uma nova leitura gera nova chave;
  /// repetir o mesmo adesivo após falha reutiliza a chave (reenvio idempotente).
  Future<void> iniciar(PayloadQr payload) async {
    if (_payload == null || _payload!.sig != payload.sig) {
      _payload = payload;
      _chaveIdempotencia = gerarUuidV4();
    }
    state = Enviando(payload: payload);
    try {
      final validacao = await _servico.iniciar(
        payloadQr: payload.toString(),
        chaveIdempotencia: _chaveIdempotencia!,
      );
      state = validacao.aberta
          ? ValidacaoAberta(validacao)
          : _estadoDeConsulta(validacao);
    } on ErroDominio catch (e) {
      await _tratarSessao(e);
      state = FalhaValidacao(e, payload: payload);
    }
  }

  Future<void> tentarNovamente() async {
    final payload = _payload;
    if (payload == null) return;
    await iniciar(payload);
  }

  /// Retoma uma validação já aberta (CE-11, mesmo operador) ou recupera o
  /// estado após uma resposta perdida (CE-02/CE-03).
  Future<void> retomar(String idValidacao) async {
    state = Enviando(payload: _payload);
    try {
      state = _estadoDeConsulta(await _servico.consultar(idValidacao));
    } on ErroDominio catch (e) {
      await _tratarSessao(e);
      state = FalhaValidacao(e, payload: _payload);
    }
  }

  /// Esperas entre reenvios da conclusão, depois que a consulta confirma que
  /// ela não foi aplicada (mesma política do cliente HTTP, ADR-0009).
  static const esperasDeReenvio = [
    Duration(seconds: 1),
    Duration(seconds: 2),
    Duration(seconds: 4),
  ];

  Future<void> concluir({
    required Leituras leituras,
    required bool codigoRotacionado,
    String? observacoes,
  }) async {
    final atual = state;
    if (atual is! ValidacaoAberta || atual.concluindo) return;
    final v = atual.validacao;
    state = atual.copiar(concluindo: true, limparErro: true);
    var reenvios = 0;
    while (true) {
      try {
        final conclusao = await _servico.concluir(
          v.idValidacao,
          leituras: leituras,
          codigoRotacionado: codigoRotacionado,
          observacoes: observacoes,
        );
        state = ValidacaoConcluida(v, conclusao);
        return;
      } on ErroDominio catch (e) {
        await _tratarSessao(e);
        if (_podeTerConcluido(e)) {
          // A resposta pode ter se perdido depois de o serviço concluir (CE-03),
          // ou um envio anterior já concluiu: consulta antes de reenviar.
          final consulta = await _consultarSemFalhar(v.idValidacao);
          if (consulta != null && consulta.status != StatusValidacao.aberta) {
            state = _estadoDeConsulta(consulta);
            return;
          }
          // A consulta confirmou a validação ainda aberta: a conclusão não foi
          // aplicada e reenviá-la é seguro (RF-23, cenário D4 da Sprint 9).
          if (consulta != null &&
              _repetivel(e) &&
              reenvios < esperasDeReenvio.length) {
            await ref.read(esperaDeReenvioProvider)(esperasDeReenvio[reenvios]);
            reenvios++;
            continue;
          }
        }
        if (e.codigo == CodigoErro.validacaoExpirada) {
          state = FalhaValidacao(e, payload: _payload);
          return;
        }
        state = ValidacaoAberta(v, erroConclusao: e);
        return;
      }
    }
  }

  /// Erros após os quais o serviço pode ter concluído sem o aplicativo saber.
  static bool _podeTerConcluido(ErroDominio e) =>
      _repetivel(e) || e.codigo == CodigoErro.validacaoJaConcluida;

  static bool _repetivel(ErroDominio e) =>
      e.codigo == CodigoErro.semConexao ||
      e.codigo == CodigoErro.dependenciaIndisponivel;

  Future<Validacao?> _consultarSemFalhar(String idValidacao) async {
    try {
      return await _servico.consultar(idValidacao);
    } on ErroDominio {
      return null; // mantém o erro original da conclusão
    }
  }

  /// Marca a validação como expirada quando o prazo vence com a tela aberta.
  void expirou() {
    final atual = state;
    if (atual is ValidacaoAberta && !atual.concluindo) {
      state = FalhaValidacao(
        ErroDominio(
          CodigoErro.validacaoExpirada,
          'o prazo de conclusão terminou',
          idValidacao: atual.validacao.idValidacao,
        ),
        payload: _payload,
      );
    }
  }

  void reiniciar() {
    _payload = null;
    _chaveIdempotencia = null;
    state = const SemValidacao();
  }

  EstadoValidacao _estadoDeConsulta(Validacao v) => switch (v.status) {
    StatusValidacao.aberta => ValidacaoAberta(v),
    StatusValidacao.concluida => ValidacaoConcluida(
      v,
      Conclusao.deValidacao(v),
    ),
    StatusValidacao.expirada => FalhaValidacao(
      ErroDominio(
        CodigoErro.validacaoExpirada,
        'a validação expirou',
        idValidacao: v.idValidacao,
      ),
      payload: _payload,
    ),
  };

  /// Sessão recusada pelo gateway mesmo após renovar o token: encerra a sessão
  /// e o roteador leva ao login (CE-04).
  Future<void> _tratarSessao(ErroDominio e) async {
    if (e.codigo == CodigoErro.sessaoExpirada) {
      await ref.read(autenticacaoProvider.notifier).sair();
    }
  }
}

final validacaoProvider =
    NotifierProvider<ValidacaoControlador, EstadoValidacao>(
      ValidacaoControlador.new,
    );

/// Espera entre reenvios da conclusão; substituível nos testes.
final esperaDeReenvioProvider = Provider<Esperar>((_) => Future<void>.delayed);
