import 'dart:convert';

import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../compartilhado/configuracao.dart';
import '../../compartilhado/erros.dart';
import '../autenticacao/sessao.dart';
import 'modelos.dart';
import 'servico_validacao.dart';

/// Obtém o ID token do operador; `renovar` força a renovação (CE-04).
typedef ObterToken = Future<String?> Function({bool renovar});

/// Espera entre retentativas; injetável nos testes.
typedef Esperar = Future<void> Function(Duration duracao);

/// Cliente HTTP do contrato: injeta o token do Firebase, aplica a política de
/// retentativa (RF-23, ADR-0009) e converte Problem Details em ErroDominio.
class ClienteHttpValidacao implements ServicoValidacao {
  ClienteHttpValidacao({
    required this._obterToken,
    Dio? dio,
    String? baseUrl,
    Esperar? esperar,
    this.esperasDeRetentativa = const [
      Duration(seconds: 1),
      Duration(seconds: 2),
      Duration(seconds: 4),
    ],
  }) : _esperar = esperar ?? Future<void>.delayed,
       _dio =
           dio ??
           Dio(
             BaseOptions(
               baseUrl: baseUrl ?? Configuracao.apiBase,
               connectTimeout: const Duration(seconds: 10),
               sendTimeout: const Duration(seconds: 10),
               receiveTimeout: const Duration(seconds: 20),
               responseType: ResponseType.plain,
               validateStatus: (_) => true,
               headers: const {
                 'Content-Type': 'application/json',
                 'Accept': 'application/json, application/problem+json',
               },
             ),
           );

  final ObterToken _obterToken;
  final Esperar _esperar;
  final Dio _dio;

  /// Esperas entre tentativas para erros de rede e 503; nunca para outros 4xx.
  final List<Duration> esperasDeRetentativa;

  @override
  Future<Validacao> iniciar({
    required String payloadQr,
    required String chaveIdempotencia,
  }) {
    final corpo = {
      'qr': jsonDecode(payloadQr),
      'dispositivo': {
        'plataforma': Configuracao.plataforma,
        'versaoApp': Configuracao.versaoApp,
      },
    };
    // Idempotente por chave: seguro repetir mesmo que a resposta anterior tenha se perdido.
    return _comRetentativa(
      () => _chamar(
        'POST',
        '/validacoes',
        corpo: corpo,
        extras: {'Idempotency-Key': chaveIdempotencia},
        esperado: 201,
        parse: Validacao.fromJson,
      ),
    );
  }

  @override
  Future<Validacao> consultar(String idValidacao) => _comRetentativa(
    () => _chamar(
      'GET',
      '/validacoes/$idValidacao',
      esperado: 200,
      parse: Validacao.fromJson,
    ),
  );

  @override
  Future<Conclusao> concluir(
    String idValidacao, {
    required Leituras leituras,
    required bool codigoRotacionado,
    String? observacoes,
  }) {
    final corpo = {
      'leituras': leituras.toJson(),
      'codigoRotacionado': codigoRotacionado,
      if (observacoes != null && observacoes.trim().isNotEmpty)
        'observacoes': observacoes.trim(),
    };
    // Sem retentativa automática aqui: o controlador consulta o estado antes
    // de reenviar uma conclusão cuja resposta se perdeu (CE-03).
    return _chamar(
      'POST',
      '/validacoes/$idValidacao/conclusao',
      corpo: corpo,
      esperado: 200,
      parse: Conclusao.fromJson,
    );
  }

  Future<T> _comRetentativa<T>(Future<T> Function() operacao) async {
    var tentativa = 0;
    while (true) {
      try {
        return await operacao();
      } on ErroDominio catch (e) {
        final repetivel =
            e.codigo == CodigoErro.semConexao ||
            e.codigo == CodigoErro.dependenciaIndisponivel;
        if (!repetivel || tentativa >= esperasDeRetentativa.length) rethrow;
        await _esperar(esperasDeRetentativa[tentativa]);
        tentativa++;
      }
    }
  }

  Future<T> _chamar<T>(
    String metodo,
    String caminho, {
    Object? corpo,
    Map<String, String> extras = const {},
    required int esperado,
    required T Function(Map<String, Object?>) parse,
  }) async {
    var resposta = await _requisitar(
      metodo,
      caminho,
      corpo,
      extras,
      renovarToken: false,
    );
    if (resposta.statusCode == 401) {
      // Token possivelmente expirado: renova uma vez e repete (CE-04).
      resposta = await _requisitar(
        metodo,
        caminho,
        corpo,
        extras,
        renovarToken: true,
      );
    }
    return _interpretar(resposta, esperado, parse);
  }

  Future<Response<String>> _requisitar(
    String metodo,
    String caminho,
    Object? corpo,
    Map<String, String> extras, {
    required bool renovarToken,
  }) async {
    final token = await _obterToken(renovar: renovarToken);
    if (token == null || token.isEmpty) {
      throw const ErroDominio(
        CodigoErro.sessaoExpirada,
        'operador sem sessão ativa',
      );
    }
    try {
      return await _dio.request<String>(
        caminho,
        data: corpo == null ? null : jsonEncode(corpo),
        options: Options(
          method: metodo,
          headers: {'Authorization': 'Bearer $token', ...extras},
        ),
      );
    } on DioException catch (e) {
      throw _erroDeTransporte(e);
    }
  }

  T _interpretar<T>(
    Response<String> resposta,
    int esperado,
    T Function(Map<String, Object?>) parse,
  ) {
    final status = resposta.statusCode ?? 0;
    final corpo = _corpoJson(resposta.data);
    if (status == esperado) {
      if (corpo == null) {
        throw const ErroDominio(
          CodigoErro.erroInterno,
          'resposta sem corpo JSON',
        );
      }
      return parse(corpo);
    }
    if (status == 401) {
      throw const ErroDominio(
        CodigoErro.sessaoExpirada,
        'token recusado pelo gateway',
      );
    }
    if (corpo != null && corpo['codigo'] != null) {
      throw Problema.fromJson(corpo).paraErro();
    }
    if (status >= 500 || status == 429) {
      throw ErroDominio(CodigoErro.dependenciaIndisponivel, 'HTTP $status');
    }
    throw ErroDominio(
      CodigoErro.erroInterno,
      'HTTP $status sem Problem Details',
    );
  }

  static Map<String, Object?>? _corpoJson(String? texto) {
    if (texto == null || texto.trim().isEmpty) return null;
    try {
      final decodificado = jsonDecode(texto);
      return decodificado is Map ? decodificado.cast<String, Object?>() : null;
    } on FormatException {
      return null;
    }
  }

  static ErroDominio _erroDeTransporte(DioException e) => switch (e.type) {
    DioExceptionType.connectionTimeout ||
    DioExceptionType.sendTimeout ||
    DioExceptionType.receiveTimeout ||
    DioExceptionType.connectionError ||
    DioExceptionType.unknown => ErroDominio(
      CodigoErro.semConexao,
      e.message ?? e.type.name,
    ),
    DioExceptionType.badCertificate => ErroDominio(
      CodigoErro.erroInterno,
      'certificado inválido: ${e.message}',
    ),
    // badResponse, cancel e tipos novos do dio (ex.: transformTimeout).
    _ => ErroDominio(CodigoErro.erroInterno, e.message ?? e.type.name),
  };
}

/// Cliente real, com o token do operador autenticado no Firebase.
final servicoValidacaoProvider = Provider<ServicoValidacao>((ref) {
  final auth = ref.watch(firebaseAuthProvider);
  return ClienteHttpValidacao(
    obterToken: ({bool renovar = false}) async {
      final usuario = auth.currentUser;
      if (usuario == null) return null;
      return usuario.getIdToken(renovar);
    },
  );
});
