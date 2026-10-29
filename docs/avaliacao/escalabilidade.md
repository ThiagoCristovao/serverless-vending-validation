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

*(seções preenchidas com os relatórios de evidências)*

## 3. Cold start e regime contínuo

*(a preencher)*

## 4. Síntese por requisito

*(a preencher)*

## 5. Observações e limitações

*(a preencher)*

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
