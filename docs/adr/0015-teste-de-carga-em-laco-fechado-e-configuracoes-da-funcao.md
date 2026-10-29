# ADR-0015 — Teste de carga em laço fechado com operadores virtuais e comparação de configurações da função

- **Status:** aceita
- **Data:** 2026-10-29
- **Sprint:** 10

## Contexto

A avaliação de escalabilidade (TCC 1, Seção 3.4.7; RNF-05 e RNF-06) pede testes de carga com
volumes simultâneos crescentes de operações de validação, medindo tempo médio e percentis
p50/p95/p99, o número de instâncias alocadas pela plataforma e a estabilidade da latência fim a
fim, além do cold start separado do regime contínuo. O RNF-05 fixa o alvo "p95 da latência do
serviço, medida no gateway, ≤ 500 ms com 10, 50 e 100 operadores simultâneos". A função roda
hoje com os padrões das Cloud Run functions para 256 MiB: 1/6 de vCPU, uma requisição por
instância e, por controle de custo, no máximo 10 instâncias. O Cloud Scheduler chama a
reconciliação de publicações a cada 5 min (ADR-0012), o que mantém uma instância aquecida o tempo
todo e torna o cold start raro em uso normal.

## Decisão

1. **Modelo de carga em laço fechado.** Cada "operador simultâneo" é um operador virtual com
   conta própria no Firebase Authentication (`carga-NNN@exemplo.invalid`), documento próprio em
   `operadores/` e máquina própria (`VM-CARGA-NNN`), que repete o fluxo real do aplicativo
   (`POST /validacoes` com `Idempotency-Key` nova, `POST /validacoes/{id}/conclusao`, pausa de
   1 s) durante uma janela fixa, após uma rampa de 30 s. A concorrência é dada pelo número de
   operadores, não por uma taxa de requisições: é o que o requisito descreve e evita que o gerador
   dispare mais do que N operadores conseguiriam.
2. **Medição em duas pontas.** O gerador registra a latência de cada requisição no cliente; após
   a janela, lê do Monitoring as latências do gateway (`serviceruntime.googleapis.com/api/
   request_latencies`, total, back-end e sobrecarga), da função (`run.googleapis.com/
   request_latencies`), as instâncias ativas por minuto e as partidas de instância. O critério do
   RNF-05 usa a métrica do gateway, como o requisito pede; o critério de regime contínuo do RNF-06
   usa a métrica da função com 10 operadores.
3. **Configurações comparadas.** A (atual: concorrência 1, 1/6 vCPU, máx. 10 instâncias) e C
   (concorrência 80, 1 vCPU, máx. 100 instâncias), cada uma com 10, 50 e 100 operadores. C
   explora a concorrência por instância que um serviço em Go sustenta; a variável de ambiente do
   Terraform volta ao padrão após os testes. As três variáveis novas (`funcao_concorrencia`,
   `funcao_cpu`, `funcao_max_instancias`) ficam com valores nulos/atuais por padrão.
4. **Cold start por inatividade real.** O cenário `frio` pausa o job de reconciliação, espera
   20 min de inatividade, faz uma validação (primeira requisição fria, conclusão quente) e repete
   seis vezes; depois retoma o job. É executado com `min_instancias` 0 e 1 (CE-18), lendo também
   a latência da função nos logs e `startup_latencies` do Monitoring.
5. **Ferramenta própria em Go** (`sistema-central-simulado/cmd/carga`), ao lado de
   `cmd/experimento`, reutilizando o pacote `laboratorio` (tokens, adesivos, gateway, Monitoring,
   Logging, relatório). A preparação (`make carga-preparar`) cria as contas pela API
   administrativa do Identity Toolkit e semeia o Firestore com `semear-firestore -lote`.

## Alternativas consideradas

- **Ferramenta de carga genérica (k6, hey, vegeta).** Mede bem HTTP puro, mas o fluxo exige
  adesivos assinados, chaves de idempotência novas, tokens por operador e o encadeamento
  abertura → conclusão; a ferramenta própria reaproveita o laboratório já existente e grava o
  relatório no mesmo formato dos experimentos da Sprint 9.
- **Um único operador para todos os fluxos.** Mais simples, mas o gateway validaria sempre o
  mesmo JWT e o serviço leria sempre o mesmo documento de operador; descartada por fidelidade ao
  requisito.
- **Modelo aberto (taxa fixa de requisições).** Útil para achar o ponto de saturação, mas não
  corresponde a "N operadores simultâneos" e pode gerar filas artificiais; fica como extensão.
- **Cold start forçado por nova revisão.** Rápido, mas mede a partida após implantação e não após
  inatividade; descartada.

## Consequências

- Cem contas de teste e cem máquinas de carga passam a existir no projeto `svv-dev`
  (`carga-lote.json` e `carga.senha` ficam em `chaves/`, fora do versionamento).
- Os testes de carga redeployam a função ao trocar de configuração; entre configurações, a
  primeira requisição é fria. Os relatórios registram a janela completa, inclusive a rampa.
- Pausar o agendador durante o cold start atrasa a reconciliação de publicações pendentes por até
  duas horas; aceitável no ambiente de desenvolvimento.
- Custo: algumas centenas de requisições por minuto por alguns minutos; dentro do orçamento do
  projeto.

## Referências

- `docs/requisitos.md` (RNF-05, RNF-06); `docs/arquitetura.md` (CE-18); ADR-0012, ADR-0013,
  ADR-0014; `docs/avaliacao/escalabilidade.md`.
