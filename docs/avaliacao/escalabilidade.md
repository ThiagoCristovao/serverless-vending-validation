# Relatório de escalabilidade (Sprint 10)

Avaliação técnica do terceiro atributo de qualidade do trabalho (TCC 1, Seção 3.4.7):
**escalabilidade**, com os alvos dos requisitos RNF-05 (p95 da latência do serviço, medida no
gateway, ≤ 500 ms com 10, 50 e 100 operadores simultâneos, com escala automática de instâncias)
e RNF-06 (cold start ≤ 1,5 s; regime contínuo com p50 ≤ 200 ms e p95 ≤ 500 ms). O desenho do
teste está no [ADR-0015](../adr/0015-teste-de-carga-em-laco-fechado-e-configuracoes-da-funcao.md);
os relatórios brutos gerados pela ferramenta ficam em [evidencias/](evidencias/). A comparação
qualitativa com a abordagem monolítica está em [analise-comparativa.md](analise-comparativa.md).

## 1. Método

| Elemento | Como |
|---|---|
| Carga | N operadores virtuais em laço fechado, cada um com conta própria no Firebase Authentication, documento em `operadores/` e máquina própria (`VM-CARGA-NNN`), repetindo o fluxo real: `POST /validacoes` (chave de idempotência nova) → `POST /validacoes/{id}/conclusao` → pausa de 1 s. Rampa de 30 s; 2 min em regime |
| Medição no cliente | latência de cada requisição no gerador (no mesmo computador da Sprint 9, pela internet doméstica); média, p50, p95, p99, máximo, erros e vazão; linha do tempo em janelas de 10 s |
| Medição no provedor | Cloud Monitoring: latência do gateway (`serviceruntime.googleapis.com/api/request_latencies`, total, parcela do back-end e sobrecarga), latência da função (`run.googleapis.com/request_latencies`), instâncias ativas por minuto, requisições e partidas de instância (`startup_latencies`) na janela do teste |
| Cold start | cenário `frio`: o job de reconciliação do Cloud Scheduler é pausado (ele chama a função a cada 5 min e mantém uma instância aquecida); após 20 min de inatividade, uma validação (primeira requisição fria, conclusão quente); seis amostras; cada requisição medida no cliente, no gateway (log de acesso `endpoints_log`, com o agente que atendeu) e na função (log de requisição do Cloud Run); partidas de instância no Monitoring |
| Ferramenta | `sistema-central-simulado/cmd/carga` (`make carga-preparar`, `make carga`, `make carga-frio`) |

Configurações da função comparadas (variáveis `funcao_*` do Terraform, ADR-0015):

| Configuração | Concorrência por instância | vCPU | Memória | Máx. de instâncias | Observação |
|---|---|---|---|---|---|
| **A** (atual) | 1 | 1/6 | 256 MiB | 10 | padrões da plataforma para 256 MiB; limite de 10 por controle de custo |
| **C** (concorrente) | 80 | 1 | 256 MiB | 100 | concorrência que um serviço em Go sustenta; exige 1 vCPU |

Critérios verificados pela ferramenta em cada execução: nenhuma requisição com erro em regime;
RNF-05 pelo p95 do gateway; RNF-06 (regime contínuo) pelo p50 e p95 da função com 10
operadores; cold start pela mediana e pelo máximo das primeiras requisições.

## 2. Resultados sob carga

Execuções de 29/10/2026 contra `svv-dev`; cada uma com rampa de 30 s e 2 min em regime
([evidências](evidencias/)). "Cliente" é a latência medida pelo gerador, que inclui a ida e
volta pela internet até `us-east1`; "gateway" e "função" são as métricas do Monitoring na
janela completa (rampa incluída). Nenhuma requisição falhou em nenhuma execução.

### 2.1 Configuração A (concorrência 1, 1/6 vCPU, máx. 10 instâncias)

| Operadores | Vazão em regime | Cliente p50 / p95 / p99 | Gateway p50 / p95 / p99 | Função p50 / p95 / p99 | Instâncias (máx.) | Partidas (média) | RNF-05 |
|---|---|---|---|---|---|---|---|
| 10 ([A-10](evidencias/carga-A-10.md)) | 12,5 req/s | 278 / 379 / 442 ms | 176 / **304** / 504 ms | 136 / 250 / 386 ms | 9 | 6 (249 ms) | ✅ |
| 50 ([A-50](evidencias/carga-A-50.md)) | 63,5 req/s | 275 / 365 / 418 ms | 157 / **256** / 410 ms | 121 / 190 / 248 ms | 10 | 3 (69 ms) | ✅ |
| 100 ([A-100](evidencias/carga-A-100.md)) | 78,3 req/s | 624 / 1592 / 1880 ms | 386 / **1834** / 2046 ms | 119 / 188 / 235 ms | 10 | 1 (49 ms) | ❌ |

Leitura. Com 10 e 50 operadores a plataforma alocou instâncias conforme a demanda (9 e 10) e o
p95 no gateway ficou em 304 e 256 ms. Com 100 operadores a demanda (cerca de 128 req/s, dado o
ciclo de cada operador) ultrapassou a capacidade do limite de 10 instâncias com uma requisição
por vez (cerca de 80 req/s com 120 ms por requisição): a vazão saturou em 78 req/s e as
requisições passaram a esperar por uma instância livre. A função em si continuou rápida (p95
188 ms, o menor das três execuções): a fila se forma antes dela, no roteamento do Cloud Run, e
aparece na métrica do gateway (p95 1 834 ms) e no cliente (p95 1 592 ms), não na da função. A
linha do tempo mostra o p95 saltando de 437 ms para 1 434 ms entre 10 e 30 s da rampa, quando o
décimo operador deixou de ter instância própria, e estabilizando entre 1,3 e 1,9 s pelo resto da
janela, sem erros nem tempo esgotado: o limite protege o custo, mas não atende ao RNF-05 com 100
operadores. A sobrecarga própria do gateway foi de 1 ms no p95 em todas as execuções.

### 2.2 Configuração C (concorrência 80, 1 vCPU, máx. 100 instâncias)

| Operadores | Vazão em regime | Cliente p50 / p95 / p99 | Gateway p50 / p95 / p99 | Função p50 / p95 / p99 | Instâncias (máx. em regime) | Partidas (média) | RNF-05 |
|---|---|---|---|---|---|---|---|
| 10 ([C-10](evidencias/carga-C-10.md)) | 11,7 req/s | 353 / 387 / 423 ms | 119 / **249** / 359 ms | 115 / 161 / 245 ms | 2 | 2 (363 ms) | ✅ |
| 50 ([C-50](evidencias/carga-C-50.md)) | 57,6 req/s | 363 / 407 / 454 ms | 120 / **247** / 260 ms | 117 / 161 / 215 ms | 1 | 0 | ✅ |
| 100 ([C-100](evidencias/carga-C-100.md)) | 120,6 req/s | 305 / 467 / 543 ms | 197 / **452** / 510 ms | 147 / 317 / 393 ms | 2 | 2 (444 ms) | ✅ |

Leitura. Com concorrência por instância, uma única instância de 1 vCPU atendeu 50 operadores
(57,6 req/s) e duas atenderam 100 (120,6 req/s, a demanda integral), com p95 no gateway de 247 e
452 ms. A função ficou mais lenta sob 100 operadores (p95 317 ms contra 161 ms com 50), sinal
de disputa de CPU dentro da instância antes de a segunda subir, mas dentro do alvo. A linha do
tempo mostra o p95 no cliente passando de 586 ms entre 30 e 40 s da rampa (uma instância
saturando) para a faixa de 440–510 ms depois que a segunda instância entrou. As 9 instâncias do
primeiro minuto da C-10 são da revisão anterior, ainda sendo trocada. Em regime, C usou de 1 a 2
instâncias onde A usou 9 a 10.

### 2.3 Comparação

| Operadores | p95 no gateway, A | p95 no gateway, C | Vazão A | Vazão C | Instâncias A | Instâncias C |
|---|---|---|---|---|---|---|
| 10 | 304 ms | 249 ms | 12,5 req/s | 11,7 req/s | 9 | 2 |
| 50 | 256 ms | 247 ms | 63,5 req/s | 57,6 req/s | 10 | 1 |
| 100 | 1 834 ms | 452 ms | 78,3 req/s (saturada) | 120,6 req/s | 10 | 2 |

A escala automática funcionou nas duas configurações: em A a plataforma subiu instâncias até o
limite de 10; em C, uma ou duas instâncias bastaram. A diferença com 100 operadores não vem da
função (p95 de 188 ms em A e 317 ms em C), mas do limite de instâncias combinado com uma
requisição por instância: em A as requisições esperam por uma instância livre e a espera aparece
no gateway e no cliente. Elevar o limite de A para 100 instâncias também atenderia ao RNF-05,
com dez vezes mais instâncias (e partidas) do que C. O RNF-05 é atendido em todas as cargas pela
configuração C e nas cargas de 10 e 50 operadores pela configuração A.

## 3. Cold start e regime contínuo

Execuções de 30/10/2026 ([cold-start-min0.md](evidencias/cold-start-min0.md) e
[cold-start-min1.md](evidencias/cold-start-min1.md)), configuração A com `min_instancias` 0 e 1.
Cada amostra, após 20 min sem tráfego e com o job do Scheduler pausado, abre uma validação
(1ª requisição, fria) e a conclui (2ª, quente). Cada requisição tem três medidas: no cliente
(ida e volta pela internet), no gateway (latência cronometrada pelo próprio gateway no log de
acesso `endpoints_log`, com o agente que atendeu) e na função (log de requisição do Cloud Run,
que inclui a partida do contêiner). "Instâncias antes" é o `instance_count` do minuto anterior.

### 3.1 `min_instancias` 0

| Amostra | 1ª requisição: cliente / gateway (agente) / função | 2ª requisição: cliente / gateway (agente) / função |
|---|---|---|
| 1 | 4,995 s / 1,758 s (ESPv2) / 1,457 s | 5,802 s / 3,520 s (cloudesf) / 0,440 s |
| 2 | 6,024 s / 2,807 s (ESPv2) / 2,526 s | 0,469 s / 0,323 s (ESPv2) / 0,304 s |
| 3 | 5,752 s / 2,816 s (ESPv2) / 2,535 s | 4,741 s / 3,428 s (cloudesf) / 0,363 s |
| 4 | 7,205 s / 5,962 s (cloudesf) / 2,654 s | 0,603 s / 0,424 s (cloudesf) / 0,391 s |
| 5 | 7,262 s / 5,981 s (cloudesf) / 2,552 s | 0,761 s / 0,586 s (cloudesf) / 0,568 s |
| 6 | 6,972 s / 5,852 s (cloudesf) / 2,478 s | 3,180 s / 0,303 s (ESPv2) / 0,278 s |

Primeira requisição no cliente: mediana 6,0 s, média 6,4 s, máximo 7,3 s. Instâncias antes: 0 em
todas as amostras; uma partida de contêiner por amostra (302 a 681 ms). **Critério não atendido**
(alvo ≤ 1,5 s).

**Onde o tempo fica.** Decompondo cada requisição fria:

- **Função: 1,5 a 2,7 s** (25 % a 45 % do total). É a partida do contêiner (0,3 a 0,7 s) mais a
  montagem preguiçosa das dependências na primeira requisição, com 1/6 de vCPU: clientes do
  Firestore e do Pub/Sub, token de acesso do servidor de metadados, chaves públicas para validar o
  ID token do Firebase e a primeira publicação no tópico. Na segunda requisição a função responde em
  0,28 a 0,57 s.
- **Gateway: 3,0 a 3,5 s**, a maior parcela, e ela aparece de duas formas conforme o agente que
  atendeu. Com o ESPv2 (amostras 1 a 3) o gateway cronometrou só 0,3 s além da função; os outros
  2,9 a 3,2 s ficam **antes** de ele começar a contar, isto é, na partida da própria instância do
  gateway, um serviço gerenciado sem instâncias mínimas configuráveis. Com o agente `cloudesf`
  (amostras 4 a 6), 3,1 a 3,3 s ficam **dentro** da medição do gateway (gateway menos função) e
  1,1 a 1,3 s antes dela.
- **Rede: 0,15 a 0,2 s** (cliente menos gateway nas requisições quentes das amostras 2, 4 e 5).

A segunda requisição, com a função já quente (0,28 a 0,44 s), ainda levou 3,2 a 5,8 s em três das
seis amostras (1, 3 e 6): foi atendida por **outra instância do gateway**, com agente diferente
do da primeira e também fria. Dois agentes se alternaram no mesmo dia (`ESPv2/2.56.0` e
`cloudesf_20261012_00_RC02`), sinal de uma migração interna do produto gerenciado, fora do
controle do projeto.

### 3.2 `min_instancias` 1

| Amostra | 1ª requisição: cliente / gateway (agente) / função | 2ª requisição: cliente / gateway (agente) / função |
|---|---|---|
| 1 | 8,040 s / 6,631 s (cloudesf) / 3,557 s | 0,505 s / 0,366 s (cloudesf) / 0,353 s |
| 2 | 6,182 s / 4,665 s (cloudesf) / 1,608 s | 0,580 s / 0,423 s (cloudesf) / 0,396 s |
| 3 | 5,132 s / 1,871 s (ESPv2) / 1,828 s | 4,619 s / 3,484 s (cloudesf) / 0,392 s |
| 4 | 6,070 s / 4,728 s (cloudesf) / 1,639 s | 0,471 s / 0,329 s (cloudesf) / 0,316 s |
| 5 | 6,389 s / 4,927 s (cloudesf) / 1,877 s | 3,398 s / 0,379 s (ESPv2) / 0,357 s |
| 6 | 6,362 s / 4,955 s (cloudesf) / 1,864 s | 3,462 s / 0,362 s (ESPv2) / 0,323 s |

Primeira requisição no cliente: mediana 6,2 s, média 6,4 s, máximo 8,0 s. Instâncias antes: 1 em
todas as amostras; nenhuma partida de contêiner. **Critério não atendido** (alvo ≤ 1,5 s).

O que a instância mínima mudou:

- **A função deixou de partir, mas não ficou rápida.** O log do Cloud Run mostra a mesma instância
  nas seis amostras e a mensagem "serviço montado" só na primeira (3,56 s), quando a instância
  recém-criada pela implantação montou as dependências. Nas cinco seguintes a instância já estava
  montada e, mesmo assim, a primeira requisição levou 1,6 a 1,9 s, contra 0,32 a 0,40 s da
  segunda: uma penalidade de ociosidade de 1,3 a 1,5 s em uma instância "quente". A causa provável
  é o restabelecimento das conexões ociosas (canais gRPC do Firestore e do Pub/Sub, token de
  acesso, chaves públicas do ID token) com 1/6 de vCPU e CPU alocada só durante requisições;
  confirmar exigiria rastreamento (Cloud Trace), fora do escopo.
- **O gateway não mudou.** 3,0 a 3,3 s com o ESPv2 (antes de ele cronometrar) e 4,4 a 4,6 s com
  o `cloudesf` (3,05 a 3,1 s dentro da medição mais 1,3 a 1,5 s antes dela). Três segundas
  requisições (amostras 3, 5 e 6) voltaram a cair em outra instância fria do gateway.

No cliente a mediana ficou igual (6,0 s → 6,2 s): a instância mínima reduziu a parcela da função
de cerca de 2,5 s para 1,8 s e não toca na parcela dominante.

### 3.3 Cold start e regime contínuo

| Medida | Regime contínuo (A-10, seção 2.1) | Primeira requisição após 20 min de inatividade |
|---|---|---|
| Cliente | p95 379 ms | mediana 6,0 s (`min_instancias` 0) e 6,2 s (`min_instancias` 1) |
| Gateway | p95 304 ms | 3,0 a 4,6 s por requisição fria, em qualquer das duas configurações |
| Função | p50 136 ms, p95 250 ms | 1,5 a 2,7 s (`min_instancias` 0); 1,6 a 1,9 s (`min_instancias` 1) |

A primeira requisição após inatividade custa cerca de vinte vezes o p95 do gateway em regime.
O alvo do RNF-06 (≤ 1,5 s) foi escrito supondo que a função fosse o único componente a partir a
frio (CE-18, "cold start da função"); a medição mostra que o API Gateway, serviço gerenciado sem
instâncias mínimas configuráveis, responde pela parcela maior e fora do controle do projeto.
Mesmo a parcela da função só caberia no alvo se a instância, além de existir, mantivesse suas
conexões ativas.

Em uso normal o job do Scheduler mantém a função aquecida (uma instância sempre alocada nas 24 h
observadas no Monitoring), mas não o gateway, porque chama a função diretamente: a primeira
validação de um operador depois de um intervalo sem uso da frota pagaria a partida do gateway, e
o aplicativo a cobre com o indicador de progresso (CE-18). Três caminhos de mitigação ficam para
decisão de operação, a registrar em ADR próprio: tráfego periódico que atravesse o gateway (o
próprio job do Scheduler chamando uma rota leve pela URL pública, custo desprezível), chamada
direta do aplicativo ao serviço Cloud Run com validação do token na função (dispensa a partida do
gateway e o contrato OpenAPI imposto por ele) ou revisão do alvo do RNF-06 para a primeira
requisição após inatividade, mantendo o alvo de regime.

## 4. Síntese por requisito

| Requisito / cenário | Resultado | Onde |
|---|---|---|
| RNF-05 — p95 no gateway ≤ 500 ms com 10, 50 e 100 operadores, com escala automática | **atendido na configuração C** (249, 247 e 452 ms; 1 a 2 instâncias) e **parcialmente na A** (304 e 256 ms com 10 e 50; 1 834 ms com 100 por saturação do limite de 10 instâncias com uma requisição por vez). A escala automática foi observada nas duas (9 a 10 instâncias em A; 1 a 2 em C) | 2.1, 2.2, 2.3 |
| RNF-06 — regime contínuo: p50 ≤ 200 ms e p95 ≤ 500 ms | **atendido**: função p50 136 ms e p95 250 ms (A-10); p50 115 ms e p95 161 ms (C-10) | 2.1, 2.2 |
| RNF-06 / CE-18 — cold start ≤ 1,5 s | **não atendido**: primeira requisição após 20 min de inatividade com mediana de 6,0 s (`min_instancias` 0) e 6,2 s (`min_instancias` 1). A função responde por 1,5 a 2,7 s e 1,6 a 1,9 s; os 3 a 4,6 s restantes são a partida do API Gateway, gerenciado e sem instâncias mínimas; a instância mínima elimina a partida do contêiner, não o problema | 3 |
| Estabilidade da latência fim a fim conforme a carga cresce | em C o p95 no cliente foi de 387, 407 e 467 ms para 10, 50 e 100 operadores; em A, 379, 365 e 1 592 ms | 2.1, 2.2 |
| Nenhuma requisição com erro sob carga | **atendido** em todas as seis execuções (41 295 requisições em regime) | evidências |

## 5. Observações e limitações

- **O limite de instâncias é uma decisão de custo com efeito direto na latência.** O p95 de
  1,8 s em A-100 não é lentidão da função (p95 188 ms), e sim espera por instância livre; o
  Cloud Run enfileira em vez de rejeitar, por isso não houve erro nem tempo esgotado. Em um
  ambiente real o limite seria dimensionado pela frota de operadores, e o painel de observabilidade
  já mostra instâncias e latência para ajustá-lo.
- **Concorrência por instância muda a economia da escala.** Com 80 requisições por instância e
  1 vCPU, uma ou duas instâncias atenderam 100 operadores; com uma requisição por instância,
  foram necessárias dez vezes mais instâncias para menos vazão. Para um serviço em Go que passa a
  maior parte do tempo esperando o Firestore, a concorrência é o ajuste mais barato.
- **A medição no cliente inclui a rede.** O gerador rodou em um computador doméstico no Paraná
  contra `us-east1`: a diferença entre o p50 do cliente (275 a 363 ms) e o do gateway (120 a
  197 ms) é a ida e volta pela internet, parecida com a de um celular em campo. O critério do
  RNF-05 usa a métrica do gateway, como o requisito define.
- **Partidas de instância durante a carga são curtas.** As partidas registradas pelo Monitoring
  durante as cargas tiveram média de 49 a 444 ms (`startup_latencies` mede o contêiner, não a
  primeira requisição); a seção 3 mede a primeira requisição fria de ponta a ponta.
- **O cold start medido é do gateway, não da função.** Dos 6 s da primeira requisição após
  inatividade, 3 a 4,6 s ficam na partida da instância do API Gateway, que o projeto não
  configura; `min_instancias = 1` na função não muda a mediana. O log de acesso do gateway
  (`endpoints_log`) foi decisivo para a atribuição: a latência que ele cronometra, comparada com a
  do cliente e com a do Cloud Run, separa as três parcelas. O máximo da métrica
  `api/request_latencies` não serve para isso, porque devolve limites de faixa do histograma.
- **Instância mínima não é instância quente.** Com a instância mantida e já montada, a primeira
  requisição após 20 min ainda levou 1,6 a 1,9 s, cinco vezes a segunda. Manter uma instância
  alocada evita a partida do contêiner; manter conexões e CPU ativas exige tráfego.
- **O produto gerenciado muda sob os pés.** Dois agentes do gateway (`ESPv2/2.56.0` e
  `cloudesf_20261012_00_RC02`) se alternaram no mesmo dia, com perfis de partida diferentes, e
  metade das segundas requisições caiu em uma instância fria do gateway mesmo com a função quente.
  Medições de cold start em serviços gerenciados precisam registrar quem atendeu.
- **Métricas do provedor chegam com atraso.** As do Cloud Run levam 3 a 4 min para ficar
  disponíveis, as do gateway cerca de 2; a ferramenta espera 4 min antes de consultá-las, e
  a contagem de requisições da função na janela ainda pode vir incompleta.
- **Modelo de carga.** Laço fechado com pausa de 1 s: a demanda cresce com o número de
  operadores (cerca de 1,3 req/s por operador) e a saturação se manifesta como latência, não
  como erro. Um modelo aberto (taxa fixa) mostraria o ponto de rejeição; fica como extensão.
- **Reverter a configuração exige valores explícitos.** O provedor mantém CPU e concorrência
  quando o Terraform omite os atributos; os padrões do ambiente dev passaram a ser explícitos
  para que aplicar sem variáveis volte à configuração A (ADR-0015).

## 6. Como reproduzir

```bash
make carga-preparar OPERADORES=100
make carga OPERADORES=10 CONFIGURACAO=A && make carga OPERADORES=50 CONFIGURACAO=A && make carga OPERADORES=100 CONFIGURACAO=A
make infra-dev-apply TF_VAR_funcao_concorrencia=80 TF_VAR_funcao_cpu=1 TF_VAR_funcao_max_instancias=100
make carga OPERADORES=10 CONFIGURACAO=C && make carga OPERADORES=50 CONFIGURACAO=C && make carga OPERADORES=100 CONFIGURACAO=C
make infra-dev-apply                                  # volta à configuração A
make carga-frio CONFIGURACAO=min0                     # ~2 h
make infra-dev-apply TF_VAR_funcao_min_instancias=1 && make carga-frio CONFIGURACAO=min1 && make infra-dev-apply
```

Cada medição de cold start leva cerca de 2 h; convém rodá-la fora da sessão do terminal (por
exemplo, com `systemd-run --user`), porque a ferramenta retoma o job do Scheduler ao terminar e
um cancelamento no meio exige retomá-lo manualmente.
