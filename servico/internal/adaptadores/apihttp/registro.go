package apihttp

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

type chaveContexto struct{}

// registroRequisicao acumula, ao longo do tratamento, o que a linha de log da
// requisição precisa: identificadores de correlação e o código de erro.
type registroRequisicao struct {
	idValidacao string
	idMaquina   string
	uid         string
	codigo      string
	erro        error
}

func registroDe(ctx context.Context) *registroRequisicao {
	reg, _ := ctx.Value(chaveContexto{}).(*registroRequisicao)
	return reg
}

// anotar guarda os identificadores de correlação para a linha de log.
func anotar(r *http.Request, idValidacao, idMaquina, uid string) {
	reg := registroDe(r.Context())
	if reg == nil {
		return
	}
	if idValidacao != "" {
		reg.idValidacao = idValidacao
	}
	if idMaquina != "" {
		reg.idMaquina = idMaquina
	}
	if uid != "" {
		reg.uid = uid
	}
}

type gravadorDeStatus struct {
	http.ResponseWriter
	status int
}

func (g *gravadorDeStatus) WriteHeader(status int) {
	g.status = status
	g.ResponseWriter.WriteHeader(status)
}

// registrarRequisicoes escreve uma linha estruturada por requisição, com os
// campos usados pelas métricas derivadas de log (docs/arquitetura.md, seção 10).
func registrarRequisicoes(log *slog.Logger, projetoId string, proximo http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		inicio := time.Now()
		reg := &registroRequisicao{}
		ctx := context.WithValue(r.Context(), chaveContexto{}, reg)
		g := &gravadorDeStatus{ResponseWriter: w, status: http.StatusOK}

		proximo.ServeHTTP(g, r.WithContext(ctx))

		atributos := []any{
			"metodo", r.Method,
			"caminho", r.URL.Path,
			"status", g.status,
			"duracaoMs", time.Since(inicio).Milliseconds(),
		}
		if reg.uid != "" {
			atributos = append(atributos, "uid", reg.uid)
		}
		if reg.idMaquina != "" {
			atributos = append(atributos, "idMaquina", reg.idMaquina)
		}
		if reg.idValidacao != "" {
			atributos = append(atributos, "idValidacao", reg.idValidacao)
		}
		if reg.codigo != "" {
			atributos = append(atributos, "codigo", reg.codigo)
		}
		if trace := traceDe(r, projetoId); trace != "" {
			atributos = append(atributos, "logging.googleapis.com/trace", trace)
		}
		switch {
		case g.status >= 500:
			if reg.erro != nil {
				atributos = append(atributos, "erro", reg.erro.Error())
			}
			log.ErrorContext(ctx, "requisição falhou", atributos...)
		case g.status >= 400:
			log.WarnContext(ctx, "requisição rejeitada", atributos...)
		default:
			log.InfoContext(ctx, "requisição atendida", atributos...)
		}
	})
}

// traceDe converte X-Cloud-Trace-Context (TRACE_ID/SPAN_ID;o=1) no formato que
// o Cloud Logging usa para correlacionar logs e traces.
func traceDe(r *http.Request, projetoId string) string {
	if projetoId == "" {
		return ""
	}
	cabecalho := r.Header.Get("X-Cloud-Trace-Context")
	if cabecalho == "" {
		return ""
	}
	id, _, _ := strings.Cut(cabecalho, "/")
	if id == "" {
		return ""
	}
	return "projects/" + projetoId + "/traces/" + id
}
