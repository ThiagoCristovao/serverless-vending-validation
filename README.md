# serverless-vending-validation

Ecossistema serverless na Google Cloud Platform para validação de máquinas de venda automática
(vending machines) que operam **offline**, tendo o **Crane National 168** como caso de estudo.

Trabalho de Conclusão de Curso 2 — Engenharia de Computação, UTFPR Câmpus Apucarana.
Autor: Thiago Cristovão de Souza. Orientador: Prof. Me. Muriel de Souza Godoi.

## O problema

Máquinas como a Crane National 168 não têm conectividade. A validação em campo depende do
aplicativo do operador e de um sistema central que, quando fica indisponível, paralisa a operação.
Este projeto desacopla a validação em um serviço serverless independente e envia os registros ao
sistema central de forma assíncrona, para que a operação em campo continue mesmo com o sistema
central fora do ar.

## Arquitetura

```mermaid
flowchart LR
  OP([Operador de campo]) --> APP[Aplicativo Flutter<br/>Android]
  APP -->|e-mail e senha| FA[Firebase<br/>Authentication]
  FA -->|JWT| APP
  APP -->|HTTPS + JWT| GW[API Gateway<br/>valida o JWT]
  GW -->|invoca com identidade própria| FN[Cloud Run function<br/>svv-validacao · Go]
  FN <--> FS[(Firestore)]
  FN -->|publica eventos| PS[[Pub/Sub<br/>svv-validacoes]]
  PS --> SC[Sistema central<br/>simulado]
  PS -. falhas sucessivas .-> DLQ[[DLQ]]
  FN --> OBS[Cloud Logging<br/>Cloud Monitoring]
```

Toda a infraestrutura é descrita em Terraform e provisionada em `us-east1`. A arquitetura completa,
com fluxos de exceção, está em [docs/arquitetura.md](docs/arquitetura.md).

## Estado do projeto

Sprints concluídas: **0** (fundação), **1** e **2** (requisitos, contrato e arquitetura),
**3** (infraestrutura base em Terraform), **4** (domínio do serviço em Go) e **5** (serviço
implantado atrás do API Gateway, com Firestore, Pub/Sub e reconciliação, verificado de ponta a
ponta em `svv-dev`). **6** (aplicativo: autenticação e leitura do QR com validação local, verificado em dispositivo
físico), **7** (aplicativo integrado ao serviço: contrassenha, leituras, conclusão e cenários de
erro, verificado em dispositivo físico contra `svv-dev`) e **8** (sistema central simulado e suíte
de ponta a ponta executada contra `svv-dev`: fluxo principal, duplicatas e DLQ) e **9** (avaliação
de resiliência e independência: cenários de falha do lado do serviço e do dispositivo executados,
com evidências em [docs/avaliacao/resiliencia.md](docs/avaliacao/resiliencia.md)) e **10**
(escalabilidade e consolidação: cargas com 10, 50 e 100 operadores em duas configurações da
função, cold start após inatividade e análise comparativa, com evidências em
[docs/avaliacao/escalabilidade.md](docs/avaliacao/escalabilidade.md) e
[docs/avaliacao/analise-comparativa.md](docs/avaliacao/analise-comparativa.md)). O
planejamento completo está em
[docs/planejamento-sprints.md](docs/planejamento-sprints.md); as decisões tomadas, em
[docs/adr](docs/adr).

## Estrutura do repositório

```
api/                       contrato OpenAPI 3.0 (canônico) e exemplos de payload
servico/                   serviço de validação em Go (Cloud Run function) — Sprint 4
aplicativo/                aplicativo Flutter do operador — Sprint 6
sistema-central-simulado/  consumidor Pub/Sub que simula o sistema central e suíte e2e — Sprint 8
ferramentas/               scripts de apoio (emuladores, injeção de falhas — Sprint 9, carga — Sprint 10)
infra/
  bootstrap/               APIs, bucket de estado e orçamento (roda uma vez)
  modulos/                 módulos Terraform por serviço gerenciado
  ambientes/dev/           composição do ambiente de desenvolvimento
docs/
  requisitos.md            requisitos funcionais e não funcionais — Sprint 1
  payload-qr.md            esquema do QR code fixado na máquina — Sprint 1
  modelo-dados.md          coleções do Firestore e esquema dos eventos — Sprint 1
  arquitetura.md           componentes, fluxos, exceções, segurança — Sprint 2
  diagramas/               fontes Mermaid dos diagramas
  avaliacao/               relatórios de resiliência (Sprint 9), escalabilidade e análise comparativa (Sprint 10), com evidências
  adr/                     registros de decisão de arquitetura
  operacao/                configuração do ambiente e convenções
  referencias/             proposta do TCC 1 (PDF)
```

## Pré-requisitos

- Linux ou macOS com `git`, `curl`, `make` e, opcionalmente, Docker (usado como alternativa para
  `terraform` e `redocly` quando não estão instalados).
- [mise](https://mise.jdx.dev) para Go e Terraform, [FVM](https://fvm.app) para Flutter e o
  [gcloud CLI](https://cloud.google.com/sdk/docs/install). Versões fixadas em `.mise.toml` e
  `aplicativo/.fvmrc`.
- Uma conta Google com faturamento habilitado na GCP.
- Um dispositivo Android físico com depuração USB (a leitura de QR não é verificável em emulador).

## Começando

```bash
git clone git@github.com:ThiagoCristovao/serverless-vending-validation.git
cd serverless-vending-validation
make configurar          # mise install + fvm install
gcloud auth login
gcloud auth application-default login
```

Em seguida, siga [docs/operacao/configuracao-ambiente.md](docs/operacao/configuracao-ambiente.md)
para criar o projeto GCP, vincular o faturamento e aplicar o bootstrap. Ao final,
`make verificar` deve passar e `make infra-dev-init` deve inicializar o ambiente contra o backend
remoto.

## Comandos

| Comando | O que faz |
|---|---|
| `make ajuda` | Lista todos os alvos |
| `make configurar` | Instala o toolchain fixado |
| `make verificar` | Terraform, OpenAPI, Go (serviço e sistema central) e Flutter (o mesmo que a CI) |
| `make infra-bootstrap-apply` | Aplica o bootstrap (APIs, bucket de estado, orçamento) |
| `make infra-dev-init` / `plan` / `apply` | Opera o ambiente de desenvolvimento |
| `make api-lint` | Valida `api/openapi.yaml` |
| `make central-executar` | Roda o sistema central simulado contra o ambiente dev (API em `:8090`) |
| `make e2e` | Suíte de ponta a ponta contra o ambiente dev (Sprint 8) |
| `make experimento CENARIO=…` | Cenário de injeção de falhas com relatório de evidências (Sprint 9) |
| `make carga OPERADORES=… CONFIGURACAO=…` | Teste de carga com operadores virtuais e relatório de evidências (Sprint 10) |
| `make carga-frio CONFIGURACAO=…` | Medição de cold start após inatividade (Sprint 10) |

## Convenções

Código, commits e documentação em português; identificadores sem acentos; Conventional Commits;
branches `sprint-N/descricao`. Detalhes em
[docs/operacao/convencoes.md](docs/operacao/convencoes.md).

## Licença

[GPL-3.0](LICENSE).
