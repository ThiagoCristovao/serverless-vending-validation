import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:svv/recursos/autenticacao/tela_entrar.dart';

// A tela só toca o Firebase quando o operador envia um formulário válido;
// por isso a validação local pode ser testada sem o Firebase inicializado.
void main() {
  Future<void> montar(WidgetTester tester) async {
    await tester.pumpWidget(
      const ProviderScope(child: MaterialApp(home: TelaEntrar())),
    );
    await tester.pumpAndSettle();
  }

  testWidgets('exige e-mail válido e senha antes de autenticar', (
    tester,
  ) async {
    await montar(tester);
    await tester.tap(find.widgetWithText(FilledButton, 'Entrar'));
    await tester.pump();
    expect(find.text('Informe um e-mail válido'), findsOneWidget);
    expect(find.text('Informe a senha'), findsOneWidget);

    await tester.enterText(
      find.widgetWithText(TextFormField, 'E-mail'),
      'sem-arroba',
    );
    await tester.enterText(find.widgetWithText(TextFormField, 'Senha'), 'x');
    await tester.tap(find.widgetWithText(FilledButton, 'Entrar'));
    await tester.pump();
    expect(find.text('Informe um e-mail válido'), findsOneWidget);
    expect(find.text('Informe a senha'), findsNothing);
  });

  testWidgets('alterna a visibilidade da senha', (tester) async {
    await montar(tester);
    final campo = find.widgetWithText(TextFormField, 'Senha');
    expect(
      tester
          .widget<TextField>(
            find.descendant(of: campo, matching: find.byType(TextField)),
          )
          .obscureText,
      isTrue,
    );
    await tester.tap(find.byTooltip('Mostrar senha'));
    await tester.pump();
    expect(
      tester
          .widget<TextField>(
            find.descendant(of: campo, matching: find.byType(TextField)),
          )
          .obscureText,
      isFalse,
    );
  });
}
