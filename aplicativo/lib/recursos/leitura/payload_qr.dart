import 'dart:convert';
import 'dart:typed_data';

import 'package:cryptography/cryptography.dart';

import '../../compartilhado/erros.dart';

/// Conteúdo decodificado do adesivo fixado na máquina (docs/payload-qr.md).
/// Espelha `dominio.PayloadQr` do serviço: mesmas regras de formato, a mesma
/// mensagem canônica assinada e a mesma noção de validade.
class PayloadQr {
  const PayloadQr({
    required this.v,
    required this.maq,
    required this.mod,
    required this.loc,
    required this.exp,
    required this.kid,
    required this.sig,
  });

  static const versao = 1;
  static const modelosSuportados = {'CN168'};
  static const _campos = {'v', 'maq', 'mod', 'loc', 'exp', 'kid', 'sig'};

  static final _padraoMaq = RegExp(r'^[A-Z0-9-]{3,32}$');
  static final _padraoLoc = RegExp(r'^[A-Z0-9-]{1,32}$');
  static final _padraoKid = RegExp(r'^[a-z0-9]{1,8}$');
  static final _padraoSig = RegExp(r'^[A-Za-z0-9_-]{86}$');
  static final _padraoData = RegExp(r'^(\d{4})-(\d{2})-(\d{2})$');

  final int v;
  final String maq;
  final String mod;
  final String loc;
  final String exp;
  final String kid;
  final String sig;

  /// Decodifica o texto lido do QR. Rejeita JSON inválido, campos
  /// desconhecidos ou ausentes e violações de formato.
  static PayloadQr decodificar(String bruto) {
    final Object? decodificado;
    try {
      decodificado = jsonDecode(bruto);
    } on FormatException {
      throw const ErroDominio(
        CodigoErro.payloadInvalido,
        'o conteúdo do QR não é um JSON válido',
      );
    }
    if (decodificado is! Map<String, Object?>) {
      throw const ErroDominio(
        CodigoErro.payloadInvalido,
        'o conteúdo do QR não é um objeto JSON',
      );
    }
    // Cópia tipada: a promoção de tipo não vale dentro das closures abaixo.
    final Map<String, Object?> mapa = decodificado;
    final desconhecidos = mapa.keys.where((k) => !_campos.contains(k));
    if (desconhecidos.isNotEmpty) {
      throw ErroDominio(
        CodigoErro.payloadInvalido,
        'campos desconhecidos: ${desconhecidos.join(', ')}',
      );
    }
    final ausentes = _campos.where((k) => !mapa.containsKey(k));
    if (ausentes.isNotEmpty) {
      throw ErroDominio(
        CodigoErro.payloadInvalido,
        'campos ausentes: ${ausentes.join(', ')}',
      );
    }
    final v = mapa['v'];
    if (v is! int) {
      throw const ErroDominio(
        CodigoErro.payloadInvalido,
        'campo v deve ser inteiro',
      );
    }
    String texto(String campo) {
      final valor = mapa[campo];
      if (valor is! String) {
        throw ErroDominio(
          CodigoErro.payloadInvalido,
          'campo $campo deve ser texto',
        );
      }
      return valor;
    }

    final p = PayloadQr(
      v: v,
      maq: texto('maq'),
      mod: texto('mod'),
      loc: texto('loc'),
      exp: texto('exp'),
      kid: texto('kid'),
      sig: texto('sig'),
    );
    p.validar();
    return p;
  }

  /// Confere estrutura e formato. Não confere validade temporal nem assinatura.
  void validar() {
    if (v != versao) {
      throw ErroDominio(
        CodigoErro.payloadInvalido,
        'versão $v do payload não é suportada',
      );
    }
    if (!_padraoMaq.hasMatch(maq)) {
      throw const ErroDominio(
        CodigoErro.payloadInvalido,
        'campo maq fora do formato esperado',
      );
    }
    if (!_padraoLoc.hasMatch(loc)) {
      throw const ErroDominio(
        CodigoErro.payloadInvalido,
        'campo loc fora do formato esperado',
      );
    }
    if (!_padraoKid.hasMatch(kid)) {
      throw const ErroDominio(
        CodigoErro.payloadInvalido,
        'campo kid fora do formato esperado',
      );
    }
    if (!_padraoSig.hasMatch(sig)) {
      throw const ErroDominio(
        CodigoErro.payloadInvalido,
        'campo sig fora do formato esperado',
      );
    }
    if (dataValidade == null) {
      throw const ErroDominio(
        CodigoErro.payloadInvalido,
        'campo exp deve ser uma data AAAA-MM-DD',
      );
    }
    if (!modelosSuportados.contains(mod)) {
      throw ErroDominio(
        CodigoErro.modeloNaoSuportado,
        'modelo "$mod" não é suportado',
      );
    }
  }

  /// Data de validade do adesivo (meia-noite UTC), ou null se `exp` for inválida.
  DateTime? get dataValidade {
    final m = _padraoData.firstMatch(exp);
    if (m == null) return null;
    final ano = int.parse(m[1]!);
    final mes = int.parse(m[2]!);
    final dia = int.parse(m[3]!);
    final data = DateTime.utc(ano, mes, dia);
    // DateTime.utc normaliza datas impossíveis (ex.: 30/02); rejeita se mudou.
    if (data.year != ano || data.month != mes || data.day != dia) return null;
    return data;
  }

  /// O adesivo vale até o fim do dia `exp` (UTC), como no serviço.
  bool vencido(DateTime referencia) {
    final validade = dataValidade;
    if (validade == null) return true;
    final r = referencia.toUtc();
    final dia = DateTime.utc(r.year, r.month, r.day);
    return dia.isAfter(validade);
  }

  /// Texto coberto pela assinatura: v|maq|mod|loc|exp|kid.
  String get mensagemCanonica => '$v|$maq|$mod|$loc|$exp|$kid';

  /// Confere a assinatura Ed25519 com a chave pública identificada por `kid`.
  Future<void> verificarAssinatura(Uint8List chavePublica) async {
    final List<int> assinatura;
    try {
      assinatura = base64Url.decode(base64Url.normalize(sig));
    } on FormatException {
      throw const ErroDominio(
        CodigoErro.assinaturaInvalida,
        'assinatura do adesivo mal formada',
      );
    }
    if (assinatura.length != 64) {
      throw const ErroDominio(
        CodigoErro.assinaturaInvalida,
        'assinatura do adesivo mal formada',
      );
    }
    final valida = await Ed25519().verify(
      utf8.encode(mensagemCanonica),
      signature: Signature(
        assinatura,
        publicKey: SimplePublicKey(chavePublica, type: KeyPairType.ed25519),
      ),
    );
    if (!valida) {
      throw const ErroDominio(
        CodigoErro.assinaturaInvalida,
        'a assinatura do adesivo não corresponde aos campos',
      );
    }
  }

  Map<String, Object?> toJson() => {
    'v': v,
    'maq': maq,
    'mod': mod,
    'loc': loc,
    'exp': exp,
    'kid': kid,
    'sig': sig,
  };

  @override
  String toString() => jsonEncode(toJson());
}
