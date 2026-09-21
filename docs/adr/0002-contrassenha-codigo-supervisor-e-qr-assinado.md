# ADR-0002: Contrassenha como código supervisor rotacionado e QR estático assinado

- **Status:** proposta (validar com orientador)
- **Data:** 2026-09-16
- **Decisores:** Thiago Cristovão de Souza; Prof. Muriel de Souza Godoi
- **Sprint:** 1

## Contexto

O TCC 1 (Seção 3.2) prevê que o serviço devolva uma "contrassenha" que o operador digita na
máquina, e que o QR carregue "um token de validação gerado periodicamente". Ficou em aberto o que a
contrassenha significa para uma Crane National 168, equipamento offline com painel mecânico, e
como um adesivo impresso poderia carregar um token que muda com o tempo. O autor não tem acesso a
uma máquina física.

O manual de programação da linha National 167/168/457/458 (Crane Merchandising Systems) documenta
que o acesso ao modo supervisor exige um **código numérico de quatro dígitos** digitado no teclado
em até seis segundos após o prompt `ENTER CODE`, que esse código é **configurável** pelo menu
"Enter a New Supervisor Code", e que o modo supervisor exibe vendas totais, contagem de vendas por
seleção e conteúdo do cofre. Esses são exatamente os dados que o TCC 1 chama de "leituras
operacionais".

## Decisão

1. **A contrassenha é o código supervisor corrente da máquina.** O serviço é a fonte da verdade
   desse código e só o revela a um operador autenticado que leu o QR daquela máquina. Sem o
   aplicativo, o operador não entra no modo supervisor.
2. **Derivação determinística, sem armazenar o código em claro:**
   `codigo = HMAC-SHA256(chave_servico, idMaquina || ":" || contador)`; tomam-se os quatro primeiros
   bytes como inteiro sem sinal big-endian, módulo 10 000, com zeros à esquerda. A chave fica no
   Secret Manager; o `contador` por máquina fica no Firestore.
3. **Rotação por visita.** A resposta da validação traz o código corrente e o **próximo**. No modo
   supervisor, o operador lê os medidores e programa o próximo código. Na confirmação de conclusão,
   informa `codigoRotacionado`; se verdadeiro, o serviço incrementa o contador. Se falso, o contador
   não muda e a próxima validação devolve os mesmos códigos.
4. **O QR é um adesivo estático assinado:** `{"v":1,"maq":…,"mod":…,"loc":…,"exp":…,"kid":…,"sig":…}`,
   com assinatura **Ed25519** sobre os campos canônicos, `exp` como validade do adesivo e `kid`
   como versão da chave. O aplicativo valida estrutura e validade localmente (e, desejável, a
   assinatura, com a chave pública embutida); o serviço valida tudo novamente.
5. O "token gerado periodicamente" do TCC 1 é reinterpretado como assinatura + validade + versão
   de chave. A rotação acontece ao reimprimir adesivos com um novo `kid`.

## Alternativas consideradas

- **Máquina não verifica nada; contrassenha é apenas um comprovante registrado** — simples, mas o
  aplicativo deixaria de ser necessário à operação, o que enfraquece o argumento de resiliência
  do trabalho. Descartada, embora o registro de comprovante continue existindo como efeito
  colateral (a validação fica persistida).
- **Máquina verifica por algoritmo compartilhado (estilo TOTP)** — exigiria firmware ou módulo
  acoplado; impossível sem acesso ao equipamento e fora do escopo declarado no TCC 1. Descartada.
- **Armazenar um código aleatório por máquina em vez de derivar por HMAC** — igualmente seguro,
  mas deixaria o código em claro no banco e não daria ao domínio um algoritmo testável. Descartada.
- **Assinar o QR com HMAC em vez de Ed25519** — o aplicativo precisaria do segredo para verificar
  localmente, o que é inaceitável em um binário distribuído. Descartada.
- **Token com validade curta no QR** — impraticável com adesivo impresso em máquina offline.
  Descartada.

## Consequências

### Positivas

- A contrassenha tem semântica real, apoiada no manual do equipamento, sem modificar a máquina.
- O domínio (Sprint 4) ganha regras testáveis: derivação, rotação, validade do adesivo, assinatura
  inválida, código malformado.
- Sem segredos no aplicativo; sem código em claro no banco.
- Independe de conectividade da máquina, coerente com o caso de estudo.

### Negativas e riscos

- Um código de quatro dígitos tem baixa entropia; a segurança vem do controle de acesso ao
  serviço e da rotação, não do código. Deve ser dito explicitamente na monografia.
- Divergência de estado se o operador confirmar `codigoRotacionado = true` sem ter programado o
  código: a máquina fica com o código antigo e o serviço avança. Caminho de recuperação
  (informar "código não aceito" e o serviço devolver o código anterior) fica como extensão.
- O manual não foi verificado quanto a restrições de código (ex.: zeros à esquerda ou códigos
  reservados). **TODO (Sprint 1):** ler as seções "Enter a New Supervisor Code" e "Enter a
  Freevend Code" do manual e registrar restrições.
- Desvio de redação em relação ao TCC 1; precisa ser aceito pelo orientador.

## Referências

- Crane Merchandising Systems. *National 167/168/457/458/764/765/784/787/797/798 Programming
  Guide*. Disponível em:
  <https://www.amequipmentsales.com/manuals/pdf/National/NATIONAL%20167%20168%20457%20458%20764%20765%20784%20787%20797%20798%20PROGRAMMING.pdf>.
  Acesso em 16 set. 2026.
- TCC 1, Seções 3.2 e 3.4.4; `docs/payload-qr.md`; `docs/requisitos.md` (RN-01 a RN-06).
- RFC 8032 (EdDSA / Ed25519); RFC 2104 (HMAC).
