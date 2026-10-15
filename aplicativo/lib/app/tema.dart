import 'package:flutter/material.dart';

/// Tema escuro com destaque verde, seguindo os esboços da Figura 2 do TCC 1.
ThemeData temaDoAplicativo() {
  final esquema = ColorScheme.fromSeed(
    seedColor: const Color(0xFF2ECC71),
    brightness: Brightness.dark,
  );
  return ThemeData(
    useMaterial3: true,
    colorScheme: esquema,
    scaffoldBackgroundColor: const Color(0xFF101418),
    appBarTheme: const AppBarTheme(centerTitle: false),
    cardTheme: CardThemeData(
      color: const Color(0xFF1A2027),
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(16)),
    ),
    inputDecorationTheme: const InputDecorationTheme(
      border: OutlineInputBorder(),
    ),
    filledButtonTheme: FilledButtonThemeData(
      style: FilledButton.styleFrom(minimumSize: const Size.fromHeight(52)),
    ),
  );
}

/// Estilo da contrassenha: grande, monoespaçada e com dígitos tabulares,
/// legível a um metro de distância (RNF-16).
const estiloContrassenha = TextStyle(
  fontFamily: 'monospace',
  fontSize: 48,
  fontWeight: FontWeight.w700,
  letterSpacing: 6,
  fontFeatures: [FontFeature.tabularFigures()],
);
