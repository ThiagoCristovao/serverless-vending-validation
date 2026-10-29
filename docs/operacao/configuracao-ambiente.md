# Configuração do ambiente

Passo a passo para sair de uma máquina limpa até um projeto GCP pronto para receber a
infraestrutura da Sprint 3. Testado em Ubuntu com zsh; ajuste os comandos de shell se usar bash.

## 1. Toolchain local

### 1.1 mise (Go e Terraform)

```bash
curl https://mise.run | sh
echo 'eval "$(~/.local/bin/mise activate zsh)"' >> ~/.zshrc   # ou ~/.bashrc com "bash"
exec $SHELL
cd serverless-vending-validation
mise trust
mise install            # instala as versões de .mise.toml
go version && terraform version
```

### 1.2 FVM e Flutter

O pacote do FVM traz um script `fvm` que depende do diretório `src/` ao lado dele; por isso ele é
extraído inteiro e chamado por um invólucro em `~/.local/bin` (nada é escrito no `.zshrc`).

```bash
URL=$(gh api repos/btwld/fvm/releases/latest --jq '.assets[] | select(.name | test("linux-x64\\.tar\\.gz$")) | .browser_download_url')
curl -sSL "$URL" | tar -xz -C /tmp && rm -rf ~/.local/share/fvm && mv /tmp/fvm ~/.local/share/fvm
printf '#!/bin/sh\nexec "%s/.local/share/fvm/fvm" "$@"\n' "$HOME" > ~/.local/bin/fvm && chmod +x ~/.local/bin/fvm
cd aplicativo
fvm install             # instala a versão de .fvmrc em ~/fvm/versions/
fvm flutter doctor
```

Use `fvm flutter` em vez de `flutter` dentro de `aplicativo/`, ou os alvos `make app-*`.

### 1.3 Android SDK (sem Android Studio)

Basta o SDK por linha de comando; o Java 21 do sistema atende ao Gradle. As licenças são aceitas de
forma não interativa.

```bash
SDK=~/Android/Sdk; mkdir -p $SDK/cmdline-tools
curl -sSL -o /tmp/ct.zip https://dl.google.com/android/repository/commandlinetools-linux-15859902_latest.zip
unzip -q /tmp/ct.zip -d /tmp/ct && mv /tmp/ct/cmdline-tools $SDK/cmdline-tools/latest
yes | $SDK/cmdline-tools/latest/bin/sdkmanager --sdk_root=$SDK --licenses >/dev/null
$SDK/cmdline-tools/latest/bin/sdkmanager --sdk_root=$SDK "platform-tools" "platforms;android-35" "platforms;android-36" "build-tools;35.0.0" "build-tools;36.0.0"
ln -sf $SDK/platform-tools/adb ~/.local/bin/adb
cd aplicativo && fvm flutter config --android-sdk $SDK && fvm flutter doctor
```

No celular: ativar as opções de desenvolvedor e a depuração USB; ao conectar, autorizar o
computador. `adb devices` deve listar o aparelho antes de `make app-executar`.

### 1.4 gcloud CLI

Siga a instalação via apt da
[documentação oficial](https://cloud.google.com/sdk/docs/install#deb). Depois:

```bash
gcloud init
gcloud auth login
gcloud auth application-default login   # credenciais usadas pelo Terraform
```

### 1.5 Alternativa sem instalação (só verificação)

Com Docker instalado, `make verificar` funciona sem `terraform` nem `redocly` locais: o Makefile
usa as imagens `hashicorp/terraform` e `redocly/cli`. Essa alternativa **não** serve para aplicar
infraestrutura.

## 2. Projeto GCP (criado pelo console do Firebase)

Crie o projeto **pelo console do Firebase**, não pelo `gcloud` (ADR-0010). Projetos criados pela
CLI não apareceram como elegíveis ao Firebase e a habilitação via API respondeu `403` mesmo com
todas as permissões presentes e sem políticas de organização restritivas.

1. Em <https://console.firebase.google.com>, clique em "Adicionar projeto", digite o nome
   (ex.: `svv-dev`), aceite ou ajuste o ID sugerido, mantenha a organização padrão e **desative o
   Google Analytics**.
2. Anote o ID do projeto e siga:

```bash
export PROJETO_ID="svv-dev"
gcloud billing accounts list                            # copie o ACCOUNT_ID
gcloud billing projects link "$PROJETO_ID" --billing-account="XXXXXX-XXXXXX-XXXXXX"
gcloud config set project "$PROJETO_ID"
gcloud auth application-default set-quota-project "$PROJETO_ID"
gcloud services enable serviceusage.googleapis.com cloudresourcemanager.googleapis.com
```

O console já habilita o Firebase; o bootstrap adota o projeto Firebase por um bloco `import` e
ativa o Identity Platform. Como essas duas habilitações são irreversíveis, ficam no bootstrap e não
no ambiente, que assim passa por `destroy` + `apply` sem passos manuais.

## 3. Bootstrap (uma vez por projeto)

```bash
cp infra/bootstrap/terraform.tfvars.example infra/bootstrap/terraform.tfvars
# editar: projeto_id, bucket_estado_nome (único no mundo), conta_faturamento_id, orcamento_*
make infra-bootstrap-init
make infra-bootstrap-plan
make infra-bootstrap-apply
terraform -chdir=infra/bootstrap output
make infra-bootstrap-salvar-estado
```

O bootstrap habilita as APIs, cria o bucket de estado com versionamento e um orçamento mensal
com alertas em 50 %, 90 % e 100 %. O `infra/bootstrap/terraform.tfstate` fica fora do git; após cada
apply, copie-o para o bucket com `make infra-bootstrap-salvar-estado`.

## 4. Ambiente de desenvolvimento

```bash
cp infra/ambientes/dev/backend.hcl.example infra/ambientes/dev/backend.hcl
cp infra/ambientes/dev/terraform.tfvars.example infra/ambientes/dev/terraform.tfvars
# editar: bucket (saída do bootstrap) e projeto_id
make infra-dev-init
make infra-dev-plan        # sem alterações até a Sprint 3
```

## 5. Firebase

O projeto já nasce habilitado no Firebase (passo 2). O Identity Platform (e-mail e senha) é ativado
pelo bootstrap (passo 3); o registro do app Android e as regras do Firestore são criados no passo 4. O
`google-services.json` do aplicativo é gerado com `make app-google-services` (Sprint 6).

## 6. Testar o serviço implantado (Sprint 5)

Com o ambiente aplicado, prepare dados de teste e chame o gateway. Os arquivos gerados ficam em
`chaves/` (ignorado pelo git).

```bash
make app-google-services                 # aplicativo/android/app/google-services.json (traz a chave de API web)
cd servico
go run ./cmd/gerar-qr -gerar-chaves -saida ../chaves
go run ./cmd/token-operador -google-services ../aplicativo/android/app/google-services.json \
   -email operador1@exemplo.invalid -senha '<senha forte>' -criar > ../chaves/operador1.json
go run ./cmd/semear-firestore -projeto svv-dev -operador-uid "$(python3 -c "import json;print(json.load(open('../chaves/operador1.json'))['uid'])")" \
   -operador-email operador1@exemplo.invalid -chave-publica ../chaves/chave-publica-qr.txt
go run ./cmd/gerar-qr -chave-privada ../chaves/chave-privada-qr.txt -png ../chaves/vm-2047.png > ../chaves/qr-vm-2047.json
cd ..
HOST=$(terraform -chdir=infra/ambientes/dev output -raw gateway_hostname)
TOKEN=$(python3 -c "import json;print(json.load(open('chaves/operador1.json'))['idToken'])")
curl -s -X POST "https://$HOST/v1/validacoes" -H "Authorization: Bearer $TOKEN" \
   -H "Idempotency-Key: $(python3 -c 'import uuid;print(uuid.uuid4())')" -H 'Content-Type: application/json' \
   -d "{\"qr\":$(cat chaves/qr-vm-2047.json)}"
```

A resposta `201` traz a contrassenha corrente e a próxima. O ID token vale uma hora; repita o
`token-operador` (sem `-criar`) para renovar. Para ver as mensagens publicadas:
`gcloud pubsub subscriptions pull svv-dev-sistema-central --auto-ack --limit=10`. Como a assinatura
tem ordenação por `idMaquina`, uma mensagem não confirmada segura as seguintes da mesma máquina:
sem `--auto-ack`, cada `pull` devolve só a próxima da fila; repita o comando até esvaziar.

## 7. Ecossistema completo e suíte de ponta a ponta (Sprint 8)

O sistema central simulado consome a assinatura `svv-dev-sistema-central` e expõe uma API
administrativa local (porta 8090). A suíte e2e sobe o consumidor dentro do próprio processo de
teste, por isso **não** execute `make central-executar` ao mesmo tempo que `make e2e`: os dois
disputariam as mesmas mensagens.

```bash
make e2e-preparar          # uma vez: semeia VM-E2E1 e VM-E2E2 (localização E2E-L) no Firestore
printf '%s' '<senha do operador de testes>' > chaves/operador1.senha && chmod 600 chaves/operador1.senha
make e2e                   # ~5 min; o cenário de DLQ espera as reentregas (10 s → 600 s)
make e2e E2E_ARGS=-short   # pula o cenário de DLQ
```

A suíte usa o gateway (saída `gateway_hostname` do Terraform), a chave de API web do
`google-services.json`, o operador de testes e a chave privada dos adesivos em `chaves/`. Ao
final imprime o resumo dos contadores do consumidor e o caminho do arquivo JSON Lines com os
eventos recebidos.

Para operar o consumidor manualmente (demonstrações e Sprint 9):

```bash
make central-executar                                   # consome svv-dev-sistema-central
curl -s localhost:8090/saude
curl -s 'localhost:8090/eventos?idMaquina=VM-2047'
curl -s -X PUT localhost:8090/admin/falhas -d '{"modo":"rejeitar","idMaquina":"VM-2047"}'
curl -s -X POST localhost:8090/admin/pausar             # indisponibilidade do sistema central
curl -s -X POST localhost:8090/admin/retomar
```

Os eventos ficam em `sistema-central-simulado/dados/eventos.jsonl` (ignorado pelo git); apague o
arquivo para recomeçar do zero.

### 7.1 Injeção de falhas (Sprint 9)

Os cenários do lado do serviço rodam sem o celular e gravam o relatório em
`docs/avaliacao/evidencias/<cenario>.md`; os dois últimos removem e restauram um vínculo IAM da
conta da função (ADR-0014). Se um deles for interrompido, `CENARIO=restaurar` repara.

```bash
make experimento CENARIO=central-indisponivel      # pausa o consumidor por 3 min (EXPERIMENTO_ARGS="-indisponibilidade 5m")
make experimento CENARIO=publicacao-indisponivel   # remove pubsub.publisher da função; espera a reconciliação
make experimento CENARIO=firestore-indisponivel    # remove datastore.user da função; 503 e recuperação
make experimento CENARIO=restaurar
```

Os cenários do dispositivo (rede intermitente e autenticação indisponível) usam o proxy e o
bloqueio DNS de `ferramentas/injecao-falhas/` (`make injecao-proxy`, `make injecao-dns`,
`make app-apk-injecao`); o roteiro está no README daquele diretório. Para rodar com a identidade da conta de serviço do consumidor,
use `go run ./cmd/central -impersonar svv-dev-central-simulado@svv-dev.iam.gserviceaccount.com`
(exige `roles/iam.serviceAccountTokenCreator` na conta).

### 7.2 Testes de carga e cold start (Sprint 10)

```bash
make carga-preparar OPERADORES=100       # uma vez: 100 contas carga-NNN@exemplo.invalid e máquinas VM-CARGA-NNN (chaves/carga.senha, chaves/carga-lote.json)
make carga OPERADORES=10 CONFIGURACAO=A  # ~5 min: rampa 30 s, 2 min em regime, 2 min de espera pelas métricas
make carga OPERADORES=50 CONFIGURACAO=A
make carga OPERADORES=100 CONFIGURACAO=A
make infra-dev-apply TF_VAR_funcao_concorrencia=80 TF_VAR_funcao_cpu=1 TF_VAR_funcao_max_instancias=100   # configuração C
make carga OPERADORES=10 CONFIGURACAO=C  # e 50, 100; depois make infra-dev-apply sem as variáveis para voltar à configuração A
make carga-frio CONFIGURACAO=min0        # ~2 h: pausa o agendador, 6 amostras após 20 min de inatividade, retoma o agendador
make infra-dev-apply TF_VAR_funcao_min_instancias=1 && make carga-frio CONFIGURACAO=min1 && make infra-dev-apply
```

Os relatórios vão para `docs/avaliacao/evidencias/carga-<conf>-<n>.md` e
`cold-start-<conf>.md`; a consolidação está em `docs/avaliacao/escalabilidade.md`. As variáveis
`TF_VAR_*` são lidas pelo Terraform e não ficam em arquivo; a configuração padrão volta ao
aplicar sem elas. Se um `carga-frio` for interrompido, retome o job
`svv-dev-reconciliar-publicacoes` no Cloud Scheduler.

## 8. Custos

Os serviços usados têm camada gratuita generosa para o volume deste trabalho. Os pontos de
atenção são os testes de carga (Sprint 10) e instâncias mínimas da função, se habilitadas. O
orçamento do bootstrap avisa por e-mail; ele **não** bloqueia gastos.

## 9. Verificação

```bash
make verificar
```

Deve terminar sem erros. Na CI o mesmo conjunto roda em cada PR, sem credenciais GCP.

## 10. Problemas comuns

| Sintoma | Causa provável | Ação |
|---|---|---|
| `Error 403: ... serviceusage.services.enable` | Conta sem papel de Owner/Editor no projeto ou projeto sem faturamento | Conferir `gcloud projects get-iam-policy` e o vínculo de faturamento |
| `googleapi: Error 409: ... bucket ... already exists` | Nome de bucket em uso por outro projeto no mundo | Trocar `bucket_estado_nome` |
| API "not enabled" logo após o bootstrap | Propagação da habilitação leva alguns minutos | Aguardar e repetir |
| `terraform init` falha em `ambientes/dev` | `backend.hcl` ausente ou bucket errado | Conferir a saída `backend_hcl_sugerido` do bootstrap |
| Terraform via Docker cria arquivos como root | Fallback Docker sem `-u` | O Makefile já passa `-u $(id -u):$(id -g)`; apague `.terraform/` e repita |
| `Error 403: ... requires a quota project` ao criar o orçamento | Provider sem `user_project_override` e `billing_project` com credenciais de usuário | Já corrigido em `infra/bootstrap/main.tf`; se aparecer em outra API, adicionar as mesmas duas linhas ao provider |
| Projeto criado via `gcloud` não aparece em "Adicionar projeto" do Firebase; `google_firebase_project` falha com `403 The caller does not have permission` | Elegibilidade do projeto no Firebase (causa não documentada pelo Google; permissões e políticas estavam corretas) | Criar o projeto pelo console do Firebase e vincular o faturamento depois (ADR-0010) |
| `Database ID '(default)' is not available ... retry in N seconds` logo após um `destroy` | O Firestore reserva o ID do banco excluído por ~5 minutos | Aguardar e repetir `make infra-dev-apply` |
| `make e2e` falha em `abrir validação` com `409` repetido | Validação aberta na máquina e2e por outro operador | Esperar 15 min (RN-04) ou concluir pelo operador que a abriu |
| `make e2e` falha em `token do operador` com `INVALID_LOGIN_CREDENTIALS` | `chaves/operador1.senha` ausente ou errada | Regravar o arquivo com a senha do operador de testes |
| Cenário de DLQ da e2e estoura o prazo | As reentregas respeitam a espera exponencial da assinatura; raramente passam de 5 min | Aumentar `SVV_E2E_PRAZO_DLQ` (ex.: `15m`) ou rodar com `E2E_ARGS=-short` |
| `pull` da assinatura devolve uma mensagem só (ou nenhuma) embora o serviço tenha publicado várias | Ordenação por chave: a mensagem anterior da mesma máquina ainda não foi confirmada, ou o `pull` voltou vazio antes de o servidor liberar a próxima | Confirmar (`ack`) e repetir o `pull` algumas vezes; `--auto-ack` faz isso de uma vez |
| `Reauthentication failed. cannot prompt during non-interactive execution` em qualquer `gcloud` | A conta do Workspace (`alunos.utfpr.edu.br`) tem política de sessão; a credencial de usuário expira periodicamente. As credenciais do Terraform (ADC) expiram separadamente | Repetir `gcloud auth login` (e, se o Terraform também falhar, `gcloud auth application-default login`) |
