# Makefile — atalhos para o dia a dia do projeto.
# Alvos em português, conforme docs/operacao/convencoes.md.
# `make` ou `make ajuda` lista os alvos.
#
# Quando terraform/redocly não estão instalados, os alvos de verificação usam
# imagens Docker como alternativa. Essa alternativa serve APENAS para
# fmt/validate/lint: os alvos que aplicam infraestrutura precisam do Terraform
# local autenticado via gcloud (Application Default Credentials).

SHELL := /bin/bash
.DEFAULT_GOAL := ajuda

TERRAFORM_VERSAO := 1.16.3
INFRA_DIRS := infra/bootstrap infra/ambientes/dev
OPENAPI := api/openapi.yaml

ifeq (,$(shell command -v terraform 2>/dev/null))
  TERRAFORM := docker run --rm -u "$(shell id -u):$(shell id -g)" -e HOME=/tmp -v "$(CURDIR)":/work -w /work hashicorp/terraform:$(TERRAFORM_VERSAO)
else
  TERRAFORM := terraform
endif

ifeq (,$(shell command -v redocly 2>/dev/null))
  ifeq (,$(shell command -v npx 2>/dev/null))
    REDOCLY := docker run --rm -u "$(shell id -u):$(shell id -g)" -v "$(CURDIR)":/spec redocly/cli
  else
    REDOCLY := npx --yes @redocly/cli@latest
  endif
else
  REDOCLY := redocly
endif

.PHONY: ajuda configurar verificar infra-formatar infra-formatar-verificar infra-validar api-lint \
        servico-formatar servico-formatar-verificar servico-verificar servico-cobertura \
        infra-bootstrap-init infra-bootstrap-plan infra-bootstrap-apply infra-bootstrap-salvar-estado \
        infra-dev-init infra-dev-plan infra-dev-apply segredo-hmac-gerar app-google-services

ajuda: ## Lista os alvos disponíveis
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-28s\033[0m %s\n", $$1, $$2}'

configurar: ## Instala o toolchain fixado (mise) e o Flutter (FVM)
	mise install
	cd aplicativo && fvm install

verificar: infra-formatar-verificar infra-validar api-lint servico-formatar-verificar servico-verificar ## Executa todas as verificações locais (o mesmo que a CI)

infra-formatar: ## Formata os arquivos Terraform
	$(TERRAFORM) fmt -recursive infra

infra-formatar-verificar: ## Falha se houver Terraform fora do padrão de formatação
	$(TERRAFORM) fmt -check -diff -recursive infra

infra-validar: ## Valida a sintaxe dos diretórios Terraform (sem backend, sem credenciais)
	@for d in $(INFRA_DIRS); do \
		echo ">> $$d"; \
		$(TERRAFORM) -chdir=$$d init -backend=false -input=false >/dev/null && \
		$(TERRAFORM) -chdir=$$d validate || exit 1; \
	done

api-lint: ## Valida o contrato OpenAPI
	$(REDOCLY) lint $(OPENAPI)

servico-formatar: ## Formata o código Go do serviço
	cd servico && gofmt -w .

servico-formatar-verificar: ## Falha se houver Go fora do padrão gofmt
	@cd servico && fora="$$(gofmt -l .)"; if [ -n "$$fora" ]; then echo "fora do gofmt:"; echo "$$fora"; exit 1; fi

servico-verificar: ## go vet e testes de unidade do serviço (rodam sem rede)
	cd servico && go vet ./... && go test ./... -count=1

servico-cobertura: ## Testes do serviço com relatório de cobertura por pacote
	cd servico && go test ./... -count=1 -coverpkg=./... -coverprofile=coverage.out && go tool cover -func=coverage.out | tail -1

infra-bootstrap-init: ## Inicializa o bootstrap (estado local, roda uma vez por projeto)
	$(TERRAFORM) -chdir=infra/bootstrap init -input=false

infra-bootstrap-plan: ## Planeja o bootstrap
	$(TERRAFORM) -chdir=infra/bootstrap plan -input=false

infra-bootstrap-apply: ## Aplica o bootstrap (APIs, bucket de estado, orçamento)
	$(TERRAFORM) -chdir=infra/bootstrap apply -input=false

infra-bootstrap-salvar-estado: ## Copia o estado local do bootstrap para o bucket de estado (backup versionado)
	gcloud storage cp infra/bootstrap/terraform.tfstate "gs://$$($(TERRAFORM) -chdir=infra/bootstrap output -raw bucket_estado_nome)/bootstrap/terraform.tfstate"

infra-dev-init: ## Inicializa o ambiente dev contra o backend GCS (requer infra/ambientes/dev/backend.hcl)
	$(TERRAFORM) -chdir=infra/ambientes/dev init -input=false -backend-config=backend.hcl

infra-dev-plan: ## Planeja o ambiente dev
	$(TERRAFORM) -chdir=infra/ambientes/dev plan -input=false

infra-dev-apply: ## Aplica o ambiente dev
	$(TERRAFORM) -chdir=infra/ambientes/dev apply -input=false

segredo-hmac-gerar: ## Gera uma chave HMAC aleatória (32 bytes, hex) e a adiciona como nova versão do segredo (fora do Terraform)
	openssl rand -hex 32 | tr -d '\n' | gcloud secrets versions add "$$($(TERRAFORM) -chdir=infra/ambientes/dev output -raw segredo_hmac_nome)" --data-file=-

app-google-services: ## Grava aplicativo/android/app/google-services.json a partir da saída do Terraform (arquivo ignorado pelo git)
	mkdir -p aplicativo/android/app
	$(TERRAFORM) -chdir=infra/ambientes/dev output -raw google_services_json > aplicativo/android/app/google-services.json
	@echo "google-services.json gravado"
