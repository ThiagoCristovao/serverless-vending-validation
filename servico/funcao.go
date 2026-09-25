// Package servico é o ponto de entrada da Cloud Run function. O Functions
// Framework exige que a função seja registrada em init(), no pacote raiz do
// módulo; as dependências de nuvem são montadas na primeira requisição, para
// que o arranque a frio fique curto e o pacote compile sem credenciais.
package servico

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sync"

	"cloud.google.com/go/firestore"
	"cloud.google.com/go/pubsub/v2"
	"github.com/GoogleCloudPlatform/functions-framework-go/functions"

	"github.com/ThiagoCristovao/serverless-vending-validation/servico/internal/adaptadores/apihttp"
	"github.com/ThiagoCristovao/serverless-vending-validation/servico/internal/adaptadores/armazenamento"
	"github.com/ThiagoCristovao/serverless-vending-validation/servico/internal/adaptadores/mensageria"
	"github.com/ThiagoCristovao/serverless-vending-validation/servico/internal/adaptadores/sistema"
	"github.com/ThiagoCristovao/serverless-vending-validation/servico/internal/aplicacao"
	"github.com/ThiagoCristovao/serverless-vending-validation/servico/internal/config"
	"github.com/ThiagoCristovao/serverless-vending-validation/servico/internal/dominio"
	"github.com/ThiagoCristovao/serverless-vending-validation/servico/internal/registro"
)

// NomeFuncao é o entry point configurado no Terraform (módulo funcao).
const NomeFuncao = "ValidarMaquina"

func init() {
	functions.HTTP(NomeFuncao, preguicoso().ServeHTTP)
}

var (
	montarUmaVez sync.Once
	manipulador  http.Handler
	erroMontagem error
)

// preguicoso adia a montagem das dependências para a primeira requisição.
func preguicoso() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		montarUmaVez.Do(func() { manipulador, erroMontagem = Montar(context.Background()) })
		if erroMontagem != nil {
			fmt.Fprintf(os.Stderr, `{"severity":"ERROR","message":"serviço não inicializado","erro":%q}`+"\n", erroMontagem.Error())
			http.Error(w, "serviço não inicializado", http.StatusServiceUnavailable)
			return
		}
		manipulador.ServeHTTP(w, r)
	})
}

// Montar constrói o manipulador HTTP com Firestore e Pub/Sub reais a partir da
// configuração de ambiente. Exportado para o executável local em modo nuvem e
// para os testes de integração.
func Montar(ctx context.Context) (http.Handler, error) {
	cfg, err := config.Carregar()
	if err != nil {
		return nil, fmt.Errorf("configuração: %w", err)
	}
	log := registro.Novo(os.Stdout, cfg.NivelLog)

	fs, err := firestore.NewClientWithDatabase(ctx, cfg.ProjetoId, cfg.BancoFirestore)
	if err != nil {
		return nil, fmt.Errorf("cliente firestore: %w", err)
	}
	ps, err := pubsub.NewClient(ctx, cfg.ProjetoId)
	if err != nil {
		return nil, fmt.Errorf("cliente pub/sub: %w", err)
	}
	gerador, err := dominio.NovoGeradorContrassenha(cfg.ChaveHmac)
	if err != nil {
		return nil, err
	}
	repositorio := armazenamento.Novo(fs)
	relogio := sistema.Relogio{}
	servico, err := aplicacao.Novo(aplicacao.Dependencias{
		Maquinas:            repositorio,
		Operadores:          repositorio,
		Chaves:              repositorio,
		Validacoes:          repositorio,
		Publicador:          mensageria.Novo(ps, cfg.TopicoValidacoes, cfg.TempoLimitePublica),
		Relogio:             relogio,
		Ids:                 sistema.GeradorUlid{Relogio: relogio},
		Contrassenha:        gerador,
		PrazoValidacao:      cfg.PrazoValidacao,
		AtrasoReconciliacao: cfg.AtrasoReconciliar,
		LoteReconciliacao:   cfg.LoteReconciliar,
		Registrador:         log,
	})
	if err != nil {
		return nil, err
	}
	log.Info("serviço montado", "projeto", cfg.ProjetoId, "topico", cfg.TopicoValidacoes, "banco", cfg.BancoFirestore)
	return apihttp.NovoManipulador(servico, apihttp.Opcoes{Registrador: log, ProjetoId: cfg.ProjetoId}), nil
}
