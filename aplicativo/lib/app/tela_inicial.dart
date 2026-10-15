import 'package:flutter/material.dart';

/// Tela exibida enquanto a sessão do Firebase ainda não foi resolvida, para
/// não mostrar a câmera nem o login antes de saber se há operador autenticado.
class TelaInicial extends StatelessWidget {
  const TelaInicial({super.key});

  @override
  Widget build(BuildContext context) {
    return const Scaffold(body: Center(child: CircularProgressIndicator()));
  }
}
