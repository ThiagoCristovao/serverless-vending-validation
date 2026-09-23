# aplicativo — aplicativo do operador (Flutter)

Implementado a partir da **Sprint 6** (autenticação e leitura) e **Sprint 7** (integração).
Alvo único nesta versão: **Android** (ADR-0004). Versão do Flutter fixada em `.fvmrc`, gerenciada
pelo FVM; use `fvm flutter ...` neste diretório.

## Criação do projeto (Sprint 6)

```bash
cd aplicativo
fvm flutter create . --org br.edu.utfpr --project-name svv --platforms android
# pacote resultante: br.edu.utfpr.svv (o mesmo registrado no Firebase pelo módulo autenticacao)
make app-google-services   # grava android/app/google-services.json a partir do Terraform
```

## Estrutura prevista

```
lib/
├── main.dart
├── app/                    roteamento, tema, inicialização do Firebase
├── recursos/
│   ├── autenticacao/       tela de login, provedor de sessão (Firebase Auth)
│   ├── leitura/            tela de leitura do QR (mobile_scanner), validação local do payload
│   └── validacao/          tela de resultado: contrassenha, dados da máquina, leituras, conclusão
└── compartilhado/          cliente HTTP com injeção do JWT e Idempotency-Key, modelos gerados
                            do contrato, tratamento de erros (Problem Details), conectividade
```

## Decisões já tomadas

- Gerenciamento de estado com **Riverpod** (ADR-0005).
- Pacotes previstos: `firebase_core`, `firebase_auth`, `mobile_scanner`, `flutter_riverpod`,
  `go_router`, `dio`, `freezed` + `json_serializable`, `connectivity_plus`, `cryptography`
  (verificação Ed25519 do QR, desejável). Versões definidas na Sprint 6.
- `google-services.json` é gerado pelo Terraform (módulo `autenticacao`) via `make app-google-services` e **não** é versionado.
- Estados de tela definidos em `docs/arquitetura.md`, seção Telas e estados.
