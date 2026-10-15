import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../app/roteador.dart';
import '../../app/tema.dart';
import '../../compartilhado/erros.dart';
import '../leitura/leitor_controlador.dart';
import 'modelos.dart';
import 'validacao_controlador.dart';

/// Tela de resultado da validação (Figura 2b do TCC 1): contrassenha em
/// destaque, próximo código, dados da máquina, leituras e conclusão.
class TelaResultado extends ConsumerWidget {
  const TelaResultado({super.key});

  void _novaLeitura(BuildContext context, WidgetRef ref) {
    ref.read(validacaoProvider.notifier).reiniciar();
    ref.read(leitorProvider.notifier).reiniciar();
    context.go(Rotas.leitura);
  }

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final estado = ref.watch(validacaoProvider);
    return Scaffold(
      appBar: AppBar(
        title: Text(switch (estado) {
          ValidacaoConcluida() => 'Visita concluída',
          _ => 'Validação da máquina',
        }),
        leading: IconButton(
          tooltip: 'Nova leitura',
          icon: const Icon(Icons.qr_code_scanner),
          onPressed: () => _novaLeitura(context, ref),
        ),
      ),
      body: SafeArea(
        child: switch (estado) {
          SemValidacao() => _Vazio(aoLer: () => _novaLeitura(context, ref)),
          Enviando() => const _Carregando('Validando a máquina…'),
          FalhaValidacao() => _Falha(
            estado: estado,
            aoNovaLeitura: () => _novaLeitura(context, ref),
          ),
          ValidacaoAberta() => _Aberta(estado: estado),
          ValidacaoConcluida() => _Concluida(
            estado: estado,
            aoNovaLeitura: () => _novaLeitura(context, ref),
          ),
        },
      ),
    );
  }
}

class _Carregando extends StatelessWidget {
  const _Carregando(this.texto);
  final String texto;

  @override
  Widget build(BuildContext context) => Center(
    child: Column(
      mainAxisSize: MainAxisSize.min,
      children: [
        const CircularProgressIndicator(),
        const SizedBox(height: 16),
        Text(texto),
      ],
    ),
  );
}

class _Vazio extends StatelessWidget {
  const _Vazio({required this.aoLer});
  final VoidCallback aoLer;

  @override
  Widget build(BuildContext context) => Center(
    child: Padding(
      padding: const EdgeInsets.all(24),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          const Text('Nenhuma validação em andamento.'),
          const SizedBox(height: 16),
          FilledButton(
            onPressed: aoLer,
            child: const Text('Ler o QR da máquina'),
          ),
        ],
      ),
    ),
  );
}

class _Falha extends ConsumerWidget {
  const _Falha({required this.estado, required this.aoNovaLeitura});
  final FalhaValidacao estado;
  final VoidCallback aoNovaLeitura;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final esquema = Theme.of(context).colorScheme;
    final erro = estado.erro;
    final controlador = ref.read(validacaoProvider.notifier);
    final podeRetomar =
        erro.codigo == CodigoErro.validacaoEmAndamento &&
        erro.idValidacao != null;
    return Padding(
      padding: const EdgeInsets.all(16),
      child: Card(
        color: esquema.errorContainer,
        child: Padding(
          padding: const EdgeInsets.all(20),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Icon(
                Icons.error_outline,
                size: 40,
                color: esquema.onErrorContainer,
              ),
              const SizedBox(height: 12),
              Text(
                mensagemParaOperador(erro.codigo),
                textAlign: TextAlign.center,
                style: TextStyle(color: esquema.onErrorContainer, fontSize: 16),
              ),
              const SizedBox(height: 20),
              if (estado.podeTentarNovamente)
                FilledButton(
                  onPressed: controlador.tentarNovamente,
                  child: const Text('Tentar novamente'),
                ),
              if (podeRetomar)
                FilledButton(
                  onPressed: () => controlador.retomar(erro.idValidacao!),
                  child: const Text('Retomar validação em andamento'),
                ),
              if (estado.podeTentarNovamente || podeRetomar)
                const SizedBox(height: 8),
              OutlinedButton(
                onPressed: aoNovaLeitura,
                child: const Text('Ler outro código'),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _Aberta extends ConsumerStatefulWidget {
  const _Aberta({required this.estado});
  final ValidacaoAberta estado;

  @override
  ConsumerState<_Aberta> createState() => _AbertaState();
}

class _AbertaState extends ConsumerState<_Aberta> {
  final _formulario = GlobalKey<FormState>();
  final _medidor = TextEditingController();
  final _unidades = TextEditingController();
  final _observacoes = TextEditingController();
  bool _codigoRotacionado = false;
  Timer? _relogio;
  Duration _restante = Duration.zero;

  @override
  void initState() {
    super.initState();
    _atualizarRestante();
    _relogio = Timer.periodic(
      const Duration(seconds: 1),
      (_) => _atualizarRestante(),
    );
  }

  void _atualizarRestante() {
    final restante = widget.estado.validacao.restante(DateTime.now());
    if (!mounted) return;
    setState(() => _restante = restante);
    if (restante == Duration.zero) {
      _relogio?.cancel();
      ref.read(validacaoProvider.notifier).expirou();
    }
  }

  @override
  void dispose() {
    _relogio?.cancel();
    _medidor.dispose();
    _unidades.dispose();
    _observacoes.dispose();
    super.dispose();
  }

  Future<void> _concluir() async {
    if (!(_formulario.currentState?.validate() ?? false)) return;
    FocusScope.of(context).unfocus();
    await ref
        .read(validacaoProvider.notifier)
        .concluir(
          leituras: Leituras(
            medidor: int.parse(_medidor.text),
            unidadesVendidas: int.parse(_unidades.text),
          ),
          codigoRotacionado: _codigoRotacionado,
          observacoes: _observacoes.text,
        );
  }

  String? _inteiroNaoNegativo(String? v) {
    final n = int.tryParse((v ?? '').trim());
    if (n == null || n < 0) return 'Informe um número inteiro não negativo';
    return null;
  }

  @override
  Widget build(BuildContext context) {
    final v = widget.estado.validacao;
    final esquema = Theme.of(context).colorScheme;
    final minutos = _restante.inMinutes.toString().padLeft(2, '0');
    final segundos = (_restante.inSeconds % 60).toString().padLeft(2, '0');

    return SingleChildScrollView(
      padding: const EdgeInsets.all(16),
      child: Form(
        key: _formulario,
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Card(
              child: Padding(
                padding: const EdgeInsets.symmetric(
                  vertical: 20,
                  horizontal: 16,
                ),
                child: Column(
                  children: [
                    Text(
                      'CONTRASSENHA',
                      style: Theme.of(context).textTheme.labelLarge
                          ?.copyWith(letterSpacing: 2),
                    ),
                    const SizedBox(height: 8),
                    SelectableText(
                      v.contrassenha ?? '----',
                      style: estiloContrassenha.copyWith(
                        color: esquema.primary,
                      ),
                    ),
                    const SizedBox(height: 4),
                    const Text('Digite este código no teclado da máquina'),
                    const Divider(height: 28),
                    Text(
                      'Próximo código a programar: ${v.proximaContrassenha ?? '----'}',
                      style: Theme.of(context).textTheme.titleMedium
                          ?.copyWith(fontFamily: 'monospace'),
                    ),
                    const SizedBox(height: 8),
                    Text(
                      'Expira em $minutos:$segundos',
                      style: Theme.of(context).textTheme.bodySmall,
                    ),
                  ],
                ),
              ),
            ),
            const SizedBox(height: 12),
            Card(
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  children: [
                    Text(
                      'DADOS DA MÁQUINA',
                      style: Theme.of(context).textTheme.labelLarge
                          ?.copyWith(letterSpacing: 2),
                    ),
                    const SizedBox(height: 8),
                    _Linha('ID', v.maquina.id),
                    _Linha('Modelo', v.maquina.modelo),
                    _Linha(
                      'Localização',
                      v.maquina.localizacao.nome.isEmpty
                          ? v.maquina.localizacao.id
                          : v.maquina.localizacao.nome,
                    ),
                    _Linha(
                      'Última validação',
                      v.maquina.ultimaValidacaoEm == null
                          ? 'nenhuma'
                          : _dataLocal(v.maquina.ultimaValidacaoEm!),
                    ),
                  ],
                ),
              ),
            ),
            const SizedBox(height: 12),
            Card(
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  children: [
                    Text(
                      'LEITURAS',
                      style: Theme.of(context).textTheme.labelLarge
                          ?.copyWith(letterSpacing: 2),
                    ),
                    const SizedBox(height: 12),
                    TextFormField(
                      controller: _medidor,
                      keyboardType: TextInputType.number,
                      inputFormatters: [FilteringTextInputFormatter.digitsOnly],
                      decoration: const InputDecoration(
                        labelText: 'Medidor (vendas totais)',
                      ),
                      validator: _inteiroNaoNegativo,
                      enabled: !widget.estado.concluindo,
                    ),
                    const SizedBox(height: 12),
                    TextFormField(
                      controller: _unidades,
                      keyboardType: TextInputType.number,
                      inputFormatters: [FilteringTextInputFormatter.digitsOnly],
                      decoration: const InputDecoration(
                        labelText: 'Unidades vendidas desde a última coleta',
                      ),
                      validator: _inteiroNaoNegativo,
                      enabled: !widget.estado.concluindo,
                    ),
                    const SizedBox(height: 12),
                    TextFormField(
                      controller: _observacoes,
                      maxLength: 500,
                      maxLines: 2,
                      decoration: const InputDecoration(
                        labelText: 'Observações (opcional)',
                      ),
                      enabled: !widget.estado.concluindo,
                    ),
                    CheckboxListTile(
                      value: _codigoRotacionado,
                      onChanged: widget.estado.concluindo
                          ? null
                          : (valor) => setState(
                              () => _codigoRotacionado = valor ?? false,
                            ),
                      contentPadding: EdgeInsets.zero,
                      controlAffinity: ListTileControlAffinity.leading,
                      title: Text(
                        'Programei o novo código ${v.proximaContrassenha ?? ''} na máquina',
                      ),
                      subtitle: const Text(
                        'Sem marcar, o código atual continua valendo na próxima visita',
                      ),
                    ),
                  ],
                ),
              ),
            ),
            const SizedBox(height: 12),
            if (widget.estado.erroConclusao != null) ...[
              _FaixaErro(widget.estado.erroConclusao!),
              const SizedBox(height: 12),
            ],
            FilledButton(
              onPressed: widget.estado.concluindo ? null : _concluir,
              child: widget.estado.concluindo
                  ? const SizedBox(
                      height: 22,
                      width: 22,
                      child: CircularProgressIndicator(strokeWidth: 2.5),
                    )
                  : const Text('Concluir visita'),
            ),
          ],
        ),
      ),
    );
  }
}

class _Concluida extends StatelessWidget {
  const _Concluida({required this.estado, required this.aoNovaLeitura});
  final ValidacaoConcluida estado;
  final VoidCallback aoNovaLeitura;

  @override
  Widget build(BuildContext context) {
    final esquema = Theme.of(context).colorScheme;
    final c = estado.conclusao;
    return SingleChildScrollView(
      padding: const EdgeInsets.all(16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Card(
            child: Padding(
              padding: const EdgeInsets.all(20),
              child: Column(
                children: [
                  Icon(
                    Icons.check_circle_outline,
                    size: 48,
                    color: esquema.primary,
                  ),
                  const SizedBox(height: 12),
                  Text(
                    'Máquina ${estado.validacao.maquina.id} validada',
                    style: Theme.of(context).textTheme.titleLarge,
                  ),
                  const SizedBox(height: 8),
                  Text('Concluída em ${_dataLocal(c.concluidaEm)}'),
                  Text(
                    c.codigoRotacionado
                        ? 'Código supervisor rotacionado.'
                        : 'Código supervisor mantido.',
                  ),
                ],
              ),
            ),
          ),
          for (final aviso in c.avisos) ...[
            const SizedBox(height: 12),
            Card(
              color: esquema.tertiaryContainer,
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: Row(
                  children: [
                    Icon(
                      Icons.warning_amber_outlined,
                      color: esquema.onTertiaryContainer,
                    ),
                    const SizedBox(width: 12),
                    Expanded(
                      child: Text(
                        mensagemDoAviso(aviso),
                        style: TextStyle(color: esquema.onTertiaryContainer),
                      ),
                    ),
                  ],
                ),
              ),
            ),
          ],
          const SizedBox(height: 20),
          FilledButton(
            onPressed: aoNovaLeitura,
            child: const Text('Nova leitura'),
          ),
        ],
      ),
    );
  }
}

class _FaixaErro extends StatelessWidget {
  const _FaixaErro(this.erro);
  final ErroDominio erro;

  @override
  Widget build(BuildContext context) {
    final esquema = Theme.of(context).colorScheme;
    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: esquema.errorContainer,
        borderRadius: BorderRadius.circular(12),
      ),
      child: Row(
        children: [
          Icon(Icons.error_outline, color: esquema.onErrorContainer),
          const SizedBox(width: 12),
          Expanded(
            child: Text(
              mensagemParaOperador(erro.codigo),
              style: TextStyle(color: esquema.onErrorContainer),
            ),
          ),
        ],
      ),
    );
  }
}

class _Linha extends StatelessWidget {
  const _Linha(this.rotulo, this.valor);
  final String rotulo;
  final String valor;

  @override
  Widget build(BuildContext context) {
    final estilo = Theme.of(context).textTheme;
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 3),
      child: Row(
        mainAxisAlignment: MainAxisAlignment.spaceBetween,
        children: [
          Text(
            rotulo,
            style: estilo.bodyMedium?.copyWith(color: estilo.bodySmall?.color),
          ),
          Flexible(
            child: Text(
              valor,
              textAlign: TextAlign.end,
              style: estilo.bodyLarge?.copyWith(fontWeight: FontWeight.w600),
            ),
          ),
        ],
      ),
    );
  }
}

String _dataLocal(DateTime instante) {
  final l = instante.toLocal();
  String dois(int n) => n.toString().padLeft(2, '0');
  return '${dois(l.day)}/${dois(l.month)}/${l.year} ${dois(l.hour)}:${dois(l.minute)}';
}
