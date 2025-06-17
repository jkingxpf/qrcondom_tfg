import 'package:flutter_test/flutter_test.dart';
import 'package:flutter/material.dart';
import 'package:front_tfg/main.dart';

void main() {
  testWidgets('Cambia a la siguiente pantalla al obtener la respuesta de la api', (WidgetTester tester) async {
    await tester.pumpWidget(const MyApp());

    await tester.pumpAndSettle();

    expect(find.byKey(const Key('boton_consentimiento')), findsOneWidget);

    await tester.tap(find.byKey(const Key('boton_consentimiento')));
    await tester.pumpAndSettle();

    expect(find.byKey(Key("lector_QR")), findsOneWidget);
  });
}