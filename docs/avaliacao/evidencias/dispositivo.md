# Evidências dos cenários no dispositivo (Sprint 9)

Registro manual da execução de 27/10/2026, das 17:24 às 18:14 (horário de Brasília; os horários
do Cloud Logging estão em UTC), em um Motorola Edge 50 Pro (Android 15) na mesma rede Wi-Fi do
computador (`192.168.1.141`; celular `192.168.1.142`). Fontes: log do proxy de rede
(`make injecao-proxy`), log de consultas do `dnsmasq` (`make injecao-dns`), Cloud Logging da
função `svv-dev-validacao`, documentos do Firestore e o relato do operador a cada passo. O
aplicativo não escreve no `logcat`, que não serviu de evidência.

| Parâmetro | Valor |
|---|---|
| Máquina | `VM-2047` (adesivo `chaves/vm-2047.png`, chave `k1`) |
| Operador | `operador1@exemplo.invalid` |
| APK dos cenários D | depuração, `SVV_API_BASE=http://192.168.1.141:8099/v1` (proxy) |
| APK dos cenários E | depuração, gateway direto |
| Política de reenvio do aplicativo | tempo esgotado 20 s; esperas 1 s, 2 s, 4 s |

## D1 — sem rede ao validar (proxy em `cortar` às 17:24:36)

| Hora | Fonte | Evento |
|---|---|---|
| 17:25:22,70 | proxy | `POST /v1/validacoes` cortado antes do envio |
| 17:25:23,74 | proxy | idem (+1 s) |
| 17:25:25,79 | proxy | idem (+2 s) |
| 17:25:29,84 | proxy | idem (+4 s) |
| — | tela | "Sem conexão" com "Tentar novamente" |
| 17:26:02 | proxy | modo `normal` |
| 17:27:49,18 | proxy | "Tentar novamente": `POST /v1/validacoes` → `201` em 5,622 s |
| 20:27:48,24 Z | função | `POST /v1/validacoes` `201`, 0,837 s (único `POST` recebido no período) |
| — | Firestore | validação `01M5ZGRCTWM77ZENHZZF5QCTCE`, chave `3bd0224f-6493-46b2-bd99-6866250cd065`, única nova para `VM-2047` |

Nesta execução o proxy ainda não registrava a `Idempotency-Key`; o campo foi acrescentado em
seguida e aparece nos cenários D2 e D4. A reutilização da chave no "Tentar novamente" é coberta
pelo teste `sem conexão na abertura: tentar novamente reaproveita a mesma chave` do aplicativo e
observada diretamente em D2.

## D3 — resposta da conclusão perdida (proxy em `cortar-resposta` às 17:29:50)

Executado antes do D2 para aproveitar a validação aberta em D1. Um observador do log devolveu o
proxy a `normal` assim que a primeira resposta foi descartada.

| Hora | Fonte | Evento |
|---|---|---|
| 17:30:55,99 | proxy | `POST …/01M5ZGRC…/conclusao` → gateway `200` em 875 ms; resposta descartada |
| 20:30:55,35 Z | função | `POST …/conclusao` `200`, 0,532 s (**único** `POST` de conclusão) |
| 17:30:58,95 | proxy | `GET /v1/validacoes/01M5ZGRC…` (consulta de recuperação do aplicativo) → `200` em 2,938 s; descartado |
| 17:30:58,97 | proxy | modo `normal` (observador) |
| 17:31:00,45 | proxy | `GET` reenviado pelo aplicativo (+1 s) → `200` em 351 ms |
| — | tela | a conclusão apareceu sozinha, sem toque adicional |
| — | Firestore | `status=concluida`, `concluidaEm=20:30:55,475 Z`, leituras 25 / 13, `codigoRotacionado=true`, eventos `iniciada` e `concluida` publicados |

## D2 — rede lenta (proxy em `atrasar` 25 s às 17:32:48)

| Hora | Fonte | Evento |
|---|---|---|
| ~17:34:22 | aplicativo | 1º `POST /v1/validacoes`, chave `51e753e4-5756-4588-9d3c-9e5e42854ea2`; desiste em 20 s |
| ~17:34:43 | aplicativo | 2º `POST`, mesma chave (+1 s após o tempo esgotado) |
| ~17:35:05 | aplicativo | 3º `POST`, mesma chave (+2 s) |
| ~17:35:29 | aplicativo | 4º `POST`, mesma chave (+4 s) |
| 17:34:48 – 17:35:54 | proxy | os quatro `POST` encaminhados após 25 s (o aplicativo já havia desistido) → `201` (25,9 / 25,2 / 25,3 / 25,3 s) |
| 20:34:47,54 Z | função | `POST` `201` 0,574 s (cria a validação) |
| 20:35:08,46 Z; 20:35:30,48 Z; 20:35:54,53 Z | função | `POST` `201` 0,059 / 0,118 / 0,156 s (reprises idempotentes) |
| — | Firestore | **um** documento com a chave: `01M5ZH5697GSC8PGRBSE6TY00V` |
| — | tela | "Sem conexão" com "Tentar novamente" após ~1,5 min |
| 17:36:03 | proxy | modo `normal` |
| 17:38:20,59 | proxy | "Tentar novamente": `POST` com a mesma chave → `201` (mesma validação) |
| 17:38:45,04 | proxy | `POST …/01M5ZH56…/conclusao` → `200` em 809 ms |

O proxy não detectou a desistência do cliente (o corpo do `POST` só é lido depois do atraso, e o
servidor Go não monitora a conexão até lá); o pedido chegou atrasado ao serviço, como numa rede
lenta real. O resultado é uma demonstração mais forte da idempotência do que a prevista no
roteiro: quatro chegadas, uma validação.

## D4 — rede intermitente (proxy em `intermitente` às 17:39:01)

O modo `intermitente` alterna estritamente uma requisição cortada e uma normal.

**1ª execução (aplicativo da Sprint 7).**

| Hora | Fonte | Evento |
|---|---|---|
| 17:41:07,44 | proxy | `POST /v1/validacoes` cortado (chave `7bffa76c-89da-4b17-bcea-f810da224f97`) |
| 17:41:09,32 | proxy | reenvio automático → `201` em 851 ms; validação `01M5ZHGTH514RDPX9ETZ196XRN`; tela abriu em ~2 s |
| 17:41:15,63 | proxy | `POST …/conclusao` cortado |
| 17:41:15,97 | proxy | `GET` de recuperação → `200` (aberta) |
| — | tela | faixa "Sem conexão", formulário mantido |
| 17:44:20,33 / 17:44:20,77 | proxy | 2º toque: `POST` cortado, `GET` `200` (aberta); mesma faixa |

Achado: a conclusão não tinha reenvio automático. Com a alternância estrita, o par "`POST`
cortado, `GET` ok" se repetia a cada toque e o fluxo não completava. O controlador passou a
reenviar a conclusão (esperas de 1, 2 e 4 s) quando a consulta confirma a validação aberta, e a
tratar `validacao_ja_concluida` consultando e exibindo a conclusão.

**2ª execução (aplicativo ajustado, instalado às 17:45:56; alternância estrita).**

| Hora | Fonte | Evento |
|---|---|---|
| 17:48:07,81 | proxy | `POST /v1/validacoes` cortado (chave `66703dab-9c2f-4142-ad15-0d2e1c55a9d6`) |
| 17:48:09,57 | proxy | reenvio → `409 validacao_em_andamento` (706 ms); "Retomar" (CE-11) |
| 17:48:11,40 / 17:48:12,80 | proxy | `GET` cortado; `GET` reenviado → `200`; formulário de volta |
| 17:48:18,67 / :18,97 | proxy | "Concluir visita": `POST` cortado; `GET` `200` (aberta) |
| 17:48:20,00 / :20,47 | proxy | reenvio +1 s: `POST` cortado; `GET` `200` |
| 17:48:22,52 / :22,79 | proxy | reenvio +2 s: `POST` cortado; `GET` `200` |
| 17:48:26,90 / :27,20 | proxy | reenvio +4 s: `POST` cortado; `GET` `200` |
| — | tela | faixa "Sem conexão" após 8,5 s (4 envios, 4 consultas); nenhum `POST` chegou ao serviço |

**3ª execução (modo `cortar` com `proporcao` 0,5 às 17:49:20).**

| Hora | Fonte | Evento |
|---|---|---|
| 17:50:27,39 | proxy | "Concluir visita": `POST …/conclusao` sorteado como normal → `200` em 951 ms |
| 20:50:26,60 Z | função | `POST …/conclusao` `200`, 0,700 s (**único** `POST` de conclusão da validação) |
| — | Firestore | `01M5ZHGT…` `concluida` às 20:50:26,760 Z |
| — | tela | tela de conclusão |

Registro da função para a validação (UTC): `POST` `201` 20:41:08; `GET` 20:41:15 e 20:44:20;
`POST` `409` 20:48:09; `GET` 20:48:12, :18, :20, :22 e :27; `POST …/conclusao` `200` 20:50:26.

## E — provedor de autenticação indisponível

`dnsmasq` no computador (`192.168.1.141:53`) respondendo `NXDOMAIN` aos hosts do Firebase
Authentication; celular com IP estático, DNS 1 = computador, dados móveis desligados; APK normal
(gateway direto). Ordem executada: E2, E1, E3, com uma única execução do `dnsmasq`, porque a
sessão já estava iniciada.

### E2 — sessão iniciada com o provedor bloqueado

| Hora | Fonte | Evento |
|---|---|---|
| 17:54:17 | dnsmasq | iniciado; bloqueados `identitytoolkit.googleapis.com` e `securetoken.googleapis.com` |
| 18:03:55 | dnsmasq | primeira consulta do celular (`mtalk.google.com`) |
| 18:04:50 | dnsmasq | `query[A]`/`[AAAA] svv-dev-gateway-4mjof8z9.ue.gateway.dev` (encaminhada); **nenhuma** consulta a hosts do Firebase |
| 21:04:50,51 Z | função | `POST /v1/validacoes` `201`, 1,495 s (validação `01M5ZJW7MP88HQ4FGVTMC7KK4R`) |
| 21:05:02,30 Z | função | `POST …/conclusao` `200`, 0,380 s |
| — | tela | fluxo normal, leitura concluída |

### E1 — login com o provedor bloqueado

| Tentativa | Evento |
|---|---|
| 1ª (~18:08) | após "Sair", o login **funcionou**. `dnsmasq`: nenhuma consulta a `identitytoolkit`, uma a `www.googleapis.com` às 18:08:42 (encaminhada). O SDK Android do Firebase Auth entra por `www.googleapis.com/identitytoolkit/v3`; host acrescentado ao bloqueio |
| 2ª (~18:11) | `dnsmasq` reiniciado às 18:10:27 com os três hosts; o login **funcionou** de novo sem nenhuma consulta a `www.googleapis.com`: cache do resolvedor do Android (TTL da resolução das 18:08:42) e conexão já aberta |
| 3ª (18:12) | Wi-Fi desligado e ligado (zera o cache e as conexões); `dnsmasq`: `query[A] www.googleapis.com from 192.168.1.142` → `config www.googleapis.com is NXDOMAIN` às 18:12:16; tela: **"Autenticação indisponível no momento. Tente em instantes"** |

### E3 — restabelecimento

`dnsmasq` parado e DNS do celular de volta ao automático (~18:14); o login com o operador1 voltou
a funcionar.

## Critérios

| Critério | Resultado |
|---|---|
| D1: reenvios com a mesma chave e `201` sem duplicar ao voltar a rede | ✅ |
| D2: tempo esgotado em 20 s, reenvios e uma única validação | ✅ |
| D3: conclusão reconhecida pela consulta, sem reenviar o `POST` | ✅ |
| D4: fluxo completo sob intermitência, sem duplicar | ✅ após o ajuste do reenvio da conclusão (com cortes aleatórios); sob alternância estrita o aplicativo desiste com mensagem após 4 envios, sem duplicar |
| E1: "Autenticação indisponível" no login | ✅ após incluir `www.googleapis.com` no bloqueio e zerar o cache do resolvedor |
| E2: validação e conclusão com o provedor bloqueado | ✅ |
| E3: login restabelecido | ✅ |
