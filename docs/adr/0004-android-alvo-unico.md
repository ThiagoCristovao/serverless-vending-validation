# ADR-0004: Android como única plataforma-alvo do aplicativo nesta versão

- **Status:** aceita (informar ao orientador)
- **Data:** 2026-09-16
- **Decisores:** Thiago Cristovão de Souza
- **Sprint:** 0

## Contexto

O TCC 1 justifica o Flutter pela geração de aplicações para Android e iOS a partir de uma base
única. Compilar, assinar e testar para iOS exige um Mac com Xcode e uma conta de desenvolvedor
Apple; o autor não dispõe de Mac. Há um dispositivo Android físico disponível, requisito para
verificar a leitura de QR code.

## Decisão

O aplicativo é criado com `--platforms android` e testado apenas em Android. O código Dart segue
independente de plataforma; nenhuma dependência exclusiva de Android é introduzida sem ADR.

## Alternativas consideradas

- **Build iOS em nuvem (Codemagic, GitHub Actions com runner macOS)** — resolve a compilação, mas
  não o teste em dispositivo físico nem a conta Apple. Custo e tempo não se justificam para o
  escopo. Descartada.
- **Adiar a decisão** — deixaria o `flutter create` com iOS configurado sem nunca ser exercitado,
  gerando arquivos mortos no repositório. Descartada.

## Consequências

### Positivas

- Menos superfície de configuração (assinatura, permissões, Firebase para iOS).
- Foco do tempo escasso no fluxo do operador.

### Negativas e riscos

- A monografia deve enquadrar o suporte a iOS como trabalho futuro, mantendo o argumento de
  portabilidade do Flutter no nível de código.

## Referências

- TCC 1, Seções 1.3 e 2.6.
