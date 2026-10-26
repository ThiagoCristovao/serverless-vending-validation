// Package admin expõe a API HTTP do sistema central simulado: consulta dos
// eventos registrados (asserções dos testes de ponta a ponta), estatísticas,
// interruptor de falhas e pausa/retomada do consumo (Sprints 8 e 9).
//
//	GET  /saude               estado resumido
//	GET  /eventos             ?idValidacao=&idMaquina=&tipo=
//	GET  /estatisticas        contadores do processador
//	GET  /admin/falhas        configuração do interruptor
//	PUT  /admin/falhas        {"modo":"rejeitar","idMaquina":"VM-2047"}
//	POST /admin/pausar        para de consumir a assinatura
//	POST /admin/retomar       volta a consumir
//	GET  /admin/estado        estado do consumo
package admin

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/ThiagoCristovao/serverless-vending-validation/sistema-central-simulado/internal/armazenamento"
	"github.com/ThiagoCristovao/serverless-vending-validation/sistema-central-simulado/internal/consumidor"
	"github.com/ThiagoCristovao/serverless-vending-validation/sistema-central-simulado/internal/falhas"
)

// Controle é o que a API precisa do consumidor.
type Controle interface {
	Iniciar(ctx context.Context) error
	Pausar()
	Estado() consumidor.Estado
}

// Dependencias da API.
type Dependencias struct {
	// Contexto base usado ao retomar o consumo.
	Contexto    context.Context
	Registro    *armazenamento.Registro
	Processador *consumidor.Processador
	Interruptor *falhas.Interruptor
	Consumidor  Controle
	Log         *slog.Logger
}

// NovoHandler monta as rotas.
func NovoHandler(d Dependencias) http.Handler {
	if d.Log == nil {
		d.Log = slog.Default()
	}
	if d.Contexto == nil {
		d.Contexto = context.Background()
	}
	mux := http.NewServeMux()

	mux.HandleFunc("GET /saude", func(w http.ResponseWriter, _ *http.Request) {
		responder(w, http.StatusOK, map[string]any{"status": "ok", "consumindo": d.Consumidor.Estado().Consumindo, "eventos": d.Registro.Total()})
	})

	mux.HandleFunc("GET /eventos", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		itens := d.Registro.Listar(armazenamento.Filtro{IdValidacao: q.Get("idValidacao"), IdMaquina: q.Get("idMaquina"), Tipo: q.Get("tipo")})
		if itens == nil {
			itens = []armazenamento.Item{}
		}
		responder(w, http.StatusOK, map[string]any{"total": len(itens), "itens": itens})
	})

	mux.HandleFunc("GET /estatisticas", func(w http.ResponseWriter, _ *http.Request) {
		responder(w, http.StatusOK, d.Processador.Estatisticas())
	})

	mux.HandleFunc("GET /admin/falhas", func(w http.ResponseWriter, _ *http.Request) {
		responder(w, http.StatusOK, d.Interruptor.Atual())
	})

	mux.HandleFunc("PUT /admin/falhas", func(w http.ResponseWriter, r *http.Request) {
		var cfg falhas.Configuracao
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&cfg); err != nil {
			responder(w, http.StatusBadRequest, map[string]string{"erro": "corpo JSON inválido: " + err.Error()})
			return
		}
		if err := d.Interruptor.Definir(cfg); err != nil {
			responder(w, http.StatusBadRequest, map[string]string{"erro": err.Error()})
			return
		}
		atual := d.Interruptor.Atual()
		d.Log.Warn("interruptor de falhas alterado", "modo", atual.Modo, "idMaquina", atual.IdMaquina)
		responder(w, http.StatusOK, atual)
	})

	mux.HandleFunc("POST /admin/pausar", func(w http.ResponseWriter, _ *http.Request) {
		d.Consumidor.Pausar()
		responder(w, http.StatusOK, d.Consumidor.Estado())
	})

	mux.HandleFunc("POST /admin/retomar", func(w http.ResponseWriter, _ *http.Request) {
		if err := d.Consumidor.Iniciar(d.Contexto); err != nil {
			responder(w, http.StatusInternalServerError, map[string]string{"erro": err.Error()})
			return
		}
		responder(w, http.StatusOK, d.Consumidor.Estado())
	})

	mux.HandleFunc("GET /admin/estado", func(w http.ResponseWriter, _ *http.Request) {
		responder(w, http.StatusOK, d.Consumidor.Estado())
	})

	return mux
}

func responder(w http.ResponseWriter, status int, corpo any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(corpo)
}
