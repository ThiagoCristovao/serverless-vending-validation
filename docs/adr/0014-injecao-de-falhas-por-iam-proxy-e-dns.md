# ADR-0014 — Injeção de falhas: pausa do consumidor, revogação temporária de IAM, proxy de rede e bloqueio DNS

- **Status:** aceita
- **Data:** 2026-10-27
- **Sprint:** 9

## Contexto

A avaliação de resiliência e independência (TCC 1, Seção 3.4.7) exige induzir, no ambiente de
desenvolvimento, a indisponibilidade do consumidor de mensagens, do sistema central, de
dependências gerenciadas (Firestore, Pub/Sub), da rede do dispositivo e do provedor de
autenticação, e registrar o comportamento com as ferramentas de observabilidade do provedor. Os
serviços gerenciados não oferecem um "interruptor de indisponibilidade"; o dispositivo é um
celular sem root; a autenticação é feita pelo SDK do Firebase diretamente contra o Google.

## Decisão

| Falha | Mecanismo | Reversão |
|---|---|---|
| Sistema central / consumidor indisponível (CE-15) | pausar o consumidor simulado pela API administrativa ou em processo (`experimento -cenario central-indisponivel`) | retomar o consumo |
| Pub/Sub indisponível ao publicar (CE-14) | remover temporariamente `roles/pubsub.publisher` da conta da função no tópico (`experimento -cenario publicacao-indisponivel`) | o próprio comando restaura o vínculo ao final (ou `-cenario restaurar`) |
| Firestore indisponível (CE-13) | remover temporariamente `roles/datastore.user` da conta da função no projeto (`experimento -cenario firestore-indisponivel`) | idem |
| Rede intermitente no dispositivo (CE-01, CE-02, CE-03) | proxy HTTP local (`ferramentas/injecao-falhas/cmd/proxy-rede`) entre o aplicativo e o gateway, com modos cortar, atrasar, 503, cortar-resposta e intermitente; APK de depuração aponta para ele | APK normal |
| Provedor de autenticação indisponível (CE-05) | `dnsmasq` local respondendo NXDOMAIN para `www.googleapis.com`, `identitytoolkit.googleapis.com` e `securetoken.googleapis.com` (hosts do Firebase Authentication usados pelo SDK Android para entrar e renovar o token), encaminhando o resto | desfazer o DNS estático no celular |

As evidências são coletadas por API (Cloud Monitoring, Cloud Logging, Firestore), pelo registro
do consumidor e pelo log do `dnsmasq`, e consolidadas em `docs/avaliacao/resiliencia.md`.

## Alternativas consideradas

- **Desabilitar APIs do projeto** (`gcloud services disable`): rejeitada, por afetar recursos de
  forma ampla e, no caso do Identity Platform, irreversível.
- **Simular falhas dentro do código do serviço** (flags de teste): rejeitada; não exercita as
  dependências reais nem o comportamento de propagação e recuperação do provedor.
- **Bloquear os hosts do Firebase por firewall**: inviável sem root no celular e sem isolar os IPs
  do Google, que são compartilhados com o gateway.
- **Emulador Android com DNS próprio**: viável, mas foge do critério de dispositivo físico das
  Sprints 6 e 7.

## Consequências

- A revogação de IAM leva alguns segundos a minutos para propagar; os experimentos sondam o
  serviço até observar o efeito e registram esse tempo. Um experimento interrompido pode deixar
  o vínculo removido: `make experimento CENARIO=restaurar` repara; `terraform apply` também.
- O proxy exige tráfego HTTP em claro entre o celular e o computador; por isso a permissão
  `usesCleartextTraffic` fica apenas no manifesto de depuração.
- Os cenários do dispositivo dependem do celular e de passos manuais (DNS estático, modo do
  proxy); ficam roteirizados em `ferramentas/injecao-falhas/README.md`.
