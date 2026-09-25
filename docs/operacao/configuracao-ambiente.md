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

```bash
curl -fsSL https://fvm.app/install.sh | bash
cd aplicativo
fvm install             # instala a versão de .fvmrc
fvm flutter doctor
```

Para Android é necessário o Android SDK (via Android Studio ou `cmdline-tools`), aceitar as licenças
com `fvm flutter doctor --android-licenses` e habilitar a depuração USB no dispositivo físico.
Use `fvm flutter` em vez de `flutter` dentro de `aplicativo/`.

### 1.3 gcloud CLI

Siga a instalação via apt da
[documentação oficial](https://cloud.google.com/sdk/docs/install#deb). Depois:

```bash
gcloud init
gcloud auth login
gcloud auth application-default login   # credenciais usadas pelo Terraform
```

### 1.4 Alternativa sem instalação (só verificação)

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

## 6. Custos

Os serviços usados têm camada gratuita generosa para o volume deste trabalho. Os pontos de
atenção são os testes de carga (Sprint 10) e instâncias mínimas da função, se habilitadas. O
orçamento do bootstrap avisa por e-mail; ele **não** bloqueia gastos.

## 7. Verificação

```bash
make verificar
```

Deve terminar sem erros. Na CI o mesmo conjunto roda em cada PR, sem credenciais GCP.

## 8. Problemas comuns

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
| `Reauthentication failed. cannot prompt during non-interactive execution` em qualquer `gcloud` | A conta do Workspace (`alunos.utfpr.edu.br`) tem política de sessão; a credencial de usuário expira periodicamente. As credenciais do Terraform (ADC) expiram separadamente | Repetir `gcloud auth login` (e, se o Terraform também falhar, `gcloud auth application-default login`) |
