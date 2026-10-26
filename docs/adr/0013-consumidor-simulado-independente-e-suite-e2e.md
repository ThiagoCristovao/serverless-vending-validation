# ADR-0013 — Sistema central simulado como módulo independente e suíte e2e contra o ambiente dev

- **Status:** aceita
- **Data:** 2026-10-26
- **Sprint:** 8

## Contexto

A Sprint 8 exige um consumidor que represente o sistema central da rede de vending e uma suíte
de testes de ponta a ponta que cubra o fluxo principal, mensagens duplicadas e o encaminhamento
à fila de mensagens mortas (DLQ) após falhas sucessivas. O sistema central real é um sistema
separado, mantido por outra equipe, que conhece apenas o contrato dos eventos
([modelo-dados.md](../modelo-dados.md), seção 6). As Sprints 9 e 10 precisam desligar e religar
esse consumidor e provocar falhas de processamento sob demanda.

## Decisão

1. **Módulo Go próprio** (`sistema-central-simulado/`), sem importar o código do serviço. O
   contrato dos eventos é reescrito no pacote `eventos` a partir da documentação, com campos
   desconhecidos tolerados e campos obrigatórios e versão conferidos. A compatibilidade entre os
   dois lados é verificada pela suíte e2e, não por compartilhamento de tipos.
2. **Idempotência por `(idValidacao, tipo)` em arquivo JSON Lines** (`armazenamento`): cada evento
   novo é acrescentado ao arquivo e sincronizado em disco antes do `ack`; o índice em memória é
   reconstruído na partida, de modo que uma duplicata é reconhecida mesmo após reinício do
   processo. Sem banco de dados: o volume é pequeno e o arquivo é a evidência dos testes.
3. **Interruptor de falhas pela API administrativa** (`falhas`, `admin`): modos `rejeitar`
   (nack antes de persistir, por máquina ou geral), `confirmacao-perdida` (persiste e não
   confirma, simulando ack perdido) e `nenhum`; além de `pausar`/`retomar` o consumo. Os
   scripts de injeção de falhas da Sprint 9 usam a mesma API.
4. **Suíte e2e dentro do módulo do consumidor** (`e2e/`, tag de compilação `e2e`), executada
   contra o ambiente `svv-dev` real: obtém o ID token do operador de testes, assina adesivos com
   a chave privada local, chama o gateway e sobe o consumidor **em processo**, com registro em
   arquivo temporário e API administrativa em porta efêmera. A CI apenas compila a suíte
   (`go vet -tags e2e`); a execução é local, por `make e2e`, porque exige credenciais.
5. **Testes de unidade do consumidor com o servidor Pub/Sub falso em processo** (`pstest`), que
   reproduz ordenação por chave e política de mensagens mortas, dispensando Docker na CI.

## Alternativas consideradas

- **Importar `servico/internal/dominio` no consumidor** (um só módulo ou `go.work`): rejeitada,
  acoplaria dois sistemas que na prática são independentes e esconderia incompatibilidades de
  contrato.
- **SQLite** para o registro: rejeitada por exigir cgo ou dependência pesada sem ganho para o
  volume do trabalho.
- **Suíte e2e em diretório próprio na raiz**: rejeitada; ela importa os pacotes do consumidor
  (para subi-lo em processo) e não faria sentido como módulo separado.
- **Produzir duplicatas só por republicação direta no tópico**: mantida como um dos casos, mas
  complementada pelo modo `confirmacao-perdida`, que reproduz a causa real de duplicata no
  at-least-once (ack perdido após processar).

## Consequências

- O consumidor roda localmente (Sprint 8) e pode ser levado ao Cloud Run sem mudanças de código,
  já que usa ADC e variáveis de ambiente; a conta de serviço `svv-dev-central-simulado` pode ser
  personificada com `-impersonar` para demonstrar o menor privilégio.
- A suíte e2e demora alguns minutos por causa da espera exponencial entre reentregas (10 s a
  600 s) antes da DLQ após 5 tentativas; o cenário pode ser pulado com `-short`.
- Duas máquinas de teste (`VM-E2E1`, `VM-E2E2`, localização `E2E-L`) ficam semeadas no Firestore
  de dev, separadas da `VM-2047` usada nas demonstrações manuais com o aplicativo.
- O `pstest` não remove da assinatura ordenada a mensagem que encaminhou à DLQ (a chave fica
  presa só no servidor falso); a retomada da mesma máquina após a DLQ é verificada só na e2e.
