# ADR-0016 — Cold start dominado pelo gateway: alvo do RNF-06 revisado e mitigação adiada

- **Status:** aceita
- **Data:** 2026-10-30
- **Sprint:** 10

## Contexto

O RNF-06 fixava "cold start (primeira requisição após inatividade) ≤ 1,5 s", escrito na Sprint 1
supondo que a função fosse o único componente a partir a frio (CE-18, "cold start da função").
A medição da Sprint 10 (`docs/avaliacao/escalabilidade.md`, seção 3; seis amostras por
configuração, 20 min de inatividade, job do Scheduler pausado) mostrou outra coisa:

- a primeira requisição levou 6,0 s de mediana com `min_instancias = 0` e 6,2 s com
  `min_instancias = 1` (máximo 8,0 s);
- a função responde por 1,5 a 2,7 s (partida do contêiner mais montagem das dependências com 1/6
  de vCPU) e, com a instância mantida e já montada, ainda por 1,6 a 1,9 s (penalidade de
  ociosidade: conexões e CPU paradas);
- os 3,0 a 4,6 s restantes são a partida da instância do API Gateway, serviço gerenciado sem
  instâncias mínimas configuráveis, cujo agente mudou duas vezes no mesmo dia
  (`ESPv2/2.56.0` e `cloudesf_20261012_00_RC02`). Metade das segundas requisições, um segundo
  depois da primeira, caiu em outra instância fria do gateway mesmo com a função quente.

Em uso normal o job de reconciliação mantém a função aquecida (ADR-0012), mas chama a função
diretamente e não atravessa o gateway. O aplicativo já trata a espera com indicador de progresso e
tem tempo limite de resposta de 20 s por tentativa, com retentativa só em falha de rede
(ADR-0009): uma primeira requisição de 6 a 8 s não gera erro nem reenvio, só espera visível.

## Decisão

1. **O alvo de 1,5 s não é atendido e fica registrado assim.** O relatório de escalabilidade
   e a síntese por requisito mantêm o resultado como "não atendido" para o alvo original, com a
   atribuição das parcelas (gateway, função, rede).
2. **O critério de aceitação do RNF-06 para o cold start passa a ser de ponta a ponta e ligado
   ao comportamento do aplicativo:** a primeira requisição após inatividade conclui em ≤ 10 s
   (metade do tempo limite de resposta do aplicativo), sem retentativa e com indicador de
   progresso. Com esse critério a medição atende (máximo 8,0 s). O alvo original de 1,5 s fica
   como meta para quando a mitigação for aplicada. O alvo de regime contínuo (p50 ≤ 200 ms,
   p95 ≤ 500 ms) não muda e foi atendido.
3. **A mitigação é adiada e documentada como trabalho futuro**, na ordem de preferência:
   tráfego periódico que atravesse o gateway (o job do Scheduler chamando uma rota leve pela URL
   pública a cada poucos minutos, custo desprezível), que manteria quentes o gateway e as conexões
   da função; chamada direta do aplicativo ao serviço Cloud Run com validação do token na função,
   que dispensa a partida do gateway ao custo de perder o contrato OpenAPI imposto por ele; e, se
   a parcela da função importar depois disso, `min_instancias = 1` combinado com o tráfego
   periódico.
4. **`min_instancias` continua 0** no ambiente de desenvolvimento: sozinho, não muda a mediana e
   tem custo fixo.

## Alternativas consideradas

- **Testar o keep-alive pelo gateway nesta sprint.** Exigiria uma rota pública leve no contrato
  OpenAPI, na função e no job do Scheduler, mais 2 h de medição; é a mitigação preferida, mas o
  entregável da sprint é a avaliação, e o resultado negativo com a atribuição das parcelas já
  responde ao requisito. Fica como primeiro item do trabalho futuro.
- **Manter o alvo de 1,5 s sem revisão.** Deixaria um requisito que nenhuma configuração da
  função atende, por um componente que o projeto não controla, sem dizer o que é aceitável em
  campo.
- **Dispensar o gateway agora.** Mudança de arquitetura (autenticação, contrato, chaves de API)
  tarde demais no cronograma e sem medição que a justifique além do cold start.

## Consequências

- `docs/requisitos.md` (RNF-06) passa a trazer o critério revisado e a referência a este ADR;
  `docs/arquitetura.md` (CE-18) passa a descrever o cold start como do gateway e da função.
- A primeira validação de um operador após um intervalo sem uso da frota continua levando
  alguns segundos, com indicador de progresso; isso deve constar na defesa como limitação do
  modelo gerenciado, não do código.
- O trabalho futuro ganha um experimento definido, com ferramenta pronta (`make carga-frio`).

## Referências

- `docs/avaliacao/escalabilidade.md` (seção 3), `docs/avaliacao/evidencias/cold-start-min0.md`
  e `cold-start-min1.md`; `docs/requisitos.md` (RNF-06); `docs/arquitetura.md` (CE-18);
  ADR-0009, ADR-0012, ADR-0015.
