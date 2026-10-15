import '../../compartilhado/erros.dart';

/// Modelos do contrato api/openapi.yaml, decodificados à mão (sem geração de
/// código): o contrato é pequeno e estável.

enum StatusValidacao {
  aberta,
  concluida,
  expirada;

  static StatusValidacao de(String? valor) => StatusValidacao.values.firstWhere(
    (s) => s.name == valor,
    orElse: () => StatusValidacao.expirada,
  );
}

class Localizacao {
  const Localizacao({required this.id, required this.nome});

  factory Localizacao.fromJson(Map<String, Object?> json) => Localizacao(
    id: json['id'] as String? ?? '',
    nome: json['nome'] as String? ?? '',
  );

  final String id;
  final String nome;
}

class Maquina {
  const Maquina({
    required this.id,
    required this.modelo,
    required this.localizacao,
    this.ultimaValidacaoEm,
  });

  factory Maquina.fromJson(Map<String, Object?> json) => Maquina(
    id: json['id'] as String? ?? '',
    modelo: json['modelo'] as String? ?? '',
    localizacao: Localizacao.fromJson(
      (json['localizacao'] as Map?)?.cast<String, Object?>() ?? const {},
    ),
    ultimaValidacaoEm: _data(json['ultimaValidacaoEm']),
  );

  final String id;
  final String modelo;
  final Localizacao localizacao;
  final DateTime? ultimaValidacaoEm;
}

class Leituras {
  const Leituras({required this.medidor, required this.unidadesVendidas});

  factory Leituras.fromJson(Map<String, Object?> json) => Leituras(
    medidor: (json['medidor'] as num?)?.toInt() ?? 0,
    unidadesVendidas: (json['unidadesVendidas'] as num?)?.toInt() ?? 0,
  );

  final int medidor;
  final int unidadesVendidas;

  Map<String, Object?> toJson() => {
    'medidor': medidor,
    'unidadesVendidas': unidadesVendidas,
  };
}

class Validacao {
  const Validacao({
    required this.idValidacao,
    required this.status,
    required this.iniciadaEm,
    required this.expiraEm,
    required this.maquina,
    this.contrassenha,
    this.proximaContrassenha,
    this.concluidaEm,
    this.leituras,
    this.codigoRotacionado,
    this.avisos = const [],
  });

  factory Validacao.fromJson(Map<String, Object?> json) => Validacao(
    idValidacao: json['idValidacao'] as String? ?? '',
    status: StatusValidacao.de(json['status'] as String?),
    contrassenha: json['contrassenha'] as String?,
    proximaContrassenha: json['proximaContrassenha'] as String?,
    iniciadaEm: _data(json['iniciadaEm']) ?? DateTime.now().toUtc(),
    expiraEm: _data(json['expiraEm']) ?? DateTime.now().toUtc(),
    concluidaEm: _data(json['concluidaEm']),
    maquina: Maquina.fromJson(
      (json['maquina'] as Map?)?.cast<String, Object?>() ?? const {},
    ),
    leituras: json['leituras'] is Map
        ? Leituras.fromJson((json['leituras'] as Map).cast<String, Object?>())
        : null,
    codigoRotacionado: json['codigoRotacionado'] as bool?,
    avisos: _textos(json['avisos']),
  );

  final String idValidacao;
  final StatusValidacao status;
  final String? contrassenha;
  final String? proximaContrassenha;
  final DateTime iniciadaEm;
  final DateTime expiraEm;
  final DateTime? concluidaEm;
  final Maquina maquina;
  final Leituras? leituras;
  final bool? codigoRotacionado;
  final List<String> avisos;

  bool get aberta => status == StatusValidacao.aberta;

  /// Tempo restante para concluir (RN-04); zero quando já passou.
  Duration restante(DateTime agora) {
    final d = expiraEm.difference(agora.toUtc());
    return d.isNegative ? Duration.zero : d;
  }
}

class Conclusao {
  const Conclusao({
    required this.idValidacao,
    required this.status,
    required this.concluidaEm,
    required this.codigoRotacionado,
    this.avisos = const [],
  });

  factory Conclusao.fromJson(Map<String, Object?> json) => Conclusao(
    idValidacao: json['idValidacao'] as String? ?? '',
    status: StatusValidacao.de(json['status'] as String?),
    concluidaEm: _data(json['concluidaEm']) ?? DateTime.now().toUtc(),
    codigoRotacionado: json['codigoRotacionado'] as bool? ?? false,
    avisos: _textos(json['avisos']),
  );

  /// Reconstrói a conclusão a partir de uma consulta, quando a resposta da
  /// conclusão se perdeu na rede (CE-03).
  factory Conclusao.deValidacao(Validacao v) => Conclusao(
    idValidacao: v.idValidacao,
    status: v.status,
    concluidaEm: v.concluidaEm ?? DateTime.now().toUtc(),
    codigoRotacionado: v.codigoRotacionado ?? false,
    avisos: v.avisos,
  );

  final String idValidacao;
  final StatusValidacao status;
  final DateTime concluidaEm;
  final bool codigoRotacionado;
  final List<String> avisos;
}

/// Erro do serviço em Problem Details (RFC 9457, ADR-0009).
class Problema {
  const Problema({
    required this.status,
    required this.codigo,
    this.title,
    this.detail,
    this.idValidacao,
  });

  factory Problema.fromJson(Map<String, Object?> json) => Problema(
    status: (json['status'] as num?)?.toInt() ?? 0,
    codigo: json['codigo'] as String? ?? 'erro_interno',
    title: json['title'] as String?,
    detail: json['detail'] as String?,
    idValidacao: json['idValidacao'] as String?,
  );

  final int status;
  final String codigo;
  final String? title;
  final String? detail;
  final String? idValidacao;

  ErroDominio paraErro() => ErroDominio(
    CodigoErro.deValor(codigo),
    detail ?? title ?? codigo,
    idValidacao: idValidacao,
  );
}

/// Texto do aviso devolvido na conclusão (RN-07, ADR-0011).
String mensagemDoAviso(String aviso) => switch (aviso) {
  'medidor_menor_que_anterior' => 'O medidor informado é menor que o da última visita. A conclusão foi registrada; confira o valor na máquina.',
  _ => aviso,
};

DateTime? _data(Object? valor) {
  if (valor is! String || valor.isEmpty) return null;
  return DateTime.tryParse(valor)?.toUtc();
}

List<String> _textos(Object? valor) => valor is List
    ? valor.whereType<String>().toList(growable: false)
    : const [];
