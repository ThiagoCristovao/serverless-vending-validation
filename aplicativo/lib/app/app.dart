import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'roteador.dart';
import 'tema.dart';

/// Raiz do aplicativo do operador.
class Aplicativo extends ConsumerWidget {
  const Aplicativo({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return MaterialApp.router(
      title: 'Validação de máquinas',
      theme: temaDoAplicativo(),
      routerConfig: ref.watch(roteadorProvider),
      locale: const Locale('pt', 'BR'),
      supportedLocales: const [Locale('pt', 'BR')],
      localizationsDelegates: GlobalMaterialLocalizations.delegates,
      debugShowCheckedModeBanner: false,
    );
  }
}
