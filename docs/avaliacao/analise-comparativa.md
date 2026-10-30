# Análise comparativa qualitativa: ecossistema serverless proposto × sistema monolítico central (Sprint 10)

Complementa as avaliações quantitativas de [resiliência e independência](resiliencia.md)
(Sprint 9) e de [escalabilidade](escalabilidade.md) (Sprint 10), como previsto na Seção 3.4.7
do TCC 1: uma comparação qualitativa entre a abordagem proposta e a abordagem monolítica
tradicional, com a discussão crítica das decisões arquiteturais e dos *trade-offs* envolvidos.
As afirmações sobre a abordagem proposta remetem às evidências coletadas; as afirmações sobre o
monólito descrevem a arquitetura de referência do problema (TCC 1, Seção 1) e não foram
medidas neste trabalho.

## 1. As duas abordagens

| Aspecto | Monólito central tradicional | Ecossistema proposto |
|---|---|---|
| Onde a validação é decidida | no sistema central, consultado pelo aplicativo a cada visita | no serviço de validação serverless; a leitura do adesivo é verificada no próprio aparelho (QR assinado, ADR-0002) |
| Dependência do sistema central durante a visita | total: sem o central não há validação | nenhuma: o central é um consumidor de eventos, atualizado depois (ADR-0013) |
| Comunicação entre validação e central | chamadas síncronas ao banco ou à API do central | eventos no Pub/Sub, com entrega *at-least-once*, ordenação por máquina e DLQ |
| Estado | banco relacional único do central | Firestore para o estado operacional do serviço; o central mantém o seu, alimentado pelos eventos |
| Capacidade | dimensionada previamente (servidores ou VMs) | instâncias alocadas pela plataforma sob demanda (Cloud Run functions), com limites configuráveis |
| Operação | servidores, banco, patches e janelas de manutenção sob responsabilidade da equipe | serviços gerenciados; a equipe opera código, configuração (Terraform) e observabilidade |
| Modelo de custo | fixo (capacidade instalada) | por uso (requisições, CPU-segundo, leituras e escritas, mensagens), com piso zero |

## 2. Comparação por atributo de qualidade

| Atributo | Monólito central | Proposta | Evidência / referência |
|---|---|---|---|
| **Independência do sistema central** | ponto único de falha: a indisponibilidade do central paralisa a operação em campo | a validação e a conclusão completam com o consumidor parado; os eventos ficam retidos e são consumidos depois, em ordem e sem duplicatas | resiliência, cenário A: 6/6 validações `201`/`200` com o consumidor pausado 3 min; 12/12 eventos consumidos 9 s após a retomada |
| **Resiliência a falhas de dependências** | uma falha no banco ou na rede do data center derruba todo o fluxo | falhas parciais têm comportamento definido: Pub/Sub indisponível não impede a resposta (publicação pendente e reconciliada, ADR-0012); Firestore indisponível devolve `503` rápido e o reenvio idempotente recupera | resiliência, cenários B e C |
| **Resiliência no dispositivo** | depende de como o cliente foi escrito; em geral uma chamada síncrona sem política de reenvio | política explícita: reenvios com a mesma chave de idempotência, consulta antes de reenviar a conclusão, distinção entre "sem conexão" e "autenticação indisponível" | resiliência, cenários D1–D4 e E1–E3; ADR-0009 |
| **Escalabilidade** | limitada pela capacidade instalada; picos exigem superprovisionamento | instâncias sobem com a demanda até o limite configurado; a concorrência por instância e o limite são decisões explícitas com custo e latência mensuráveis | escalabilidade, configurações A e C com 10, 50 e 100 operadores |
| **Latência** | previsível em regime (processo sempre quente), mas sujeita a filas quando saturado | em regime depende da configuração da instância; a primeira requisição após inatividade paga o cold start, que a medição atribuiu sobretudo ao gateway; instância mínima na função não basta, tráfego periódico pelo gateway é o caminho | escalabilidade, seções de regime e de cold start |
| **Consistência** | forte, em um único banco transacional | forte dentro do serviço (transações do Firestore: uma validação aberta por máquina, idempotência por chave); eventual entre o serviço e o central (eventos) | modelo de dados; ADR-0009; cenário A (eventos retidos) |
| **Segurança** | perímetro do data center; autenticação própria | autenticação delegada ao Firebase, JWT validado no gateway antes da função, função não invocável publicamente, menor privilégio por conta de serviço, nenhum acesso direto do aplicativo ao banco | RNF-07 a RNF-11 |
| **Observabilidade** | a ser construída (logs, métricas, alertas) | fornecida pela plataforma: logs estruturados, métricas de latência, instâncias, backlog e DLQ, painel e alertas em Terraform | módulo `observabilidade`; evidências das Sprints 9 e 10 coletadas por API |
| **Custo** | fixo, independente do volume; eficiente em volume alto e constante | proporcional ao uso; muito baixo em volume baixo ou intermitente (o caso de equipes de campo), com orçamento e limite de instâncias como salvaguardas | bootstrap (orçamento); limites da função |
| **Complexidade** | concentrada em uma base de código e um banco; o acoplamento é o preço | distribuída: contrato OpenAPI, eventos versionados, idempotência, reconciliação, DLQ e ferramentas de laboratório para testar tudo isso | ADR-0006, ADR-0009, ADR-0012, ADR-0013 |
| **Portabilidade** | alta (o monólito roda em qualquer servidor) | dependente dos serviços gerenciados do provedor (Cloud Run functions, Firestore, Pub/Sub, API Gateway, Firebase); mitigada pelo contrato e pelas portas do domínio em Go | arquitetura hexagonal do serviço; ADR-0003 |

## 3. Trade-offs das decisões arquiteturais

- **Validação local do adesivo (ADR-0002).** Tirar o central do caminho crítico exigiu levar
  parte da confiança para o aparelho: um QR assinado com chave privada fora da nuvem e a
  contrassenha derivada por HMAC no serviço. O ganho é a independência; o custo é a gestão de
  chaves (rotação por `kid`, validade do adesivo) e a exposição da chave pública no serviço.
- **Mensageria em vez de chamada síncrona ao central (ADR-0013).** Garante que a operação em
  campo não dependa do central, ao preço da consistência eventual: o central vê a validação
  segundos ou minutos depois, e precisa tolerar duplicatas (`at-least-once`) e lidar com a DLQ.
  A ordenação por máquina evita que uma conclusão chegue antes da abertura.
- **Resposta antes da publicação (ADR-0012).** A validação responde ao operador assim que
  persiste, mesmo que o Pub/Sub esteja indisponível; a reconciliação pelo Scheduler republica o
  que ficou pendente. O custo é mais um componente e um atraso de até 5 min na pior hipótese;
  um efeito colateral útil é manter a função aquecida, embora o gateway, chamado só pelo
  aplicativo, não se beneficie (Sprint 10, seção 3 do relatório de escalabilidade).
- **Idempotência por chave do cliente (ADR-0009).** Transforma reenvios em operações seguras e
  dá ao aplicativo uma política de retentativa simples. Exige persistir e indexar a chave, e o
  cenário D4 mostrou que a política precisa cobrir também a conclusão.
- **Serverless com limites explícitos.** A escala automática é real, mas o limite de instâncias
  (controle de custo) e a concorrência por instância são decisões: a configuração padrão da
  plataforma (uma requisição por instância, 1/6 de vCPU) favorece custo baixo e isolamento, e
  limita a vazão; a configuração concorrente (80 por instância, 1 vCPU) usa melhor cada instância
  em Go, à custa de instâncias maiores. A avaliação de escalabilidade mede os dois lados.
- **Cold start.** Inerente ao modelo com piso zero e, na medição da Sprint 10, dominado pelo
  API Gateway (3 a 4,6 s de uma primeira requisição de 6 s), não pela função (1,5 a 2,7 s):
  `min_instancias = 1` elimina a partida do contêiner, mas não a do gateway nem a penalidade de
  ociosidade da própria instância (1,6 a 1,9 s). Esconder o cold start exige tráfego periódico que
  atravesse o gateway, ou dispensar o gateway; a escolha é de operação e custo, não de código
  ([escalabilidade.md](escalabilidade.md), seção 3).
- **Dependência do provedor.** O ecossistema usa cinco serviços gerenciados; trocar de nuvem
  significaria reimplementar adaptadores, não o domínio. É um custo aceito em troca de operação
  mínima para uma equipe pequena.

## 4. Quando o monólito central continua sendo a melhor escolha

- Operação com conectividade garantida e equipe de infraestrutura própria, em que o custo fixo
  já está pago e a latência em regime é prioritária.
- Requisitos de consistência imediata entre a validação e os demais processos do central (por
  exemplo, bloqueio em tempo real de um operador), que a consistência eventual não atende.
- Volume alto e constante, em que o custo por uso supera o custo da capacidade instalada.
- Restrições regulatórias ou contratuais que impeçam serviços gerenciados de terceiros.

## 5. Síntese

A abordagem proposta troca o acoplamento e a simplicidade de um sistema central único por
independência operacional, resiliência definida por cenário e escala sob demanda, pagando com
consistência eventual, mais componentes e dependência do provedor. Para o problema do trabalho,
em que a operação em campo não pode parar quando o central falha e a conectividade é
intermitente, os atributos ganhos são os que importam: os cenários da Sprint 9 mostram a
operação completando sem o central e sem perder eventos, e os testes da Sprint 10 mostram como a
plataforma escala e a que custo em latência. O monólito continua adequado onde essas condições
não valem.
