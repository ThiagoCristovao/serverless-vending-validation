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
| `atrasar` | segura a requisição por 25 s e só então a encaminha, mesmo que o aplicativo já tenha desistido (rede lenta: o pedido chega atrasado ao serviço) | tempo esgotado (20 s) → reenvios com a mesma `Idempotency-Key`; o serviço recebe todos e responde a mesma validação |
| `indisponivel` | responde `503 dependencia_indisponivel` | reenvios → "Serviço indisponível" |
| `cortar-resposta` | encaminha, espera o gateway responder e fecha a conexão sem devolver | o serviço processou, o aplicativo não soube: antes de reenviar uma conclusão, consulta `GET /validacoes/{id}` (CE-03) |
| `intermitente` | alterna uma requisição cortada e uma normal | os reenvios automáticos absorvem as falhas |

```bash
make injecao-proxy MODO=normal                 # porta 8099; imprime o IP local
curl -s -X PUT localhost:8099/admin/modo -d '{"modo":"cortar-resposta"}'
curl -s -X PUT localhost:8099/admin/modo -d '{"modo":"atrasar","atraso":"25s","proporcao":1}'
curl -s localhost:8099/admin/modo
```

Cada requisição gera uma linha JSON no stdout (`metodo`, `caminho`, `modoAplicado`,
`chaveIdempotencia` quando o cabeçalho está presente, `status`, `duracao`): é a evidência do lado
do computador; o Cloud Logging e o Firestore completam o quadro (o aplicativo não escreve no
`logcat`).

### Roteiro no celular (APK de injeção instalado, proxy em `normal`)

| # | Cenário | Passos | Esperado |
|---|---|---|---|
| D1 | Sem rede ao validar (CE-01/CE-02) | proxy em `cortar`; ler o QR; "Validar máquina" | "Reenviando…" e, após os três reenvios, "Sem conexão" com "Tentar novamente"; o proxy registra 4 cortes com a **mesma** `Idempotency-Key`. Voltar o proxy a `normal` e tocar "Tentar novamente": `201` sem validação duplicada |
| D2 | Rede lenta (CE-02) | proxy em `atrasar` (25 s); ler o QR; "Validar máquina" | o aplicativo desiste em 20 s e reenvia; ao voltar a `normal`, "Tentar novamente" conclui |
| D3 | Resposta da conclusão perdida (CE-03) | com a validação aberta em `normal`, trocar para `cortar-resposta`; preencher as leituras; "Concluir visita"; devolver o proxy a `normal` logo após a resposta descartada (ex.: `tail -n0 -F` no log com `grep -m1 "resposta do gateway descartada"` seguido do `PUT`) | o proxy mostra o `200` do gateway descartado; o aplicativo consulta `GET` (reenvios em 1, 2 e 4 s) e exibe a conclusão **sem reenviar** o `POST`. Se a queda durar mais que as consultas, aparece "Sem conexão" e "Concluir visita" repete o ciclo |
| D4 | Intermitência (RF-23) | proxy em `cortar` com `proporcao` 0,5 (cortes aleatórios); fluxo completo (validar e concluir). O modo `intermitente` (alternância estrita) é o pior caso: use-o para ver os reenvios, não para completar o fluxo | o fluxo completa; o proxy mostra as requisições cortadas e as reenviadas (abertura com a mesma chave; conclusão reenviada depois de a consulta confirmar "aberta"). Sob alternância estrita o par `POST`/`GET` deixa o `POST` sempre cortado e o aplicativo desiste após 4 envios com "Sem conexão", sem duplicar |

Ao final, reinstale o APK normal (`make app-apk` e `adb install -r`).

## dns-bloqueio.sh — provedor de autenticação indisponível

Sobe um `dnsmasq` neste computador que responde `NXDOMAIN` para os hosts do Firebase
Authentication e encaminha o resto a resolvedores públicos: `www.googleapis.com` (o SDK Android
entra por `identitytoolkit/v3` nesse host), `identitytoolkit.googleapis.com` (API v1) e
`securetoken.googleapis.com` (renovação do token). O gateway continua resolvendo: só a
autenticação "cai". Desligue os dados móveis do celular, para o Android não contornar o DNS do
Wi-Fi.

```bash
make injecao-dns            # porta 53, pede sudo; imprime o IP a configurar no celular
```

No celular: dados móveis desligados; Wi-Fi → rede atual → editar → IP **estático** (mesmo IP,
gateway e prefixo que o DHCP deu) → DNS 1 = IP deste computador (DNS 2 em branco). O resolvedor do
Android guarda as respostas pelo TTL e o SDK reaproveita conexões abertas: depois de ativar o
bloqueio, desligue e ligue o Wi-Fi antes de tentar entrar. Desfaça tudo ao terminar.

### Roteiro no celular (APK normal)

Com a sessão já iniciada, a ordem E2, E1, E3 usa uma única execução do `dnsmasq`.

| # | Cenário | Passos | Esperado |
|---|---|---|---|
| E2 | Sessão iniciada continua operando (CE-05) | com a sessão iniciada, ativar o bloqueio; ler o QR, validar e concluir | o fluxo completa normalmente: o token em cache vale 1 h e o gateway não depende do provedor; o `dnsmasq` só vê a consulta do gateway |
| E1 | Login com a autenticação indisponível (CE-05) | sair do aplicativo; desligar e ligar o Wi-Fi; tentar entrar | "Autenticação indisponível no momento" (a rede está no ar, só o provedor não responde); o `dnsmasq` registra `config www.googleapis.com is NXDOMAIN` |
| E3 | Restabelecimento | parar o `dnsmasq` e voltar o DNS do celular ao automático; entrar | o login volta a funcionar |

## Verificação

```bash
make injecao-verificar      # vet + testes do proxy (modos, cabeçalhos, API administrativa)
```
