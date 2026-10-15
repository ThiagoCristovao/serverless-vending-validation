import 'dart:convert';
import 'dart:typed_data';

import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:svv/compartilhado/erros.dart';
import 'package:svv/recursos/validacao/cliente_http.dart';
import 'package:svv/recursos/validacao/modelos.dart';

/// Transporte falso: registra as requisições e devolve respostas roteirizadas.
class TransporteFalso implements HttpClientAdapter {
  final List<RequestOptions> requisicoes = [];
  final List<ResponseBody Function(RequestOptions)> roteiro = [];

  @override
  Future<ResponseBody> fetch(
    RequestOptions options,
    Stream<Uint8List>? requestStream,
    Future<void>? cancelFuture,
  ) async {
    requisicoes.add(options);
    if (roteiro.isEmpty) {
      throw StateError(
        'sem resposta roteirizada para ${options.method} ${options.path}',
      );
    }
    return roteiro.removeAt(0)(options);
  }

  @override
  void close({bool force = false}) {}
}

ResponseBody json(
  int status,
  Object corpo, {
  String tipo = 'application/json',
}) => ResponseBody.fromString(
  jsonEncode(corpo),
  status,
  headers: {
    'content-type': [tipo],
  },
);

const payloadQr =
    '{"v":1,"maq":"VM-2047","mod":"CN168","loc":"BLA-T","exp":"2027-09-25","kid":"k1","sig":"MsxhS6H59KpgEjML9psz4vihfRHusUTyDuMlAjdTERGRzOMa9QWb-q_7eCGx3aXFx2ieCxO2Gf2DQ8alyDeuDg"}';

final aberta = {
  'idValidacao': '01J8ZK3V9Q7XW2N4M6P8R0T2Y4',
  'status': 'aberta',
  'contrassenha': '4729',
  'proximaContrassenha': '0318',
  'iniciadaEm': '2026-09-17T13:05:12Z',
  'expiraEm': '2026-09-17T13:20:12Z',
  'concluidaEm': null,
  'maquina': {
    'id': 'VM-2047',
    'modelo': 'CN168',
    'localizacao': {'id': 'BLA-T', 'nome': 'Bloco A - Térreo'},
    'ultimaValidacaoEm': null,
  },
};

void main() {
  late TransporteFalso transporte;
  late List<bool> renovacoes;
  late ClienteHttpValidacao cliente;
  late List<Duration> esperas;

  setUp(() {
    transporte = TransporteFalso();
    renovacoes = [];
    esperas = [];
    final dio = Dio(
      BaseOptions(
        baseUrl: 'https://gateway.teste/v1',
        responseType: ResponseType.plain,
        validateStatus: (_) => true,
      ),
    )..httpClientAdapter = transporte;
    cliente = ClienteHttpValidacao(
      obterToken: ({bool renovar = false}) async {
        renovacoes.add(renovar);
        return renovar ? 'token-novo' : 'token-antigo';
      },
      dio: dio,
      esperar: (d) async => esperas.add(d),
    );
  });

  test(
    'iniciar envia token, chave de idempotência e o payload, e lê o 201',
    () async {
      transporte.roteiro.add((_) => json(201, aberta));
      final v = await cliente.iniciar(
        payloadQr: payloadQr,
        chaveIdempotencia: 'chave-1',
      );

      expect(v.contrassenha, '4729');
      final req = transporte.requisicoes.single;
      expect(req.method, 'POST');
      expect(req.path, '/validacoes');
      expect(req.headers['Authorization'], 'Bearer token-antigo');
      expect(req.headers['Idempotency-Key'], 'chave-1');
      final corpo = jsonDecode(req.data as String) as Map;
      expect((corpo['qr'] as Map)['maq'], 'VM-2047');
      expect((corpo['dispositivo'] as Map)['plataforma'], 'android');
    },
  );

  test(
    '401 renova o token uma vez e repete; se persistir, sessão expirada',
    () async {
      transporte.roteiro
        ..add((_) => json(401, {'code': 401, 'message': 'Jwt is expired'}))
        ..add((_) => json(201, aberta));
      await cliente.iniciar(payloadQr: payloadQr, chaveIdempotencia: 'chave-1');
      expect(renovacoes, [false, true]);
      expect(
        transporte.requisicoes.last.headers['Authorization'],
        'Bearer token-novo',
      );

      transporte.roteiro
        ..add((_) => json(401, {'code': 401, 'message': 'Jwt is expired'}))
        ..add((_) => json(401, {'code': 401, 'message': 'Jwt is expired'}));
      await expectLater(
        cliente.consultar('01J8ZK3V9Q7XW2N4M6P8R0T2Y4'),
        throwsA(
          isA<ErroDominio>().having(
            (e) => e.codigo,
            'codigo',
            CodigoErro.sessaoExpirada,
          ),
        ),
      );
    },
  );

  test('Problem Details do serviço vira ErroDominio com idValidacao', () async {
    transporte.roteiro.add(
      (_) => json(409, {
        'type': 'urn:svv:erro:validacao_em_andamento',
        'title': 'Validação em andamento',
        'status': 409,
        'codigo': 'validacao_em_andamento',
        'idValidacao': '01J8ZK3V9Q7XW2N4M6P8R0T2Y4',
      }, tipo: 'application/problem+json'),
    );
    await expectLater(
      cliente.iniciar(payloadQr: payloadQr, chaveIdempotencia: 'chave-2'),
      throwsA(
        isA<ErroDominio>()
            .having((e) => e.codigo, 'codigo', CodigoErro.validacaoEmAndamento)
            .having(
              (e) => e.idValidacao,
              'idValidacao',
              '01J8ZK3V9Q7XW2N4M6P8R0T2Y4',
            ),
      ),
    );
    expect(esperas, isEmpty, reason: '4xx não deve gerar retentativa');
  });

  test('falha de rede e 503 são repetidos com esperas 1s, 2s, 4s e depois desistem', () async {
    transporte.roteiro
      ..add(
        (o) => throw DioException.connectionError(
          requestOptions: o,
          reason: 'sem rede',
        ),
      )
      ..add(
        (_) => json(503, {
          'codigo': 'dependencia_indisponivel',
          'status': 503,
          'type': 'x',
          'title': 'x',
        }),
      )
      ..add((_) => json(201, aberta));
    final v = await cliente.iniciar(
      payloadQr: payloadQr,
      chaveIdempotencia: 'chave-3',
    );
    expect(v.aberta, isTrue);
    expect(esperas, [const Duration(seconds: 1), const Duration(seconds: 2)]);
    expect(transporte.requisicoes.length, 3);
    expect(
      transporte.requisicoes.map((r) => r.headers['Idempotency-Key']).toSet(),
      {'chave-3'},
    );

    transporte.roteiro.addAll(
      List.generate(
        4,
        (_) =>
            (o) => throw DioException.connectionError(
              requestOptions: o,
              reason: 'sem rede',
            ),
      ),
    );
    await expectLater(
      cliente.consultar('01J8ZK3V9Q7XW2N4M6P8R0T2Y4'),
      throwsA(
        isA<ErroDominio>().having(
          (e) => e.codigo,
          'codigo',
          CodigoErro.semConexao,
        ),
      ),
    );
  });

  test('concluir não repete sozinho uma falha de rede (o controlador consulta antes)', () async {
    transporte.roteiro.add(
      (o) => throw DioException.connectionError(
        requestOptions: o,
        reason: 'sem rede',
      ),
    );
    await expectLater(
      cliente.concluir(
        '01J8ZK3V9Q7XW2N4M6P8R0T2Y4',
        leituras: const Leituras(medidor: 1, unidadesVendidas: 1),
        codigoRotacionado: true,
      ),
      throwsA(
        isA<ErroDominio>().having(
          (e) => e.codigo,
          'codigo',
          CodigoErro.semConexao,
        ),
      ),
    );
    expect(transporte.requisicoes.length, 1);
  });
}
