# ferramentas — scripts de apoio

Utilitários que não fazem parte do produto, mas sustentam o desenvolvimento e a avaliação.

| Ferramenta | Onde | Sprint | Propósito |
|---|---|---|---|
| `gerar-qr` | `servico/cmd/gerar-qr` | 5 | Gera o par de chaves Ed25519, assina payloads conforme `docs/payload-qr.md` e emite o PNG do QR |
| `semear-firestore` | `servico/cmd/semear-firestore` | 5 | Cadastra máquina, operador e chave pública no Firestore do projeto ou no emulador |
| `token-operador` | `servico/cmd/token-operador` | 5 | Cria (opcionalmente) um operador no Firebase Auth e devolve um ID token para chamar o gateway |
| `emuladores/` | aqui | 5 | `docker-compose.yml` com os emuladores do Firestore e do Pub/Sub para os testes de integração |
| `central` | `sistema-central-simulado/cmd/central` | 8 | Sistema central simulado: consome a assinatura, registra eventos de forma idempotente e expõe a API administrativa (falhas, pausa) |
| suíte e2e | `sistema-central-simulado/e2e` | 8 | Testes de ponta a ponta contra o ambiente dev (`make e2e`): fluxo principal, duplicatas e DLQ |
| `experimento` | `sistema-central-simulado/cmd/experimento` | 9 | Cenários de injeção de falhas no lado do serviço (sistema central, Pub/Sub e Firestore indisponíveis) com relatório de evidências (`make experimento`) |
| `proxy-rede` | `injecao-falhas/cmd/proxy-rede` | 9 | Proxy HTTP entre o aplicativo e o gateway que corta, atrasa, responde 503 ou descarta respostas (`make injecao-proxy`) |
| `dns-bloqueio.sh` | `injecao-falhas/` | 9 | `dnsmasq` que torna o Firebase Authentication irresolúvel para o celular (`make injecao-dns`) |
| `carga` | `sistema-central-simulado/cmd/carga` | 10 | Teste de carga em laço fechado com operadores virtuais (`make carga-preparar`, `make carga`) e medição de cold start (`make carga-frio`), com relatório de evidências (ADR-0015) |

As ferramentas em Go vivem dentro do módulo `servico` para reutilizar o domínio (assinatura do
QR, documentos do Firestore). Exemplo de preparação de um ambiente para testes manuais:

```bash
cd servico
go run ./cmd/gerar-qr -gerar-chaves -saida ../chaves            # chaves (fora do git)
go run ./cmd/token-operador -google-services ../aplicativo/android/app/google-services.json \
   -email operador1@exemplo.invalid -senha 'uma-senha-forte' -criar     # devolve uid e idToken
go run ./cmd/semear-firestore -projeto svv-dev -operador-uid <uid> -chave-publica ../chaves/chave-publica-qr.txt
go run ./cmd/gerar-qr -chave-privada ../chaves/chave-privada-qr.txt -png ../chaves/vm-2047.png
```
