# ADR-0011: Medidor menor que o anterior gera aviso, não erro

- **Status:** aceita
- **Data:** 2026-09-25
- **Decisores:** Thiago Cristovão de Souza
- **Sprint:** 1 (requisito RN-07) e 4 (domínio)

## Contexto

O medidor interno de vendas da máquina só cresce em operação normal. Um valor informado pelo
operador menor que o da última validação concluída indica, na maioria das vezes, erro de
digitação. Havia duas opções: rejeitar a conclusão (422) ou aceitá-la com um sinal.

## Decisão

A conclusão é **aceita** e recebe o aviso `medidor_menor_que_anterior`, devolvido na resposta
(`avisos`) e registrado no evento `validacao.concluida`. O aplicativo exibe o aviso na tela de
conclusão; o sistema central pode tratá-lo como pendência de conferência.

## Alternativas consideradas

- **Erro 422 obrigando a correção** — bloquearia casos legítimos (placa trocada ou medidor zerado em
  manutenção) e exigiria no aplicativo um fluxo de "confirmar mesmo assim" com um campo extra na
  requisição. Descartada.
- **Ignorar silenciosamente** — perde-se um sinal útil de qualidade de dados. Descartada.

## Consequências

### Positivas

- Nenhum caminho de erro adicional no aplicativo; o dado é sempre registrado.
- O sistema central recebe o sinal sem depender de regra própria.

### Negativas e riscos

- Erros de digitação entram na base; a correção é responsabilidade do sistema central.

## Referências

- `docs/requisitos.md` (RN-07), `docs/arquitetura.md` (CE-21), `api/openapi.yaml` (`Aviso`).
