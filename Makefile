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
        servico-integracao servico-local emuladores-subir emuladores-parar \
        app-analisar app-testar app-verificar app-apk app-executar \
        central-formatar central-formatar-verificar central-verificar central-executar e2e-preparar e2e \
        experimento injecao-formatar-verificar injecao-verificar injecao-proxy injecao-dns app-apk-injecao \
        infra-bootstrap-init infra-bootstrap-plan infra-bootstrap-apply infra-bootstrap-salvar-estado \
        infra-dev-init infra-dev-plan infra-dev-apply segredo-hmac-gerar app-google-services

# Ambiente dev usado pelo consumidor local e pela suíte e2e (Sprint 8).
PROJETO_ID ?= $(shell gcloud config get-value project 2>/dev/null)
PREFIXO ?= svv-dev
E2E_OPERADOR_EMAIL ?= operador1@exemplo.invalid
# Variáveis SVV_E2E_* usadas pela suíte e2e e pelos experimentos (rodam em sistema-central-simulado/).
E2E_AMBIENTE = SVV_E2E_PROJETO="$(PROJETO_ID)" SVV_E2E_PREFIXO="$(PREFIXO)" SVV_E2E_GATEWAY="https://$$($(TERRAFORM) -chdir=../infra/ambientes/dev output -raw gateway_hostname)/v1" SVV_E2E_GOOGLE_SERVICES="$(CURDIR)/aplicativo/android/app/google-services.json" SVV_E2E_OPERADOR_EMAIL="$(E2E_OPERADOR_EMAIL)" SVV_E2E_OPERADOR_SENHA="$$(cat "$(CURDIR)/chaves/operador1.senha")" SVV_E2E_CHAVE_PRIVADA="$(CURDIR)/chaves/chave-privada-qr.txt"
# Injeção de falhas no dispositivo (Sprint 9).
IP_LOCAL ?= $(shell ip -4 route get 1.1.1.1 2>/dev/null | awk '{print $$7; exit}')
MODO ?= normal
CENARIO ?= central-indisponivel
# Testes de carga (Sprint 10).
OPERADORES ?= 10
CONFIGURACAO ?= A
CARGA_AMBIENTE = $(E2E_AMBIENTE) SVV_CARGA_SENHA="$$(cat "$(CURDIR)/chaves/carga.senha")" SVV_CARGA_LOTE="$(CURDIR)/chaves/carga-lote.json"

ajuda: ## Lista os alvos disponíveis
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-28s\033[0m %s\n", $$1, $$2}'

configurar: ## Instala o toolchain fixado (mise) e o Flutter (FVM)
	mise install
	cd aplicativo && fvm install

verificar: infra-formatar-verificar infra-validar api-lint servico-formatar-verificar servico-verificar central-formatar-verificar central-verificar injecao-formatar-verificar injecao-verificar app-verificar ## Executa todas as verificações locais (o mesmo que a CI)

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

servico-local: ## Executa o serviço localmente com dados em memória (porta 8080)
	cd servico && go run ./cmd/local

emuladores-subir: ## Sobe os emuladores do Firestore (8081) e do Pub/Sub (8085) em Docker
	docker compose -f ferramentas/emuladores/docker-compose.yml up -d

emuladores-parar: ## Para os emuladores
	docker compose -f ferramentas/emuladores/docker-compose.yml down

servico-integracao: ## Testes de integração contra os emuladores (exige make emuladores-subir)
	cd servico && FIRESTORE_EMULATOR_HOST=localhost:8081 PUBSUB_EMULATOR_HOST=localhost:8085 go test -tags integracao ./internal/adaptadores/... -count=1

central-formatar: ## Formata o código Go do sistema central simulado
	cd sistema-central-simulado && gofmt -w .

central-formatar-verificar: ## Falha se houver Go do sistema central simulado fora do padrão gofmt
	@cd sistema-central-simulado && fora="$$(gofmt -l .)"; if [ -n "$$fora" ]; then echo "fora do gofmt:"; echo "$$fora"; exit 1; fi

central-verificar: ## go vet (inclusive a suíte e2e) e testes do sistema central simulado (sem rede: Pub/Sub falso em processo)
	cd sistema-central-simulado && go vet ./... && go vet -tags e2e ./e2e/ && go test ./... -count=1

central-executar: ## Executa o sistema central simulado contra a assinatura do ambiente dev (API administrativa em :8090)
	cd sistema-central-simulado && go run ./cmd/central -projeto "$(PROJETO_ID)" -assinatura "$(PREFIXO)-sistema-central"

e2e-preparar: ## Semeia as máquinas VM-E2E1 e VM-E2E2 no Firestore do ambiente dev (uma vez; exige chaves/)
	@for m in VM-E2E1 VM-E2E2; do \
		(cd servico && go run ./cmd/semear-firestore -projeto "$(PROJETO_ID)" -maquina $$m -loc E2E-L -loc-nome "Laboratório de testes e2e" \
			-operador-uid "$$(python3 -c "import json;print(json.load(open('../chaves/operador1.json'))['uid'])")" \
			-operador-email "$(E2E_OPERADOR_EMAIL)" -chave-publica ../chaves/chave-publica-qr.txt) || exit 1; \
	done

e2e: ## Suíte de ponta a ponta contra o ambiente dev, com o consumidor em processo (exige chaves/ e make e2e-preparar; ~5 min)
	cd sistema-central-simulado && $(E2E_AMBIENTE) go test -tags e2e ./e2e/ -count=1 -v -timeout 25m $(E2E_ARGS)

experimento: ## Cenário de injeção de falhas contra o ambiente dev, com relatório em docs/avaliacao/evidencias (CENARIO=central-indisponivel|publicacao-indisponivel|firestore-indisponivel|restaurar)
	cd sistema-central-simulado && $(E2E_AMBIENTE) go run ./cmd/experimento -cenario "$(CENARIO)" -saida "$(CURDIR)/docs/avaliacao/evidencias/$(CENARIO).md" $(EXPERIMENTO_ARGS)

carga-preparar: ## Cria OPERADORES=100 operadores virtuais (Firebase Auth) e máquinas VM-CARGA-NNN no Firestore do dev (uma vez; exige chaves/)
	@test -s chaves/carga.senha || (openssl rand -base64 18 | tr -d '\n' > chaves/carga.senha && chmod 600 chaves/carga.senha && echo "senha dos operadores de carga gerada em chaves/carga.senha")
	cd sistema-central-simulado && $(CARGA_AMBIENTE) go run ./cmd/carga -cenario preparar -operadores "$(OPERADORES)"
	cd servico && go run ./cmd/semear-firestore -projeto "$(PROJETO_ID)" -lote ../chaves/carga-lote.json -chave-publica ../chaves/chave-publica-qr.txt

carga: ## Teste de carga com OPERADORES=10|50|100 operadores e CONFIGURACAO=A|C, relatório em docs/avaliacao/evidencias/carga-<conf>-<n>.md (CARGA_ARGS="-duracao 2m -rampa 30s -pausa 1s")
	cd sistema-central-simulado && $(CARGA_AMBIENTE) go run ./cmd/carga -cenario carga -operadores "$(OPERADORES)" -configuracao "$(CONFIGURACAO)" -saida "$(CURDIR)/docs/avaliacao/evidencias/carga-$(CONFIGURACAO)-$(OPERADORES).md" $(CARGA_ARGS)

carga-frio: ## Cold start após inatividade (pausa o agendador; ~2 h) com CONFIGURACAO=min0|min1, relatório em docs/avaliacao/evidencias/cold-start-<conf>.md
	cd sistema-central-simulado && $(E2E_AMBIENTE) go run ./cmd/carga -cenario frio -configuracao "$(CONFIGURACAO)" -saida "$(CURDIR)/docs/avaliacao/evidencias/cold-start-$(CONFIGURACAO).md" $(CARGA_ARGS)

injecao-formatar-verificar: ## Falha se houver Go das ferramentas de injeção de falhas fora do padrão gofmt
	@cd ferramentas/injecao-falhas && fora="$$(gofmt -l .)"; if [ -n "$$fora" ]; then echo "fora do gofmt:"; echo "$$fora"; exit 1; fi

injecao-verificar: ## go vet e testes das ferramentas de injeção de falhas
	cd ferramentas/injecao-falhas && go vet ./... && go test ./... -count=1

injecao-proxy: ## Proxy de falhas de rede para o celular na porta 8099 (MODO=normal|cortar|atrasar|indisponivel|cortar-resposta|intermitente)
	cd ferramentas/injecao-falhas && go run ./cmd/proxy-rede -destino "https://$$($(TERRAFORM) -chdir=../../infra/ambientes/dev output -raw gateway_hostname)" -modo "$(MODO)"

injecao-dns: ## Bloqueia a resolução do Firebase Authentication para o celular (dnsmasq na porta 53; pede sudo)
	ferramentas/injecao-falhas/dns-bloqueio.sh

app-apk-injecao: ## APK de depuração apontando para o proxy de falhas neste computador (IP_LOCAL=$(IP_LOCAL))
	cd aplicativo && fvm flutter build apk --debug --dart-define=SVV_API_BASE=http://$(IP_LOCAL):8099/v1

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

app-analisar: ## Análise estática do aplicativo Flutter (via FVM)
	cd aplicativo && fvm flutter analyze

app-testar: ## Testes de unidade e de widget do aplicativo
	cd aplicativo && fvm flutter test

app-verificar: ## Formatação, análise e testes do aplicativo (o mesmo que a CI)
	cd aplicativo && fvm dart format --output=none --set-exit-if-changed lib test && fvm flutter analyze && fvm flutter test

app-apk: ## Compila o APK de depuração (build/app/outputs/flutter-apk/app-debug.apk)
	cd aplicativo && fvm flutter build apk --debug

app-executar: ## Instala e executa no dispositivo Android conectado (adb devices)
	cd aplicativo && fvm flutter run
