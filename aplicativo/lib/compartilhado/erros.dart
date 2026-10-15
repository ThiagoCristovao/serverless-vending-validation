/// Códigos de erro estáveis, espelhando a enumeração `codigo` do contrato
/// (api/openapi.yaml) e os erros de validação local do adesivo. O aplicativo
/// decide a mensagem e a ação pelo código, nunca pelo texto do servidor.
enum CodigoErro {
  payloadInvalido('payload_invalido'),
  modeloNaoSuportado('modelo_nao_suportado'),
  qrExpirado('qr_expirado'),
  chaveQrDesconhecida('chave_qr_desconhecida'),
  assinaturaInvalida('assinatura_invalida'),
  maquinaDesconhecida('maquina_desconhecida'),
  maquinaInativa('maquina_inativa'),
  operadorNaoAutorizado('operador_nao_autorizado'),
  validacaoEmAndamento('validacao_em_andamento'),
  validacaoJaConcluida('validacao_ja_concluida'),
  validacaoExpirada('validacao_expirada'),
  validacaoNaoEncontrada('validacao_nao_encontrada'),
  chaveIdempotenciaConflitante('chave_idempotencia_conflitante'),
  dependenciaIndisponivel('dependencia_indisponivel'),
  erroInterno('erro_interno'),
  // Erros que só existem no aplicativo.
  semConexao('sem_conexao'),
  sessaoExpirada('sessao_expirada'),
  credenciaisInvalidas('credenciais_invalidas'),
  autenticacaoIndisponivel('autenticacao_indisponivel'),
  cameraNegada('camera_negada');

  const CodigoErro(this.valor);

  /// Valor serializado no contrato.
  final String valor;

  static CodigoErro deValor(String? valor) => CodigoErro.values.firstWhere(
    (c) => c.valor == valor,
    orElse: () => CodigoErro.erroInterno,
  );
}

/// Erro de domínio: código estável, mensagem legível e, quando aplicável, a
/// validação envolvida.
class ErroDominio implements Exception {
  const ErroDominio(this.codigo, this.mensagem, {this.idValidacao});

  final CodigoErro codigo;
  final String mensagem;
  final String? idValidacao;

  @override
  String toString() => '${codigo.valor}: $mensagem';
}

/// Mensagem exibida ao operador para cada código (docs/arquitetura.md, seção 9).
String mensagemParaOperador(CodigoErro codigo) => switch (codigo) {
  CodigoErro.payloadInvalido =>
    'Código não reconhecido. Aponte para o adesivo da máquina.',
  CodigoErro.modeloNaoSuportado =>
    'Este modelo de máquina não é atendido pelo aplicativo.',
  CodigoErro.qrExpirado =>
    'Adesivo vencido. Solicite a substituição à administração.',
  CodigoErro.chaveQrDesconhecida =>
    'Código inválido: adesivo de origem desconhecida.',
  CodigoErro.assinaturaInvalida => 'Código inválido: o adesivo não confere.',
  CodigoErro.maquinaDesconhecida => 'Máquina não cadastrada.',
  CodigoErro.maquinaInativa => 'Máquina inativa. Procure a administração.',
  CodigoErro.operadorNaoAutorizado =>
    'Operador sem permissão. Procure a administração.',
  CodigoErro.validacaoEmAndamento =>
    'Esta máquina já está em validação por outro operador.',
  CodigoErro.validacaoJaConcluida => 'Esta validação já foi concluída.',
  CodigoErro.validacaoExpirada => 'A validação expirou. Leia o QR novamente.',
  CodigoErro.validacaoNaoEncontrada => 'Validação não encontrada.',
  CodigoErro.chaveIdempotenciaConflitante =>
    'Pedido repetido com dados diferentes. Leia o QR novamente.',
  CodigoErro.dependenciaIndisponivel =>
    'Serviço indisponível. Tentando novamente…',
  CodigoErro.erroInterno => 'Falha inesperada. Tente novamente em instantes.',
  CodigoErro.semConexao => 'Sem conexão. Verifique a rede e tente novamente.',
  CodigoErro.sessaoExpirada => 'Sessão expirada. Entre novamente.',
  CodigoErro.credenciaisInvalidas => 'E-mail ou senha inválidos.',
  CodigoErro.autenticacaoIndisponivel =>
    'Autenticação indisponível no momento. Tente em instantes.',
  CodigoErro.cameraNegada => 'Precisamos da câmera para ler o código da máquina. Permita o acesso nas configurações do aparelho.',
};
