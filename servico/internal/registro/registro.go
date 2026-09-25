// Package registro configura o log estruturado no formato que o Cloud Logging
// interpreta (severity, message, time), com slog da biblioteca padrão.
package registro

import (
	"io"
	"log/slog"
	"time"
)

// Novo cria um registrador JSON compatível com o Cloud Logging. Em execução
// local, o mesmo formato continua legível.
func Novo(saida io.Writer, nivel slog.Level) *slog.Logger {
	h := slog.NewJSONHandler(saida, &slog.HandlerOptions{
		Level: nivel,
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			switch a.Key {
			case slog.TimeKey:
				if t, ok := a.Value.Any().(time.Time); ok {
					return slog.String("time", t.UTC().Format(time.RFC3339Nano))
				}
			case slog.LevelKey:
				if nivel, ok := a.Value.Any().(slog.Level); ok {
					return slog.String("severity", severidade(nivel))
				}
			case slog.MessageKey:
				return slog.String("message", a.Value.String())
			}
			return a
		},
	})
	return slog.New(h)
}

func severidade(n slog.Level) string {
	switch {
	case n >= slog.LevelError:
		return "ERROR"
	case n >= slog.LevelWarn:
		return "WARNING"
	case n >= slog.LevelInfo:
		return "INFO"
	default:
		return "DEBUG"
	}
}
