package laboratorio

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"cloud.google.com/go/pubsub/v2"

	"github.com/ThiagoCristovao/serverless-vending-validation/sistema-central-simulado/internal/admin"
	"github.com/ThiagoCristovao/serverless-vending-validation/sistema-central-simulado/internal/armazenamento"
	"github.com/ThiagoCristovao/serverless-vending-validation/sistema-central-simulado/internal/consumidor"
	"github.com/ThiagoCristovao/serverless-vending-validation/sistema-central-simulado/internal/falhas"
)

// ConsumidorLocal é o sistema central simulado dentro do processo que o usa,
// com registro em diretório temporário.
type ConsumidorLocal struct {
	Registro    *armazenamento.Registro
	Interruptor *falhas.Interruptor
	Processador *consumidor.Processador
	Consumidor  *consumidor.Consumidor
	Diretorio   string
	Log         *slog.Logger
}

// NovoConsumidorLocal prepara (sem iniciar) o consumidor da assinatura.
func NovoConsumidorLocal(cli *pubsub.Client, assinatura string, log *slog.Logger) (*ConsumidorLocal, error) {
	if log == nil {
		log = slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	}
	diretorio, err := os.MkdirTemp("", "svv-laboratorio-")
	if err != nil {
		return nil, err
	}
	registro, err := armazenamento.Abrir(filepath.Join(diretorio, "eventos.jsonl"))
	if err != nil {
		return nil, err
	}
	interruptor := falhas.Novo()
	proc := consumidor.NovoProcessador(registro, interruptor, log)
	return &ConsumidorLocal{
		Registro:    registro,
		Interruptor: interruptor,
		Processador: proc,
		Consumidor:  consumidor.Novo(cli, assinatura, proc, log),
		Diretorio:   diretorio,
		Log:         log,
	}, nil
}

// Iniciar começa (ou retoma) o consumo.
func (c *ConsumidorLocal) Iniciar(ctx context.Context) error { return c.Consumidor.Iniciar(ctx) }

// Pausar interrompe o consumo (indisponibilidade deliberada, CE-15).
func (c *ConsumidorLocal) Pausar() { c.Consumidor.Pausar() }

// HandlerAdmin expõe a API administrativa deste consumidor.
func (c *ConsumidorLocal) HandlerAdmin(ctx context.Context) http.Handler {
	return admin.NovoHandler(admin.Dependencias{Contexto: ctx, Registro: c.Registro, Processador: c.Processador, Interruptor: c.Interruptor, Consumidor: c.Consumidor, Log: c.Log})
}

// Encerrar pausa o consumo e fecha o registro (o arquivo é mantido).
func (c *ConsumidorLocal) Encerrar() {
	c.Consumidor.Pausar()
	_ = c.Registro.Fechar()
}
