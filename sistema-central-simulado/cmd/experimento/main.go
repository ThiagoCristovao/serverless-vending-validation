// Comando experimento executa os cenários de injeção de falhas da Sprint 9
// contra o ambiente implantado e grava um relatório de evidências em Markdown.
//
//	make experimento CENARIO=central-indisponivel
//	make experimento CENARIO=publicacao-indisponivel
//	make experimento CENARIO=firestore-indisponivel
//	make experimento CENARIO=restaurar          # repara os vínculos IAM se um cenário foi interrompido
//
// Usa as variáveis SVV_E2E_* (as mesmas da suíte e2e) e as credenciais ADC do
// gcloud para Monitoring, Logging, Firestore e IAM.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"cloud.google.com/go/pubsub/v2"

	"github.com/ThiagoCristovao/serverless-vending-validation/sistema-central-simulado/internal/laboratorio"
)

type experimento struct {
	cfg               laboratorio.Configuracao
	rel               *laboratorio.Relatorio
	gateway           *laboratorio.Gateway
	adesivo           *laboratorio.Adesivo
	operador          laboratorio.Operador
	pubsub            *pubsub.Client
	nuvem             *laboratorio.Nuvem
	central           *laboratorio.ConsumidorLocal
	validacoes        int
	indisponibilidade time.Duration
	prazoRecuperacao  time.Duration
}

func main() {
	var (
		cenario = flag.String("cenario", "", "central-indisponivel | publicacao-indisponivel | firestore-indisponivel | restaurar")
		n       = flag.Int("validacoes", 6, "validações (abertura + conclusão) executadas durante a falha")
		dur     = flag.Duration("indisponibilidade", 3*time.Minute, "duração da indisponibilidade do sistema central (cenário central-indisponivel)")
		prazo   = flag.Duration("prazo-recuperacao", 12*time.Minute, "prazo máximo para a recuperação após sanar a falha")
		saida   = flag.String("saida", "", "arquivo Markdown do relatório (padrão: stdout)")
	)
	flag.Parse()
	if *cenario == "" {
		falhar(errors.New("informe -cenario"))
	}

	ctx, parar := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer parar()

	cfg, err := laboratorio.CarregarConfiguracao()
	if err != nil {
		falhar(err)
	}
	x := &experimento{cfg: cfg, validacoes: *n, indisponibilidade: *dur, prazoRecuperacao: *prazo}
	x.rel = laboratorio.NovoRelatorio(titulo(*cenario))
	x.rel.Parametro("projeto", cfg.Projeto)
	x.rel.Parametro("gateway", cfg.Gateway)
	x.rel.Parametro("validações durante a falha", *n)

	if x.pubsub, err = pubsub.NewClient(ctx, cfg.Projeto); err != nil {
		falhar(err)
	}
	defer x.pubsub.Close()
	if x.nuvem, err = laboratorio.NovaNuvem(ctx, cfg.Projeto, x.pubsub); err != nil {
		falhar(err)
	}
	defer x.nuvem.Fechar()

	if *cenario == "restaurar" {
		if err := x.restaurar(ctx); err != nil {
			falhar(err)
		}
		return
	}

	if x.adesivo, err = laboratorio.NovoAdesivo(cfg); err != nil {
		falhar(err)
	}
	if x.operador, err = laboratorio.EntrarComoOperador(ctx, cfg); err != nil {
		falhar(fmt.Errorf("token do operador: %w", err))
	}
	x.gateway = laboratorio.NovoGateway(cfg, x.operador, func(l string) { fmt.Fprintln(os.Stderr, "   "+l) })

	log := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	if x.central, err = laboratorio.NovoConsumidorLocal(x.pubsub, cfg.Assinatura(), log); err != nil {
		falhar(err)
	}
	defer x.central.Encerrar()
	x.rel.Parametro("registro do consumidor", x.central.Registro.Caminho())

	switch *cenario {
	case "central-indisponivel":
		err = x.centralIndisponivel(ctx)
	case "publicacao-indisponivel":
		err = x.publicacaoIndisponivel(ctx)
	case "firestore-indisponivel":
		err = x.firestoreIndisponivel(ctx)
	default:
		falhar(fmt.Errorf("cenário desconhecido %q", *cenario))
	}
	if err != nil {
		x.rel.Verificar(false, "experimento interrompido: %v", err)
	}

	markdown := x.rel.Markdown()
	if *saida == "" {
		fmt.Print(markdown)
	} else {
		if err := os.MkdirAll(filepath.Dir(*saida), 0o755); err != nil {
			falhar(err)
		}
		if err := os.WriteFile(*saida, []byte(markdown), 0o644); err != nil {
			falhar(err)
		}
		fmt.Fprintln(os.Stderr, "relatório gravado em", *saida)
	}
	if !x.rel.Ok() {
		os.Exit(2)
	}
}

func titulo(cenario string) string {
	switch cenario {
	case "central-indisponivel":
		return "Cenário A — sistema central indisponível durante validações (CE-15, RNF-01, RNF-03)"
	case "publicacao-indisponivel":
		return "Cenário B — Pub/Sub indisponível ao publicar (CE-14, RNF-04, ADR-0012)"
	case "firestore-indisponivel":
		return "Cenário C — Firestore indisponível (CE-13, RF-23)"
	}
	return cenario
}

// restaurar repara os vínculos IAM que os cenários removem, caso uma execução
// tenha sido interrompida antes de restaurá-los.
func (x *experimento) restaurar(ctx context.Context) error {
	conta := "serviceAccount:" + x.cfg.ContaFuncao()
	if err := x.nuvem.GarantirPublicador(ctx, x.cfg.Topico(), conta); err != nil {
		return err
	}
	fmt.Fprintln(os.Stderr, "publicador do tópico garantido:", conta)
	if err := x.nuvem.GarantirPapelProjeto(ctx, "roles/datastore.user", conta); err != nil {
		return err
	}
	fmt.Fprintln(os.Stderr, "roles/datastore.user garantido:", conta)
	return nil
}

func falhar(err error) {
	fmt.Fprintln(os.Stderr, "erro:", err)
	os.Exit(1)
}
