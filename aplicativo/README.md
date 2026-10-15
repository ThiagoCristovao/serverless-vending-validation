# aplicativo — aplicativo do operador (Flutter)

Alvo único nesta versão: **Android 8.0+** (ADR-0004, RNF-17). Flutter fixado em `.fvmrc` e
gerenciado pelo FVM; use `fvm flutter ...` neste diretório (ou os alvos `make app-*`).

| Sprint | Estado | Conteúdo |
|---|---|---|
| 6 | **concluída** (verificada em Motorola Edge 50 Pro, Android 15, em 15/10/2026) | estrutura em camadas com Riverpod, login com Firebase Authentication, leitura do QR com `mobile_scanner`, validação local do adesivo (estrutura, chave, assinatura Ed25519, validade), estados de câmera e de erro |
| 7 | pendente | cliente HTTP com JWT e `Idempotency-Key`, tela de resultado (contrassenha, dados, leituras, conclusão), tratamento dos erros do serviço, testes dos fluxos |

## Estrutura

```
lib/
├── main.dart                        inicializa o Firebase (google-services.json) e o ProviderScope
├── app/
│   ├── app.dart                     MaterialApp.router, tema, pt-BR
│   ├── roteador.dart                GoRouter: /, /entrar e /leitura; redirect reage ao provedor de sessão
│   ├── tela_inicial.dart            espera enquanto a sessão do Firebase é resolvida
│   └── tema.dart                    tema escuro com destaque verde (Figura 2); estilo da contrassenha
├── compartilhado/
│   └── erros.dart                   CodigoErro (espelha o contrato), ErroDominio, mensagens ao operador
└── recursos/
    ├── autenticacao/
    │   ├── sessao.dart              sessaoProvider (authStateChanges) e AutenticacaoControlador (entrar/sair)
    │   └── tela_entrar.dart         formulário e-mail/senha; estados autenticando, erro, indisponível
    └── leitura/
        ├── payload_qr.dart          PayloadQr: decodificação, formato, validade, assinatura Ed25519
        ├── chaves_qr.dart           chaves públicas embutidas por kid (--dart-define=SVV_CHAVE_PUBLICA_K1)
        ├── validador_local.dart     RF-04/RF-05: chave conhecida → assinatura → validade
        ├── leitor_controlador.dart  estados: aguardando, decodificando, inválida, válida
        └── tela_leitura.dart        câmera com moldura, painel de estado, dados decodificados, câmera negada
test/
├── recursos/leitura/payload_qr_test.dart       vetores gerados pela ferramenta Go (compatibilidade Dart ↔ Go)
└── recursos/autenticacao/tela_entrar_test.dart validação do formulário
```

## Preparar e executar

```bash
make app-google-services        # android/app/google-services.json a partir do Terraform (não versionado)
make app-verificar              # format, analyze, test
make app-apk                    # build/app/outputs/flutter-apk/app-debug.apk
adb devices                     # celular com depuração USB autorizada
make app-executar               # fvm flutter run no dispositivo
```

Para testar a leitura sem máquina: gere um adesivo com `servico/cmd/gerar-qr` (ou use
`chaves/vm-2047.png`) e exiba o PNG em outra tela. Operador de testes: criado por
`servico/cmd/token-operador -criar` (ver docs/operacao/configuracao-ambiente.md, seção 6).

## Decisões

- **Riverpod** (ADR-0005), sem geração de código: `StreamProvider` para a sessão, `AsyncNotifier`
  para autenticação, `Notifier` com estados selados para a leitura.
- **Validação local completa** (RF-05 atendido): a chave pública `k1` de desenvolvimento está
  embutida; outro ambiente passa a sua com `--dart-define`. O serviço repete todas as verificações.
- **Firebase sem `firebase_options.dart`**: no Android a configuração vem do `google-services.json`
  gerado pelo Terraform, com o plugin Gradle `com.google.gms.google-services` 4.5.0.
- `minSdk = 26`; permissão de câmera é pedida pelo `mobile_scanner`; `INTERNET` declarada no manifesto
  para os builds de release.
- Pacotes: flutter_riverpod 3.4, go_router 18, mobile_scanner 7.4, firebase_core 5, firebase_auth 7,
  cryptography 2.9 (Ed25519), flutter_localizations.

## Critério da Sprint 6

Em dispositivo físico: entrar com a conta de operador, apontar para um QR válido e ver os dados
decodificados, sem comunicação com o serviço de validação. Casos negativos: QR de outro conteúdo,
adesivo vencido e adesivo adulterado devem ser recusados com mensagem.

**Verificado em 15/10/2026** em um Motorola Edge 50 Pro conectado por depuração Wi-Fi: login com o
operador de testes, leitura do adesivo `VM-2047` com o cartão de dados decodificados, recusa de um QR
qualquer com "Código não reconhecido" e saída/retorno ao login. Logcat sem erros de Flutter ou do
Android; eventos de câmera e de sessão do Firebase registrados.

Lição de conexão: o aparelho não enumerou na USB com o cabo disponível (cabo só de carga); a
depuração por Wi-Fi (`adb pair` + `adb connect`) resolveu sem cabo.
