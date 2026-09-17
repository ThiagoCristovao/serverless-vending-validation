# ADR-0009: Erros em Problem Details (RFC 9457) e idempotência por chave do cliente

- **Status:** aceita
- **Data:** 2026-09-16
- **Decisores:** Thiago Cristovão de Souza
- **Sprint:** 1 (contrato) e 2 (arquitetura)

## Contexto

A Sprint 1 pede "códigos de erro e formatos de resposta" no contrato; a Sprint 2 pede "estratégia
de tratamento de erros, idempotência da validação e retentativa em caso de falha de rede no
dispositivo". Em campo, a rede móvel é intermitente: uma resposta perdida não pode gerar duas
validações nem duas rotações de código.

## Decisão

1. **Erros** usam `application/problem+json` (RFC 9457) com os campos padrão (`type`, `title`,
   `status`, `detail`, `instance`) e duas extensões: `codigo` (enumeração estável de erros de
   domínio, ex.: `validacao_em_andamento`) e `idValidacao` quando aplicável. O aplicativo decide o
   que mostrar pelo `codigo`, nunca pelo texto.
2. **Idempotência** em `POST /v1/validacoes` por cabeçalho `Idempotency-Key` (UUID gerado pelo
   aplicativo por leitura de QR). Repetir a mesma chave com o mesmo corpo devolve a mesma resposta
   (201); mesma chave com corpo diferente devolve 422 `chave_idempotencia_conflitante`. Além disso,
   o domínio garante **uma única validação aberta por máquina** (409 `validacao_em_andamento`).
3. **Conclusão** (`POST /v1/validacoes/{id}/conclusao`) é idempotente por natureza: repetir sobre
   uma validação já concluída devolve 200 com o mesmo resultado se o corpo for igual, ou 409
   `validacao_ja_concluida` se divergir.
4. **Retentativa no dispositivo:** até 3 reenvios com espera exponencial (1 s, 2 s, 4 s) para
   erros de rede, tempo esgotado e 503; nunca para outros 4xx. Se a resposta da conclusão se
   perder, o aplicativo consulta `GET /v1/validacoes/{id}` antes de reenviar.

## Alternativas consideradas

- **Formato de erro próprio (`{erro, mensagem}`)** — reinventa um padrão existente; descartada.
- **Idempotência derivada do corpo (hash do payload do QR + operador + janela)** — funciona, mas
  não distingue duas leituras legítimas do mesmo QR em sequência; descartada em favor da chave
  explícita, mantendo a regra de negócio de validação única por máquina.
- **Sem idempotência, tratando duplicatas na conclusão** — deixaria contrassenhas "fantasmas" e
  eventos duplicados; descartada.

## Consequências

### Positivas

- Contrato de erro uniforme entre serviço, aplicativo e testes; fácil de mapear para estados de
  tela.
- Cenários CE-02 e CE-03 da arquitetura têm comportamento determinístico e testável (Sprint 8).

### Negativas e riscos

- O serviço precisa persistir a chave de idempotência com a validação e consultá-la em transação
  (ver `docs/modelo-dados.md`).
- Chaves de idempotência antigas devem expirar junto com a validação para não crescer sem limite.

## Referências

- RFC 9457, *Problem Details for HTTP APIs*; IETF draft *The Idempotency-Key HTTP Header Field*.
- `api/openapi.yaml`; `docs/arquitetura.md`, seções Idempotência e Cenários de exceção.
