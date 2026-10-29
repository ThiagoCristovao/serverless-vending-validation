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
| Cold start | cenário `frio`: o job de reconciliação do Cloud Scheduler é pausado (ele chama a função a cada 5 min e mantém uma instância aquecida); após 20 min de inatividade, uma validação (primeira requisição fria, conclusão quente); seis amostras; latência da função lida nos logs e partidas no Monitoring |
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

*(a preencher)*

## 4. Síntese por requisito

| Requisito / cenário | Resultado | Onde |
|---|---|---|
| RNF-05 — p95 no gateway ≤ 500 ms com 10, 50 e 100 operadores, com escala automática | **atendido na configuração C** (249, 247 e 452 ms; 1 a 2 instâncias) e **parcialmente na A** (304 e 256 ms com 10 e 50; 1 834 ms com 100 por saturação do limite de 10 instâncias com uma requisição por vez). A escala automática foi observada nas duas (9 a 10 instâncias em A; 1 a 2 em C) | 2.1, 2.2, 2.3 |
| RNF-06 — regime contínuo: p50 ≤ 200 ms e p95 ≤ 500 ms | **atendido**: função p50 136 ms e p95 250 ms (A-10); p50 115 ms e p95 161 ms (C-10) | 2.1, 2.2 |
| RNF-06 / CE-18 — cold start ≤ 1,5 s | *(seção 3, a preencher)* | 3 |
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
