# aplicativo — aplicativo do operador (Flutter)

Alvo único nesta versão: **Android 8.0+** (ADR-0004, RNF-17). Flutter fixado em `.fvmrc` e
gerenciado pelo FVM; use `fvm flutter ...` neste diretório (ou os alvos `make app-*`).

| Sprint | Estado | Conteúdo |
|---|---|---|
| 6 | **concluída** (verificada em Motorola Edge 50 Pro, Android 15, em 15/10/2026) | estrutura em camadas com Riverpod, login com Firebase Authentication, leitura do QR com `mobile_scanner`, validação local do adesivo (estrutura, chave, assinatura Ed25519, validade), estados de câmera e de erro |
| 7 | **concluída** (verificada no mesmo aparelho contra o serviço em `svv-dev`, em 15/10/2026) | cliente HTTP com JWT e `Idempotency-Key`, retentativa, tela de resultado (contrassenha, próximo código, dados, leituras, conclusão), erros do serviço e de rede, faixa de conectividade, testes dos fluxos |

## Estrutura

```
lib/
├── main.dart                        inicializa o Firebase (google-services.json) e o ProviderScope
├── app/
│   ├── app.dart                     MaterialApp.router, tema, pt-BR
│   ├── roteador.dart                GoRouter: /, /entrar, /leitura e /resultado; redirect reage ao provedor de sessão
│   ├── tela_inicial.dart            espera enquanto a sessão do Firebase é resolvida
│   └── tema.dart                    tema escuro com destaque verde (Figura 2); estilo da contrassenha
├── compartilhado/
│   └── erros.dart                   CodigoErro (espelha o contrato), ErroDominio, mensagens ao operador
├── compartilhado/ (cont.)
│   ├── configuracao.dart            SVV_API_BASE (gateway) e versão, via --dart-define
│   ├── uuid.dart                    UUID v4 para a Idempotency-Key (uma por leitura)
│   └── conectividade.dart           faixa global "Sem conexão" (connectivity_plus)
└── recursos/
    ├── validacao/
    │   ├── modelos.dart             Validacao, Maquina, Leituras, Conclusao, Problema (contrato)
    │   ├── servico_validacao.dart   porta do serviço (interface); falso nos testes
    │   ├── cliente_http.dart        dio: token do Firebase, renovação em 401, retentativa 1 s/2 s/4 s, Problem Details
    │   ├── validacao_controlador.dart estados: enviando, aberta, concluída, falha; CE-02/CE-03/CE-11
    │   └── tela_resultado.dart      Figura 2b: contrassenha, próximo código, dados, leituras, conclusão, avisos
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
├── recursos/leitura/payload_qr_test.dart          vetores gerados pela ferramenta Go (compatibilidade Dart ↔ Go)
├── recursos/autenticacao/tela_entrar_test.dart    validação do formulário
├── recursos/validacao/modelos_test.dart           leitura dos exemplos do contrato (api/exemplos)
├── recursos/validacao/cliente_http_test.dart      transporte falso: token, idempotência, 401, Problem Details, retentativa
└── recursos/validacao/tela_resultado_test.dart    serviço falso: fluxo principal, 409 + retomar, sem rede, CE-03, erro de negócio
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

## Comportamento em falhas (Sprint 7)

| Situação | Comportamento |
|---|---|
| Sem rede ou 503 ao abrir | até 3 reenvios automáticos (1 s, 2 s, 4 s) com a mesma `Idempotency-Key`; depois, mensagem e "Tentar novamente" |
| Resposta da conclusão perdida | o app consulta `GET /validacoes/{id}` antes de reenviar; se já concluiu, mostra a conclusão (CE-03) |
| 401 do gateway | renova o ID token uma vez e repete; se persistir, encerra a sessão e volta ao login (CE-04) |
| 409 validação em andamento | mensagem e "Retomar validação em andamento" (consulta); se for de outro operador, 403 explica |
| Prazo de 15 min vencido na tela | contagem regressiva chega a zero, estado passa a expirado, "Ler outro código" |
| Erros de negócio na conclusão | mensagem pelo `codigo` do Problem Details; a validação continua aberta |
| Sem rede (geral) | faixa persistente no topo; a leitura local do QR continua funcionando |

## Critério da Sprint 7

"O fluxo da Figura 1 é executável de ponta a ponta em dispositivo físico, incluindo os caminhos de
exceção."

**Verificado em 15/10/2026** no mesmo Motorola Edge 50 Pro, contra o serviço implantado em `svv-dev`:

- duas visitas completas à máquina `VM-2047`: abertura, contrassenha e próximo código na tela,
  leituras (15000/140 e depois 16000/150), confirmação da rotação e conclusão, com contrassenhas
  diferentes em cada visita;
- nova leitura do mesmo adesivo com a validação ainda aberta: `409` com a mensagem de validação em
  andamento e retomada pelo botão correspondente (CE-11);
- conclusão tentada em modo avião: faixa "Sem conexão", mensagem de rede e conclusão na retentativa
  após religar a rede (CE-02/CE-03).

Conferido no servidor: as duas validações `concluida` no Firestore com as leituras digitadas,
`maquinas/VM-2047` com `contadorCodigo = 3`, `ultimoMedidor = 16000` e sem validação aberta; quatro
mensagens na assinatura `svv-dev-sistema-central` na ordem iniciada → concluída → iniciada → concluída,
cada uma entregue uma única vez, e DLQ vazia. Logs da função na sequência: `201`, `200`, `201`, `409`,
`GET 200`, `200`, todos abaixo de 1,1 s (o primeiro com partida a frio).

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
