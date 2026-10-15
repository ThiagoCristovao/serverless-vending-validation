import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:svv/compartilhado/erros.dart';
import 'package:svv/recursos/leitura/payload_qr.dart';
import 'package:svv/recursos/validacao/cliente_http.dart';
import 'package:svv/recursos/validacao/modelos.dart';
import 'package:svv/recursos/validacao/servico_validacao.dart';
import 'package:svv/recursos/validacao/tela_resultado.dart';
import 'package:svv/recursos/validacao/validacao_controlador.dart';

/// Serviço falso com comportamento roteirizado por chamada.
class ServicoFalso implements ServicoValidacao {
  final List<Future<Validacao> Function()> aoIniciar = [];
  final List<Future<Validacao> Function()> aoConsultar = [];
  final List<Future<Conclusao> Function()> aoConcluir = [];
  final List<String> chamadas = [];

  @override
  Future<Validacao> iniciar({
    required String payloadQr,
    required String chaveIdempotencia,
  }) {
    chamadas.add('iniciar:$chaveIdempotencia');
    return aoIniciar.removeAt(0)();
  }

  @override
  Future<Validacao> consultar(String idValidacao) {
    chamadas.add('consultar:$idValidacao');
    return aoConsultar.removeAt(0)();
  }

  @override
  Future<Conclusao> concluir(
    String idValidacao, {
    required Leituras leituras,
    required bool codigoRotacionado,
    String? observacoes,
  }) {
    chamadas.add(
      'concluir:$idValidacao:${leituras.medidor}:$codigoRotacionado',
    );
    return aoConcluir.removeAt(0)();
  }
}

final agora = DateTime.now().toUtc();

Validacao abertaExemplo() => Validacao(
  idValidacao: '01J8ZK3V9Q7XW2N4M6P8R0T2Y4',
  status: StatusValidacao.aberta,
  contrassenha: '4729',
  proximaContrassenha: '0318',
  iniciadaEm: agora,
  expiraEm: agora.add(const Duration(minutes: 15)),
  maquina: const Maquina(
    id: 'VM-2047',
    modelo: 'CN168',
    localizacao: Localizacao(id: 'BLA-T', nome: 'Bloco A - Térreo'),
  ),
);

Validacao concluidaExemplo({List<String> avisos = const []}) => Validacao(
  idValidacao: '01J8ZK3V9Q7XW2N4M6P8R0T2Y4',
  status: StatusValidacao.concluida,
  iniciadaEm: agora,
  expiraEm: agora.add(const Duration(minutes: 15)),
  concluidaEm: agora.add(const Duration(minutes: 5)),
  maquina: const Maquina(
    id: 'VM-2047',
    modelo: 'CN168',
    localizacao: Localizacao(id: 'BLA-T', nome: 'Bloco A - Térreo'),
  ),
  leituras: const Leituras(medidor: 14832, unidadesVendidas: 137),
  codigoRotacionado: true,
  avisos: avisos,
);

const payload = PayloadQr(
  v: 1,
  maq: 'VM-2047',
  mod: 'CN168',
  loc: 'BLA-T',
  exp: '2027-09-25',
  kid: 'k1',
  sig: 'x',
);

Future<ProviderContainer> montar(
  WidgetTester tester,
  ServicoFalso servico,
) async {
  final container = ProviderContainer(
    overrides: [servicoValidacaoProvider.overrideWithValue(servico)],
  );
  addTearDown(container.dispose);
  await tester.pumpWidget(
    UncontrolledProviderScope(
      container: container,
      child: const MaterialApp(home: TelaResultado()),
    ),
  );
  return container;
}

Future<void> preencherEConcluir(
  WidgetTester tester, {
  bool rotacionar = true,
}) async {
  await tester.enterText(
    find.widgetWithText(TextFormField, 'Medidor (vendas totais)'),
    '14832',
  );
  await tester.enterText(
    find.widgetWithText(
      TextFormField,
      'Unidades vendidas desde a última coleta',
    ),
    '137',
  );
  if (rotacionar) {
    await tester.ensureVisible(find.byType(CheckboxListTile));
    await tester.tap(find.byType(CheckboxListTile));
    await tester.pump();
  }
  await tester.ensureVisible(
    find.widgetWithText(FilledButton, 'Concluir visita'),
  );
  await tester.tap(find.widgetWithText(FilledButton, 'Concluir visita'));
  await tester.pumpAndSettle();
}

void main() {
  testWidgets(
    'fluxo principal: abre, mostra a contrassenha e conclui com rotação',
    (tester) async {
      final servico = ServicoFalso()
        ..aoIniciar.add(() async => abertaExemplo())
        ..aoConcluir.add(
          () async => Conclusao(
            idValidacao: '01J8ZK3V9Q7XW2N4M6P8R0T2Y4',
            status: StatusValidacao.concluida,
            concluidaEm: agora,
            codigoRotacionado: true,
          ),
        );
      final container = await montar(tester, servico);

      await container.read(validacaoProvider.notifier).iniciar(payload);
      await tester.pumpAndSettle();
      expect(find.text('4729'), findsOneWidget);
      expect(
        find.textContaining('Próximo código a programar: 0318'),
        findsOneWidget,
      );
      expect(find.text('VM-2047'), findsOneWidget);

      await preencherEConcluir(tester);
      expect(find.text('Máquina VM-2047 validada'), findsOneWidget);
      expect(find.text('Código supervisor rotacionado.'), findsOneWidget);
      expect(
        servico.chamadas.last,
        'concluir:01J8ZK3V9Q7XW2N4M6P8R0T2Y4:14832:true',
      );
    },
  );

  testWidgets('leituras inválidas não são enviadas', (tester) async {
    final servico = ServicoFalso()..aoIniciar.add(() async => abertaExemplo());
    final container = await montar(tester, servico);
    await container.read(validacaoProvider.notifier).iniciar(payload);
    await tester.pumpAndSettle();

    await tester.ensureVisible(
      find.widgetWithText(FilledButton, 'Concluir visita'),
    );
    await tester.tap(find.widgetWithText(FilledButton, 'Concluir visita'));
    await tester.pumpAndSettle();
    expect(
      find.text('Informe um número inteiro não negativo'),
      findsNWidgets(2),
    );
    expect(servico.chamadas.where((c) => c.startsWith('concluir')), isEmpty);
  });

  testWidgets(
    '409 em andamento oferece retomar; a retomada mostra a validação aberta',
    (tester) async {
      final servico = ServicoFalso()
        ..aoIniciar.add(
          () async => throw const ErroDominio(
            CodigoErro.validacaoEmAndamento,
            'em andamento',
            idValidacao: '01J8ZK3V9Q7XW2N4M6P8R0T2Y4',
          ),
        )
        ..aoConsultar.add(() async => abertaExemplo());
      final container = await montar(tester, servico);
      await container.read(validacaoProvider.notifier).iniciar(payload);
      await tester.pumpAndSettle();

      expect(
        find.text(mensagemParaOperador(CodigoErro.validacaoEmAndamento)),
        findsOneWidget,
      );
      await tester.tap(find.text('Retomar validação em andamento'));
      await tester.pumpAndSettle();
      expect(find.text('4729'), findsOneWidget);
      expect(servico.chamadas, [
        'iniciar:${servico.chamadas.first.split(':')[1]}',
        'consultar:01J8ZK3V9Q7XW2N4M6P8R0T2Y4',
      ]);
    },
  );

  testWidgets(
    'sem conexão na abertura: tentar novamente reaproveita a mesma chave de idempotência',
    (tester) async {
      final servico = ServicoFalso()
        ..aoIniciar.add(
          () async =>
              throw const ErroDominio(CodigoErro.semConexao, 'sem rede'),
        )
        ..aoIniciar.add(() async => abertaExemplo());
      final container = await montar(tester, servico);
      await container.read(validacaoProvider.notifier).iniciar(payload);
      await tester.pumpAndSettle();

      expect(
        find.text(mensagemParaOperador(CodigoErro.semConexao)),
        findsOneWidget,
      );
      await tester.tap(find.text('Tentar novamente'));
      await tester.pumpAndSettle();
      expect(find.text('4729'), findsOneWidget);
      final chaves = servico.chamadas
          .where((c) => c.startsWith('iniciar'))
          .map((c) => c.split(':')[1])
          .toSet();
      expect(
        chaves.length,
        1,
        reason: 'o reenvio deve usar a mesma Idempotency-Key',
      );
    },
  );

  testWidgets(
    'resposta da conclusão perdida: consulta e reconhece a conclusão (CE-03)',
    (tester) async {
      final servico = ServicoFalso()
        ..aoIniciar.add(() async => abertaExemplo())
        ..aoConcluir.add(
          () async =>
              throw const ErroDominio(CodigoErro.semConexao, 'sem rede'),
        )
        ..aoConsultar.add(
          () async => concluidaExemplo(avisos: ['medidor_menor_que_anterior']),
        );
      final container = await montar(tester, servico);
      await container.read(validacaoProvider.notifier).iniciar(payload);
      await tester.pumpAndSettle();

      await preencherEConcluir(tester);
      expect(find.text('Máquina VM-2047 validada'), findsOneWidget);
      expect(find.textContaining('medidor informado é menor'), findsOneWidget);
      expect(servico.chamadas.last, 'consultar:01J8ZK3V9Q7XW2N4M6P8R0T2Y4');
    },
  );

  testWidgets(
    'erro de negócio na conclusão mantém a validação aberta com a mensagem',
    (tester) async {
      final servico = ServicoFalso()
        ..aoIniciar.add(() async => abertaExemplo())
        ..aoConcluir.add(
          () async => throw const ErroDominio(
            CodigoErro.validacaoJaConcluida,
            'já concluída',
          ),
        );
      final container = await montar(tester, servico);
      await container.read(validacaoProvider.notifier).iniciar(payload);
      await tester.pumpAndSettle();

      await preencherEConcluir(tester);
      expect(
        find.text(mensagemParaOperador(CodigoErro.validacaoJaConcluida)),
        findsOneWidget,
      );
      expect(
        find.text('4729'),
        findsOneWidget,
        reason: 'a tela continua na validação aberta',
      );
    },
  );
}
