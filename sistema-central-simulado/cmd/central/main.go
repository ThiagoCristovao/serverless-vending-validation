// Comando central executa o sistema central simulado: consome a assinatura de
// validações do Pub/Sub, registra os eventos de forma idempotente em um arquivo
// JSON Lines e expõe a API administrativa (internal/admin).
//
//	go run ./cmd/central -projeto svv-dev -assinatura svv-dev-sistema-central
//
// Com PUBSUB_EMULATOR_HOST definido, consome do emulador local.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cloud.google.com/go/pubsub/v2"
	"google.golang.org/api/option"

	"github.com/ThiagoCristovao/serverless-vending-validation/sistema-central-simulado/internal/admin"
	"github.com/ThiagoCristovao/serverless-vending-validation/sistema-central-simulado/internal/armazenamento"
	"github.com/ThiagoCristovao/serverless-vending-validation/sistema-central-simulado/internal/consumidor"
	"github.com/ThiagoCristovao/serverless-vending-validation/sistema-central-simulado/internal/falhas"
)

func main() {
	var (
		projeto    = flag.String("projeto", os.Getenv("SVV_CENTRAL_PROJETO"), "ID do projeto GCP (SVV_CENTRAL_PROJETO)")
		assinatura = flag.String("assinatura", ambiente("SVV_CENTRAL_ASSINATURA", "svv-dev-sistema-central"), "assinatura consumida (SVV_CENTRAL_ASSINATURA)")
		arquivo    = flag.String("arquivo", ambiente("SVV_CENTRAL_ARQUIVO", "dados/eventos.jsonl"), "arquivo JSON Lines dos eventos; vazio = só memória (SVV_CENTRAL_ARQUIVO)")
		porta      = flag.String("porta", ambiente("SVV_CENTRAL_PORTA", "8090"), "porta da API administrativa (SVV_CENTRAL_PORTA)")
		impersonar = flag.String("impersonar", os.Getenv("SVV_CENTRAL_IMPERSONAR"), "e-mail da conta de serviço a personificar via ADC (SVV_CENTRAL_IMPERSONAR)")
		nivel      = flag.String("nivel-log", ambiente("SVV_CENTRAL_NIVEL_LOG", "info"), "nível de log: debug, info, warn ou error")
		pausado    = flag.Bool("pausado", false, "inicia sem consumir; retomar com POST /admin/retomar")
	)
	flag.Parse()
	if *projeto == "" {
		falhar(errors.New("informe -projeto ou SVV_CENTRAL_PROJETO"))
	}

	var nivelLog slog.Level
	if err := nivelLog.UnmarshalText([]byte(*nivel)); err != nil {
		falhar(fmt.Errorf("nível de log: %w", err))
	}
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: nivelLog}))

	ctx, parar := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer parar()

	var opcoes []option.ClientOption
	if *impersonar != "" {
		opcoes = append(opcoes, option.ImpersonateCredentials(*impersonar))
	}
	cli, err := pubsub.NewClient(ctx, *projeto, opcoes...)
	if err != nil {
		falhar(fmt.Errorf("cliente Pub/Sub: %w", err))
	}
	defer cli.Close()

	registro, err := armazenamento.Abrir(*arquivo)
	if err != nil {
		falhar(fmt.Errorf("registro: %w", err))
	}
	defer registro.Fechar()

	interruptor := falhas.Novo()
	proc := consumidor.NovoProcessador(registro, interruptor, log)
	cons := consumidor.Novo(cli, *assinatura, proc, log)
	if !*pausado {
		if err := cons.Iniciar(ctx); err != nil {
			falhar(err)
		}
	}

	servidor := &http.Server{
		Addr:              ":" + *porta,
		Handler:           admin.NovoHandler(admin.Dependencias{Contexto: ctx, Registro: registro, Processador: proc, Interruptor: interruptor, Consumidor: cons, Log: log}),
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		if err := servidor.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("API administrativa encerrada", "erro", err.Error())
			parar()
		}
	}()
	log.Info("sistema central simulado no ar", "projeto", *projeto, "assinatura", *assinatura, "arquivo", registro.Caminho(), "porta", *porta, "eventosCarregados", registro.Total(), "consumindo", !*pausado)

	<-ctx.Done()
	log.Info("encerrando")
	cons.Pausar()
	ctxEncerrar, cancelar := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelar()
	_ = servidor.Shutdown(ctxEncerrar)
}

func ambiente(nome, padrao string) string {
	if v := os.Getenv(nome); v != "" {
		return v
	}
	return padrao
}

func falhar(err error) {
	fmt.Fprintln(os.Stderr, "erro:", err)
	os.Exit(1)
}
