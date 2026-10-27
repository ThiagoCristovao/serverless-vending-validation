# ferramentas/injecao-falhas — injeção de falhas no dispositivo (Sprint 9)

Ferramentas para provocar, de forma controlada e reversível, as falhas que a avaliação de
resiliência exige no lado do **dispositivo**: rede intermitente entre o aplicativo e o gateway e
indisponibilidade do provedor de autenticação (ADR-0014). As falhas do lado do serviço (sistema
central, Pub/Sub e Firestore indisponíveis) ficam em `sistema-central-simulado/cmd/experimento`
(`make experimento`). Os resultados são consolidados em
[docs/avaliacao/resiliencia.md](../../docs/avaliacao/resiliencia.md).

## proxy-rede — rede intermitente

Proxy HTTP entre o aplicativo e o API Gateway. O celular, na mesma rede Wi-Fi, usa um APK de
depuração compilado com `SVV_API_BASE=http://<ip-deste-computador>:8099/v1`
(`make app-apk-injecao`; o manifesto de depuração permite tráfego HTTP em claro).

| Modo | Efeito | O que o aplicativo vê |
|---|---|---|
| `normal` | encaminha tudo | fluxo normal |
| `cortar` | fecha a conexão sem responder | erro de rede → reenvios em 1 s, 2 s e 4 s → "Sem conexão" e "Tentar novamente" |
| `atrasar` | segura a resposta por 25 s | tempo esgotado (20 s) → reenvios |
| `indisponivel` | responde `503 dependencia_indisponivel` | reenvios → "Serviço indisponível" |
| `cortar-resposta` | encaminha, espera o gateway responder e fecha a conexão sem devolver | o serviço processou, o aplicativo não soube: antes de reenviar uma conclusão, consulta `GET /validacoes/{id}` (CE-03) |
| `intermitente` | alterna uma requisição cortada e uma normal | os reenvios automáticos absorvem as falhas |

```bash
make injecao-proxy MODO=normal                 # porta 8099; imprime o IP local
curl -s -X PUT localhost:8099/admin/modo -d '{"modo":"cortar-resposta"}'
curl -s -X PUT localhost:8099/admin/modo -d '{"modo":"atrasar","atraso":"25s","proporcao":1}'
curl -s localhost:8099/admin/modo
```

Cada requisição gera uma linha JSON no stdout (`metodo`, `caminho`, `modoAplicado`, `status`,
`duracao`): é a evidência do lado do computador; `adb logcat` e o Cloud Logging completam o quadro.

### Roteiro no celular (APK de injeção instalado, proxy em `normal`)

| # | Cenário | Passos | Esperado |
|---|---|---|---|
| D1 | Sem rede ao validar (CE-01/CE-02) | proxy em `cortar`; ler o QR; "Validar máquina" | "Reenviando…" e, após os três reenvios, "Sem conexão" com "Tentar novamente"; o proxy registra 4 cortes com a **mesma** `Idempotency-Key`. Voltar o proxy a `normal` e tocar "Tentar novamente": `201` sem validação duplicada |
| D2 | Rede lenta (CE-02) | proxy em `atrasar` (25 s); ler o QR; "Validar máquina" | o aplicativo desiste em 20 s e reenvia; ao voltar a `normal`, "Tentar novamente" conclui |
| D3 | Resposta da conclusão perdida (CE-03) | com a validação aberta em `normal`, trocar para `cortar-resposta`; preencher as leituras; "Concluir visita"; voltar a `normal`; "Tentar novamente" | o proxy mostra o `200` do gateway descartado; na retentativa o aplicativo faz `GET` e exibe a conclusão **sem reenviar** o `POST` |
| D4 | Intermitência (RF-23) | proxy em `intermitente`; fluxo completo (validar e concluir) | o fluxo completa; o proxy mostra as requisições cortadas e as reenviadas |

Ao final, reinstale o APK normal (`make app-apk` e `adb install -r`).

## dns-bloqueio.sh — provedor de autenticação indisponível

Sobe um `dnsmasq` neste computador que responde `NXDOMAIN` para
`identitytoolkit.googleapis.com` e `securetoken.googleapis.com` (hosts do Firebase
Authentication) e encaminha o resto a resolvedores públicos. O gateway continua resolvendo: só
a autenticação "cai".

```bash
make injecao-dns            # porta 53, pede sudo; imprime o IP a configurar no celular
```

No celular: Wi-Fi → rede atual → editar → IP **estático** → DNS 1 = IP deste computador (DNS 2
em branco). Desfaça ao terminar.

### Roteiro no celular (APK normal)

| # | Cenário | Passos | Esperado |
|---|---|---|---|
| E1 | Login com a autenticação indisponível (CE-05) | sair do aplicativo; ativar o bloqueio; tentar entrar | "Autenticação indisponível no momento" (a rede está no ar, só o provedor não responde); o `dnsmasq` registra as consultas bloqueadas |
| E2 | Sessão iniciada continua operando (CE-05) | desativar o bloqueio; entrar; ativar o bloqueio de novo; ler o QR, validar e concluir | o fluxo completa normalmente: o token em cache vale 1 h e o gateway não depende do provedor |
| E3 | Restabelecimento | desativar o bloqueio (ou desfazer o DNS estático); sair e entrar | o login volta a funcionar |

## Verificação

```bash
make injecao-verificar      # vet + testes do proxy (modos, cabeçalhos, API administrativa)
```
