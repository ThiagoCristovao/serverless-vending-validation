import 'dart:convert';
import 'dart:io';

import 'package:flutter_test/flutter_test.dart';
import 'package:svv/compartilhado/erros.dart';
import 'package:svv/recursos/validacao/modelos.dart';

// Os exemplos do contrato (api/exemplos) são a referência: se o serviço e o
// aplicativo os leem igual, falam a mesma língua.
Map<String, Object?> exemplo(String nome) =>
    (jsonDecode(File('../api/exemplos/$nome').readAsStringSync()) as Map)
        .cast<String, Object?>();

void main() {
  test('resposta de abertura (201)', () {
    final v = Validacao.fromJson(exemplo('validacao-resposta.json'));
    expect(v.idValidacao, '01J8ZK3V9Q7XW2N4M6P8R0T2Y4');
    expect(v.status, StatusValidacao.aberta);
    expect(v.aberta, isTrue);
    expect(v.contrassenha, '4729');
    expect(v.proximaContrassenha, '0318');
    expect(v.maquina.id, 'VM-2047');
    expect(v.maquina.localizacao.nome, 'Bloco A - Térreo');
    expect(v.concluidaEm, isNull);
    expect(
      v.restante(DateTime.utc(2026, 9, 17, 13, 10)),
      const Duration(minutes: 10, seconds: 12),
    );
    expect(v.restante(DateTime.utc(2026, 9, 17, 13, 30)), Duration.zero);
  });

  test('consulta de validação concluída (200)', () {
    final v = Validacao.fromJson(exemplo('consulta-resposta.json'));
    expect(v.status, StatusValidacao.concluida);
    expect(v.contrassenha, isNull);
    expect(v.leituras?.medidor, 14832);
    expect(v.codigoRotacionado, isTrue);
    expect(v.avisos, isEmpty);
    final c = Conclusao.deValidacao(v);
    expect(c.concluidaEm, v.concluidaEm);
    expect(c.codigoRotacionado, isTrue);
  });

  test('resposta de conclusão (200)', () {
    final c = Conclusao.fromJson(exemplo('conclusao-resposta.json'));
    expect(c.status, StatusValidacao.concluida);
    expect(c.codigoRotacionado, isTrue);
    expect(c.avisos, isEmpty);
  });

  test('erro em Problem Details vira ErroDominio com código e validação', () {
    final e = Problema.fromJson(exemplo('erro-409.json')).paraErro();
    expect(e.codigo, CodigoErro.validacaoEmAndamento);
    expect(e.idValidacao, '01J8ZK3V9Q7XW2N4M6P8R0T2Y4');
    expect(e.mensagem, contains('VM-2047'));
  });

  test('código desconhecido cai em erro_interno e aviso tem mensagem', () {
    expect(CodigoErro.deValor('algo_novo'), CodigoErro.erroInterno);
    expect(mensagemDoAviso('medidor_menor_que_anterior'), contains('medidor'));
  });
}
