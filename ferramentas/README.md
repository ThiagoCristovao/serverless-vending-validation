# ferramentas — scripts de apoio

Utilitários que não fazem parte do produto, mas sustentam o desenvolvimento e a avaliação.

| Ferramenta | Onde | Sprint | Propósito |
|---|---|---|---|
| `gerar-qr` | `servico/cmd/gerar-qr` | 5 | Gera o par de chaves Ed25519, assina payloads conforme `docs/payload-qr.md` e emite o PNG do QR |
| `semear-firestore` | `servico/cmd/semear-firestore` | 5 | Cadastra máquina, operador e chave pública no Firestore do projeto ou no emulador |
| `token-operador` | `servico/cmd/token-operador` | 5 | Cria (opcionalmente) um operador no Firebase Auth e devolve um ID token para chamar o gateway |
| `emuladores/` | aqui | 5 | `docker-compose.yml` com os emuladores do Firestore e do Pub/Sub para os testes de integração |
| `injecao-falhas/` | aqui | 9 | Scripts que desligam o consumidor, simulam rede intermitente e indisponibilidade da autenticação |
| `carga/` | aqui | 10 | Scripts k6 para os testes de carga e coleta de p50/p95/p99, cold start e regime contínuo |

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
