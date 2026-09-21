# Requisitos

**Sprint 1 — Requisitos e modelagem de dados.** Documento consolidado de requisitos do ecossistema
de validação. Deriva do fluxo da Figura 1 do TCC 1 e das decisões registradas em [adr/](adr/).
Lacunas estão marcadas com `> **TODO:**`.

Critério de conclusão da sprint: todo campo trafegado entre os componentes tem tipo,
obrigatoriedade e origem definidos ([payload-qr.md](payload-qr.md), [modelo-dados.md](modelo-dados.md),
[`api/openapi.yaml`](../api/openapi.yaml)), e o contrato permite implementar cliente e servidor de
forma independente.

## 1. Objetivo e escopo

Permitir que um operador de campo valide uma máquina de vending offline (caso de estudo: Crane
National 168) por meio de um aplicativo móvel e de um serviço serverless independente do sistema
central, de modo que a operação continue mesmo com o sistema central indisponível. Fora do escopo
(TCC 1, Seção 4): integração com sistemas legados reais, múltiplos modelos de máquina, pagamentos,
gestão completa de frota, cadastro administrativo com interface.

## 2. Atores

| Ator | Descrição |
|---|---|
| **Operador de campo** | Pessoa que visita a máquina para abastecimento, coleta e manutenção. Usa o aplicativo. Cadastrado previamente com e-mail e senha. |
| **Aplicativo** | Aplicativo Flutter (Android) do operador. |
| **Serviço de validação** | Cloud Run function em Go, exposta pelo API Gateway. |
| **Sistema central** | Sistema de retaguarda da rede de vending. Neste trabalho, um consumidor simulado do Pub/Sub. |
| **Administrador da rede** | Cadastra máquinas, operadores e chaves de assinatura do QR; imprime adesivos. Sem interface neste trabalho: atua por scripts (`ferramentas/`). |

## 3. Fluxo de operação (Figura 1 do TCC 1, refinado)

1. O operador autentica-se no aplicativo (e-mail e senha, Firebase Authentication).
2. Abre o leitor e aponta a câmera para o QR fixado na máquina.
3. O aplicativo valida o payload localmente (estrutura, validade do adesivo, modelo suportado e,
   desejável, assinatura).
4. O aplicativo envia `POST /v1/validacoes` autenticada, com chave de idempotência.
5. O serviço verifica assinatura, validade, cadastro da máquina e do operador; cria a validação;
   deriva a contrassenha corrente e a próxima; persiste; publica `validacao.iniciada`.
6. O aplicativo exibe a contrassenha, a próxima contrassenha e os dados da máquina.
7. O operador digita a contrassenha no teclado da máquina (código supervisor), lê o medidor e a
   contagem de vendas no modo supervisor, programa o próximo código e informa as leituras no
   aplicativo.
8. O aplicativo envia `POST /v1/validacoes/{id}/conclusao`; o serviço fecha a validação,
   rotaciona o contador se `codigoRotacionado` for verdadeiro e publica `validacao.concluida`.
9. O sistema central consome os eventos quando estiver disponível.

Fluxo alternativo A — sem rede no passo 4 ou 8: o aplicativo informa o operador e permite tentar
novamente sem reler o QR ([arquitetura.md](arquitetura.md), CE-01 a CE-03).

Fluxo alternativo B — validação aberta expira antes da conclusão: o operador precisa reler o QR;
o contador não rotaciona.

## 4. Requisitos funcionais

Prioridade: **E** essencial (sem ele o fluxo não fecha), **I** importante, **D** desejável.

| ID | Requisito | Origem | Pri. | Componente | Critério de aceitação |
|---|---|---|---|---|---|
| RF-01 | Autenticar o operador por e-mail e senha via Firebase Authentication | Fluxo 1 | E | Aplicativo | Credenciais válidas levam à tela de leitura; inválidas exibem erro sem revelar qual campo falhou |
| RF-02 | Manter a sessão e renovar o ID token automaticamente antes de cada chamada | Fluxo 1 | E | Aplicativo | Após 1 h de uso, a chamada seguinte não devolve 401 |
| RF-03 | Ler QR code com a câmera do dispositivo | Fluxo 2 | E | Aplicativo | Um QR válido é decodificado em até 2 s a 20–40 cm em luz ambiente |
| RF-04 | Validar localmente o payload do QR: estrutura, tipos, `exp`, modelo | Fluxo 3 | E | Aplicativo | Payload malformado ou vencido não gera requisição e exibe motivo |
| RF-05 | Verificar localmente a assinatura Ed25519 do QR com a chave pública embutida | Fluxo 3 | D | Aplicativo | Adesivo forjado é rejeitado sem rede |
| RF-06 | Enviar requisição de validação autenticada com `Idempotency-Key` | Fluxo 4 | E | Aplicativo | Cabeçalhos `Authorization: Bearer` e `Idempotency-Key` presentes; reenvio usa a mesma chave |
| RF-07 | Verificar assinatura, `kid`, `exp` e modelo do payload | Fluxo 5 | E | Serviço | 422 para assinatura inválida ou `kid` desconhecido; 410 para vencido |
| RF-08 | Verificar que a máquina existe e está ativa | Fluxo 5 | E | Serviço | 404 para desconhecida; 409 para inativa |
| RF-09 | Verificar que o operador existe e está ativo | Fluxo 5 | E | Serviço | 403 para inexistente ou inativo |
| RF-10 | Garantir uma única validação aberta por máquina | Fluxo 5 | E | Serviço | Segunda requisição para a mesma máquina devolve 409 com o `idValidacao` existente |
| RF-11 | Derivar a contrassenha corrente e a próxima (HMAC, contador) | Fluxo 5 | E | Serviço | Mesmo `(idMaquina, contador)` produz sempre o mesmo código de 4 dígitos |
| RF-12 | Persistir a validação com estado `aberta`, prazo e chave de idempotência | Fluxo 5 | E | Serviço | Documento em `validacoes` conforme modelo de dados |
| RF-13 | Publicar `validacao.iniciada` no tópico de validações | Fluxo 5 | E | Serviço | Mensagem com atributos `tipo`, `idValidacao`, `idMaquina` e chave de ordenação |
| RF-14 | Responder à requisição repetida com a mesma chave de idempotência com o mesmo resultado | Fluxo 4 | E | Serviço | Reenvio devolve 201 idêntico; corpo diferente devolve 422 |
| RF-15 | Exibir contrassenha, próxima contrassenha e dados da máquina | Fluxo 6 | E | Aplicativo | Contrassenha legível a 1 m; dados: id, modelo, localização, última validação |
| RF-16 | Capturar leituras operacionais (medidor, unidades vendidas) | Fluxo 7 | E | Aplicativo | Campos numéricos obrigatórios, não negativos |
| RF-17 | Registrar se o próximo código foi programado na máquina | Fluxo 7 | E | Aplicativo | Confirmação explícita do operador (`codigoRotacionado`) |
| RF-18 | Concluir a validação: fechar, rotacionar contador se aplicável, persistir leituras | Fluxo 8 | E | Serviço | Estado `concluida`; contador incrementado somente com `codigoRotacionado = true` |
| RF-19 | Publicar `validacao.concluida` | Fluxo 8 | E | Serviço | Mensagem com leituras e indicador de rotação |
| RF-20 | Consultar validação por identificador | Fluxo alt. A | I | Serviço, Aplicativo | `GET /v1/validacoes/{id}` devolve o estado atual; contrassenhas só quando `aberta` |
| RF-21 | Expirar validações abertas após o prazo | Fluxo alt. B | I | Serviço | Após `expiraEm`, consulta devolve `expirada`; nova validação da máquina é aceita |
| RF-22 | Tratar erros e exibir estado ao operador por `codigo` | Fluxos 4–8 | E | Aplicativo | Cada `codigo` do contrato tem mensagem e ação definidas ([arquitetura.md](arquitetura.md), Telas) |
| RF-23 | Reenviar automaticamente em falha de rede com espera exponencial | Fluxo alt. A | E | Aplicativo | Até 3 reenvios (1 s, 2 s, 4 s); nunca para 4xx exceto renovação após 401 |
| RF-24 | Consumir eventos do tópico e registrá-los de forma idempotente | Fluxo 9 | E | Sistema central simulado | Evento duplicado não gera segundo registro |
| RF-25 | Encaminhar eventos com falhas sucessivas para a fila de mensagens mortas | Fluxo 9 | E | Infraestrutura | Após 5 tentativas, mensagem na DLQ |
| RF-26 | Registrar logs estruturados com identificadores de correlação em todas as operações | Todos | I | Serviço | Toda linha de log de uma requisição contém `idValidacao` (quando houver), `idMaquina`, `uid` |

## 5. Requisitos não funcionais

| ID | Atributo | Requisito | Métrica / alvo | Verificação |
|---|---|---|---|---|
| RNF-01 | Resiliência | Nenhum evento é perdido quando o consumidor do sistema central está indisponível | 100 % dos eventos publicados durante a indisponibilidade são consumidos após o retorno | Sprint 9, injeção de falhas |
| RNF-02 | Resiliência | Eventos que falham repetidamente não bloqueiam os demais | DLQ após 5 tentativas; consumo dos outros continua | Sprint 8 |
| RNF-03 | Independência | Validação e conclusão completam com o sistema central indisponível | 0 requisições com erro atribuível ao sistema central | Sprint 9 |
| RNF-04 | Independência | A resposta da validação não depende da publicação no Pub/Sub | Persistência precede a publicação; falha de publicação não gera erro ao operador | Sprint 5/9 (ver CE-14) |
| RNF-05 | Escalabilidade | O serviço atende operadores simultâneos sem degradação perceptível | p95 da latência do serviço abaixo do alvo com N operadores simultâneos. `> **TODO:** definir N (sugestão: 10, 50, 100) e alvo de p95 (sugestão: 500 ms) com o orientador` | Sprint 10, teste de carga |
| RNF-06 | Tempo de resposta | Cold start aceitável para uso em campo | `> **TODO:** alvo de cold start (sugestão: < 1,5 s) e de regime contínuo (sugestão: p50 < 200 ms)` | Sprint 10 |
| RNF-07 | Segurança | Somente operadores autenticados acessam o serviço | Requisição sem JWT válido é rejeitada pelo gateway (401) antes de chegar à função | Sprint 5 |
| RNF-08 | Segurança | A função não é invocável publicamente | Somente a conta de serviço do gateway tem `run.invoker`; chamada direta devolve 403 | Sprint 5 |
| RNF-09 | Segurança | Menor privilégio para cada conta de serviço | Papéis conforme módulo `iam`; nenhum `Editor`/`Owner` em contas de serviço | Sprint 3 |
| RNF-10 | Segurança | Nenhum segredo no aplicativo nem no repositório | Chave HMAC no Secret Manager; chave privada do QR fora da nuvem; `.gitignore` cobre credenciais | Sprint 3/6 |
| RNF-11 | Segurança | O aplicativo não acessa o Firestore diretamente | Regras negam todo acesso de cliente | Sprint 3 |
| RNF-12 | Observabilidade | Logs estruturados, métricas de latência e de backlog, alertas de DLQ | Painel e alertas do módulo `observabilidade` | Sprint 3/5 |
| RNF-13 | Idempotência | Reenvios não duplicam validações, rotações nem eventos | Testes de duplicação | Sprint 8 |
| RNF-14 | Reprodutibilidade | Ambiente recriável do zero por Terraform | `destroy` + `apply` sem intervenção manual | Sprint 3 |
| RNF-15 | Testabilidade | Regras de domínio testáveis sem nuvem | `go test ./...` sem rede | Sprint 4 |
| RNF-16 | Usabilidade em campo | Fluxo em no máximo três telas; contrassenha legível; feedback em cada estado | Revisão de telas; teste em dispositivo | Sprint 7 |
| RNF-17 | Compatibilidade | Android como plataforma-alvo | `> **TODO:** versão mínima do Android (sugestão: API 26)` | Sprint 6 |
| RNF-18 | Custo | Gasto mensal controlado | Orçamento com alertas; `max_instance_count` na função | Sprint 0/5 |

## 6. Regras de negócio

| ID | Regra |
|---|---|
| RN-01 | A contrassenha é o código supervisor de 4 dígitos da máquina, derivado por HMAC de `(idMaquina, contador)` (ADR-0002). |
| RN-02 | A próxima contrassenha é derivada de `contador + 1` e só passa a valer quando a conclusão informa `codigoRotacionado = true`. |
| RN-03 | Uma máquina tem no máximo uma validação `aberta` por vez. |
| RN-04 | Uma validação aberta expira em **15 minutos** (`> **TODO:** confirmar com orientador`); depois disso não pode ser concluída. |
| RN-05 | O QR é válido apenas até `exp` (validade do adesivo) e apenas se assinado por uma chave `kid` ativa. |
| RN-06 | Só operadores com `ativo = true` podem validar. |
| RN-07 | Leituras operacionais são inteiros não negativos; o medidor não pode ser menor que o da última validação concluída da mesma máquina (`> **TODO:** decidir se é erro 422 ou apenas aviso`). |
| RN-08 | Modelos suportados nesta versão: `CN168`. |

## 7. Restrições e premissas

- Máquina totalmente offline; toda comunicação passa pelo dispositivo do operador.
- O operador tem conectividade móvel intermitente, mas não nula, durante a visita.
- Cadastro de máquinas, operadores e chaves é feito por scripts com dados fictícios.
- Avaliação em ambiente de desenvolvimento, sem usuários finais (TCC 1, Seção 3.4.7).

## 8. Glossário

| Termo | Definição |
|---|---|
| Contrassenha | Código supervisor de 4 dígitos que dá acesso ao modo de serviço da máquina |
| Validação | Registro de uma visita do operador a uma máquina, do início à conclusão |
| Leituras operacionais | Valores lidos no modo supervisor: medidor (vendas totais) e unidades vendidas |
| Adesivo | QR code impresso e fixado na máquina, com payload assinado |
| `kid` | Identificador da versão da chave que assinou o adesivo |
| DLQ | Fila de mensagens mortas (dead-letter queue) |

## 9. Rastreabilidade

| Requisitos | Endpoint / componente | Sprint de implementação |
|---|---|---|
| RF-01, RF-02 | Firebase Auth SDK no aplicativo | 6 |
| RF-03, RF-04, RF-05 | Tela de leitura | 6 |
| RF-06, RF-14, RF-15, RF-22, RF-23 | `POST /v1/validacoes`, tela de resultado | 4 (domínio), 5 (HTTP), 7 (app) |
| RF-07 a RF-13 | `POST /v1/validacoes` | 4, 5 |
| RF-16 a RF-19 | `POST /v1/validacoes/{id}/conclusao` | 4, 5, 7 |
| RF-20, RF-21 | `GET /v1/validacoes/{id}` | 4, 5, 7 |
| RF-24, RF-25 | Sistema central simulado, módulo `pubsub` | 3, 8 |
| RF-26 | Adaptador HTTP e logs | 5 |
