# Planejamento de Sprints — TCC 2

**Projeto:** Aplicação mobile com arquitetura serverless para validação de máquinas de vendas
**Autor:** Thiago Cristovão de Souza
**Orientador:** Prof. Me. Muriel de Souza Godoi
**Curso:** Engenharia de Computação — UTFPR, Câmpus Apucarana

---

## 1. Propósito deste documento

Este documento organiza a execução do TCC 2 em sprints, detalhando as sete fases do método
descrito na Seção 3.4 da proposta em incrementos menores. Cada sprint entrega um artefato
verificável, de modo que o progresso seja demonstrável a qualquer momento e que eventuais
atrasos fiquem contidos em um incremento em vez de comprometerem uma fase inteira.

Não são definidas datas nem durações. A ordem das sprints reflete dependências técnicas: uma
sprint só inicia quando os artefatos de que depende estiverem concluídos.

## 2. Convenções

Cada sprint é descrita por quatro elementos:

- **Objetivo** — a pergunta que a sprint responde ou o incremento que ela entrega
- **Backlog** — as atividades previstas
- **Entregável** — o artefato concreto que resulta da sprint
- **Critério de conclusão** — a condição objetiva que caracteriza a sprint como encerrada

A redação da monografia não figura como sprint isolada. Conforme previsto na etapa 10 do
cronograma, ela é transversal: ao final de cada sprint, o capítulo correspondente é atualizado
com as decisões tomadas e os resultados obtidos, evitando o acúmulo de redação ao final.

---

## 3. Visão geral

| Sprint | Título | Fase do método (Seção 3.4) |
|--------|--------|----------------------------|
| 0 | Fundação do ambiente e do repositório | — (preparatória) |
| 1 | Requisitos e modelagem de dados | 3.4.1 |
| 2 | Projeto da arquitetura | 3.4.2 |
| 3 | Infraestrutura base como código | 3.4.3 |
| 4 | Serviço de validação — domínio | 3.4.4 |
| 5 | Serviço de validação — integração e implantação | 3.4.3 / 3.4.4 |
| 6 | Aplicação mobile — autenticação e leitura | 3.4.5 |
| 7 | Aplicação mobile — integração e conclusão do fluxo | 3.4.5 |
| 8 | Integração e testes end-to-end | 3.4.6 |
| 9 | Avaliação de resiliência e independência | 3.4.7 |
| 10 | Avaliação de escalabilidade e consolidação | 3.4.7 |

---

## Sprint 0 — Fundação do ambiente e do repositório

**Objetivo.** Estabelecer a base técnica e organizacional do projeto antes de qualquer
implementação, de modo que todo o código produzido a partir da Sprint 1 já nasça versionado,
rastreável e com segredos protegidos.

**Backlog**

- Criar o repositório e definir a estrutura de diretórios (serviço em Go, aplicação Flutter,
  módulos Terraform, documentação)
- Configurar `.gitignore` cobrindo as três stacks, com atenção a arquivos de estado do
  Terraform e credenciais de conta de serviço
- Definir e registrar a licença do código, em conjunto com o orientador
- Criar o projeto na Google Cloud Platform, habilitar as APIs necessárias e configurar o
  faturamento
- Criar o bucket no Cloud Storage que servirá de backend remoto para o estado do Terraform
- Definir a convenção de branches e de mensagens de commit

**Entregável.** Repositório estruturado, com README inicial, licença e projeto GCP acessível.

**Critério de conclusão.** É possível clonar o repositório e autenticar na GCP seguindo apenas
as instruções do README.

---

## Sprint 1 — Requisitos e modelagem de dados

**Objetivo.** Converter o fluxo esboçado na Figura 1 da proposta em especificações formais,
eliminando as indefinições deixadas em aberto no TCC 1.

**Backlog**

- Elaborar os requisitos funcionais a partir do fluxo de operação: autenticação, leitura do QR
  code, validação local, requisição ao serviço, geração da contrassenha, envio assíncrono ao
  sistema central e persistência dos registros
- Elaborar os requisitos não funcionais, contemplando resiliência, escalabilidade, segurança,
  observabilidade e tempo de resposta, com métricas associadas sempre que aplicável
- Definir o esquema definitivo do payload do QR code, fechando a proposta preliminar
  (identificador da máquina, modelo, identificador de localização e token de validação)
- Modelar as coleções e os documentos do Firestore, incluindo os campos de leitura operacional
  informados pelo operador
- Especificar o contrato da API em OpenAPI, incluindo códigos de erro e formatos de resposta

**Entregável.** Documento consolidado de requisitos e especificação OpenAPI versionada.

**Critério de conclusão.** Todo campo trafegado entre os componentes tem tipo, obrigatoriedade
e origem definidos. O contrato da API é suficiente para implementar cliente e servidor de
forma independente.

---

## Sprint 2 — Projeto da arquitetura

**Objetivo.** Detalhar como os componentes se comunicam e como o sistema se comporta diante de
falhas, antes de escrever código de produção.

**Backlog**

- Elaborar o diagrama de componentes do ecossistema
- Elaborar os diagramas de sequência do fluxo principal e dos fluxos de exceção
- Definir os tópicos e as assinaturas do Pub/Sub, incluindo a fila de mensagens mortas e a
  política de retentativa
- Especificar a estratégia de autenticação e autorização entre a aplicação mobile, o API
  Gateway e o serviço de validação, definindo quais claims do JWT são verificadas e em que
  camada
- Definir a estratégia de tratamento de erros, idempotência da validação e retentativa em caso
  de falha de rede no dispositivo
- Refinar os esboços de tela da Figura 2, incorporando os estados de carregamento, erro e
  falha de conectividade

**Entregável.** Documento de arquitetura com diagramas, modelo de dados e especificação dos
fluxos de exceção.

**Critério de conclusão.** Cada cenário de falha previsto na avaliação técnica (Seção 3.4.7)
tem um comportamento esperado documentado.

---

## Sprint 3 — Infraestrutura base como código

**Objetivo.** Provisionar, exclusivamente por meio de Terraform, os recursos que não dependem
do código da aplicação, validando que o ambiente é reproduzível desde o início.

**Backlog**

- Configurar o backend remoto do Terraform no Cloud Storage, com versionamento e bloqueio de
  estado
- Provisionar o Firestore e as regras de segurança iniciais
- Provisionar os tópicos, as assinaturas e a fila de mensagens mortas do Pub/Sub
- Provisionar o Firebase Authentication e configurar o provedor de identidade
- Definir as contas de serviço e as políticas de IAM segundo o princípio do menor privilégio
- Configurar os recursos de Cloud Logging e Cloud Monitoring
- Parametrizar os módulos para permitir múltiplos ambientes

**Entregável.** Módulos Terraform versionados que provisionam o ambiente de desenvolvimento a
partir do zero.

**Critério de conclusão.** Executar `terraform destroy` seguido de `terraform apply` recria o
ambiente integralmente, sem intervenção manual no console da GCP.

---

## Sprint 4 — Serviço de validação: domínio

**Objetivo.** Implementar e testar as regras de negócio da validação isoladamente, sem
dependência de infraestrutura de nuvem.

**Backlog**

- Estruturar o projeto em Go segundo os princípios de arquitetura limpa, separando as camadas
  de transporte, aplicação, domínio e infraestrutura
- Implementar a decodificação e a validação do payload do QR code
- Implementar a verificação do token de validação transportado pelo código
- Implementar o algoritmo de geração da contrassenha
- Definir as interfaces (ports) para persistência e publicação de mensagens, sem implementá-las
- Escrever os testes unitários das regras de domínio, incluindo os casos de código inválido,
  expirado e malformado

**Entregável.** Pacote de domínio com cobertura de testes, compilável e testável sem
credenciais da GCP.

**Critério de conclusão.** A suíte de testes unitários executa integralmente em ambiente local,
sem acesso à rede.

---

## Sprint 5 — Serviço de validação: integração e implantação

**Objetivo.** Conectar o domínio aos serviços gerenciados e expor o serviço como um endpoint
autenticado e funcional.

**Backlog**

- Implementar o adaptador de persistência no Firestore
- Implementar o adaptador de publicação no Pub/Sub
- Implementar o handler HTTP conforme o contrato OpenAPI definido na Sprint 1
- Estender os módulos Terraform para provisionar o Cloud Functions de segunda geração e o API
  Gateway
- Configurar a validação do JWT na camada do API Gateway
- Instrumentar o serviço com logs estruturados e métricas
- Escrever os testes de integração utilizando os emuladores do Firestore e do Pub/Sub

**Entregável.** Serviço de validação implantado e acessível por meio do API Gateway, exigindo
autenticação.

**Critério de conclusão.** Uma requisição autenticada com payload válido retorna a contrassenha,
persiste o registro no Firestore e publica a mensagem no tópico correspondente.

---

## Sprint 6 — Aplicação mobile: autenticação e leitura

**Objetivo.** Entregar a primeira metade do fluxo do operador, até a extração dos dados do QR
code.

**Backlog**

- Estruturar o projeto Flutter em camadas e definir a abordagem de gerenciamento de estado
- Integrar o Firebase Authentication e implementar a tela de autenticação do operador
- Implementar a tela de leitura do QR code utilizando o pacote `mobile_scanner`, conforme
  esboçado na Figura 2a
- Implementar a validação local dos dados extraídos do código
- Tratar as permissões de câmera e os estados de erro de leitura

**Entregável.** Aplicação que autentica o operador, lê um QR code válido e exibe os dados
decodificados.

**Critério de conclusão.** O fluxo funciona em dispositivo físico, do login até a decodificação,
sem comunicação com o serviço de validação.

---

## Sprint 7 — Aplicação mobile: integração e conclusão do fluxo

**Objetivo.** Completar o fluxo do operador, conectando a aplicação ao serviço de validação.

**Backlog**

- Implementar o cliente HTTP com injeção automática do token de autenticação
- Implementar a tela de resultado da validação, exibindo a contrassenha e os dados da máquina,
  conforme esboçado na Figura 2b
- Implementar a captura das leituras operacionais informadas pelo operador
- Implementar a confirmação de conclusão da operação
- Tratar os cenários de erro: falha de rede, token expirado, código já validado e resposta de
  erro do serviço
- Escrever os testes automatizados dos principais fluxos de interface

**Entregável.** Aplicação mobile completa, integrada ao serviço de validação.

**Critério de conclusão.** O fluxo previsto na Figura 1 é executável de ponta a ponta em
dispositivo físico, incluindo os caminhos de exceção.

---

## Sprint 8 — Integração e testes end-to-end

**Objetivo.** Validar o ecossistema como um todo, incluindo o consumo das mensagens pelo
sistema central.

**Backlog**

- Implementar o consumidor simulado do sistema central, assinante do tópico do Pub/Sub
- Implementar a suíte de testes end-to-end cobrindo o fluxo completo, da leitura do código até
  o consumo da mensagem
- Verificar o comportamento de idempotência diante de mensagens duplicadas
- Verificar o encaminhamento de mensagens para a fila de mensagens mortas após sucessivas
  falhas de processamento
- Consolidar a documentação de execução do ambiente completo

**Entregável.** Suíte de testes end-to-end automatizada e consumidor simulado.

**Critério de conclusão.** A suíte executa contra o ambiente de desenvolvimento e cobre tanto o
fluxo principal quanto os cenários de duplicação e falha de processamento.

---

## Sprint 9 — Avaliação de resiliência e independência

**Objetivo.** Produzir evidências para dois dos três atributos de qualidade que fundamentam o
objetivo geral do trabalho.

**Backlog**

- Executar os testes de injeção de falhas previstos na Seção 3.4.7: indisponibilidade do
  consumidor de mensagens, falhas intermitentes de rede no dispositivo e indisponibilidade do
  provedor de autenticação
- Verificar que as operações em andamento não são perdidas e que o operador recebe retorno
  adequado em cada cenário
- Executar os cenários de indisponibilidade deliberada do sistema central durante operações de
  validação
- Verificar que as mensagens geradas durante a indisponibilidade permanecem preservadas e são
  consumidas após o restabelecimento, sem perda e sem intervenção manual
- Coletar as evidências por meio das ferramentas de observabilidade do provedor

**Entregável.** Relatório de resiliência e independência, com evidências coletadas.

**Critério de conclusão.** Cada cenário de falha documentado na Sprint 2 foi executado e teve
seu comportamento registrado.

---

## Sprint 10 — Avaliação de escalabilidade e consolidação

**Objetivo.** Concluir a avaliação técnica e consolidar a análise arquitetural do trabalho.

**Backlog**

- Executar os testes de carga simulando volumes simultâneos crescentes de operações de
  validação
- Coletar o tempo de resposta médio e os percentis p50, p95 e p99
- Registrar o número de instâncias paralelas alocadas pela plataforma e a estabilidade da
  latência fim a fim conforme a carga cresce
- Medir separadamente os tempos de resposta em cenário de cold start e em regime contínuo
- Elaborar a análise comparativa qualitativa entre a abordagem proposta e a abordagem
  monolítica tradicional, discutindo os trade-offs das decisões arquiteturais
- Consolidar os resultados das Sprints 9 e 10 nos capítulos de resultados da monografia

**Entregável.** Relatório de escalabilidade e análise comparativa qualitativa.

**Critério de conclusão.** Os três atributos de qualidade do objetivo geral têm evidências
quantitativas ou qualitativas associadas.

---

## 4. Dependências entre sprints

A ordem proposta não é arbitrária. As dependências que não admitem inversão são:

- A Sprint 1 precede todas as demais, pois o contrato da API e o modelo de dados condicionam
  tanto o serviço quanto a aplicação mobile
- A Sprint 3 precede a Sprint 5, pois o serviço só pode ser implantado sobre infraestrutura
  provisionada
- A Sprint 5 precede a Sprint 7, pois a aplicação mobile só integra com um endpoint existente
- As Sprints 9 e 10 dependem da Sprint 8, pois a avaliação pressupõe o ecossistema integrado

As Sprints 4 e 6 são as únicas que admitem execução concorrente, uma vez que o domínio do
serviço e a camada de leitura do aplicativo não dependem entre si. Essa concorrência é uma
alternativa caso o cronograma se mostre apertado.

## 5. Riscos observados

- **Concentração de risco na Sprint 5.** É a sprint que integra o maior número de serviços
  gerenciados simultaneamente. Configurações de IAM e de validação de JWT no API Gateway são
  as fontes prováveis de atraso.
- **Dependência de dispositivo físico nas Sprints 6 e 7.** A leitura de QR code não é
  plenamente verificável em emulador.
- **Custo dos testes de carga na Sprint 10.** Vale definir limites de orçamento na GCP antes da
  execução, dado o modelo de cobrança por invocação.
