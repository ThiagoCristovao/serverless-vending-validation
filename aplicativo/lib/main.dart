import 'package:firebase_core/firebase_core.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'app/app.dart';

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();
  // No Android, a configuração vem de android/app/google-services.json, gerado
  // pelo Terraform (make app-google-services) e nunca versionado.
  await Firebase.initializeApp();
  runApp(const ProviderScope(child: Aplicativo()));
}
