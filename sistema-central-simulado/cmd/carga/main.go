// Comando carga executa a avaliação de escalabilidade da Sprint 10 contra o
// ambiente implantado e grava relatórios de evidências em Markdown.
//
//	make carga-preparar OPERADORES=100            # cria operadores e máquinas de carga (uma vez)
//	make carga OPERADORES=50 CONFIGURACAO=A       # 50 operadores virtuais em laço fechado
//	make carga-frio CONFIGURACAO=min0             # cold start após inatividade (pausa o agendador)
//
// Usa as variáveis SVV_E2E_* (as mesmas da suíte e2e), SVV_CARGA_SENHA e
// SVV_CARGA_LOTE, e as credenciais ADC do gcloud para Monitoring, Logging,
// Identity Toolkit (criação de contas) e Cloud Scheduler.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/ThiagoCristovao/serverless-vending-validation/sistema-central-simulado/internal/laboratorio"
)

type avaliacao struct {
	cfg          laboratorio.Configuracao
	rel          *laboratorio.Relatorio
	nuvem        *laboratorio.Nuvem
	adesivo      *laboratorio.Adesivo
	configuracao string
	operadores   int
	duracao      time.Duration
	rampa        time.Duration
	pausa        time.Duration
	amostras     int
	inatividade  time.Duration
	lote         string
	senha        string
}

func main() {
	var (
		cenario     = flag.String("cenario", "", "preparar | carga | frio")
		n           = flag.Int("operadores", 10, "operadores virtuais simultâneos (carga) ou a criar (preparar)")
		conf        = flag.String("configuracao", "A", "rótulo da configuração da função sob teste (A, C, min0, min1…)")
		dur         = flag.Duration("duracao", 2*time.Minute, "duração da carga em regime, após a rampa")
		rampa       = flag.Duration("rampa", 30*time.Second, "tempo para colocar todos os operadores em atividade")
		pausa       = flag.Duration("pausa", time.Second, "pausa de cada operador entre uma validação e a seguinte")
		amostras    = flag.Int("amostras", 6, "amostras de cold start (frio)")
		inatividade = flag.Duration("inatividade", 20*time.Minute, "inatividade antes de cada amostra de cold start")
		saida       = flag.String("saida", "", "arquivo Markdown do relatório (padrão: stdout)")
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
	x := &avaliacao{
		cfg: cfg, configuracao: *conf, operadores: *n, duracao: *dur, rampa: *rampa, pausa: *pausa,
		amostras: *amostras, inatividade: *inatividade,
		lote: os.Getenv("SVV_CARGA_LOTE"), senha: os.Getenv("SVV_CARGA_SENHA"),
	}
	if x.nuvem, err = laboratorio.NovaNuvem(ctx, cfg.Projeto, nil); err != nil {
		falhar(err)
	}
	defer x.nuvem.Fechar()
	if x.adesivo, err = laboratorio.NovoAdesivo(cfg); err != nil {
		falhar(err)
	}

	switch *cenario {
	case "preparar":
		if err := x.preparar(ctx); err != nil {
			falhar(err)
		}
		return
	case "carga":
		x.rel = laboratorio.NovoRelatorio(fmt.Sprintf("Carga — configuração %s, %d operadores simultâneos (RNF-05, RNF-06)", *conf, *n))
		err = x.carga(ctx)
	case "frio":
		x.rel = laboratorio.NovoRelatorio(fmt.Sprintf("Cold start — configuração %s (RNF-06, CE-18)", *conf))
		err = x.frio(ctx)
	default:
		falhar(fmt.Errorf("cenário desconhecido %q", *cenario))
	}
	if err != nil {
		x.rel.Verificar(false, "avaliação interrompida: %v", err)
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

func falhar(err error) {
	fmt.Fprintln(os.Stderr, "erro:", err)
	os.Exit(1)
}
