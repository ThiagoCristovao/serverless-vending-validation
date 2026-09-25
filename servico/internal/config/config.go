// Package config lê a configuração do serviço das variáveis de ambiente
// definidas pelo Terraform (módulo funcao) ou pelo desenvolvedor.
package config

import (
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"
)

// Config é a configuração de execução do serviço na nuvem.
type Config struct {
	ProjetoId          string
	BancoFirestore     string
	TopicoValidacoes   string
	ChaveHmac          []byte
	PrazoValidacao     time.Duration
	AtrasoReconciliar  time.Duration
	LoteReconciliar    int
	NivelLog           slog.Level
	TempoLimitePublica time.Duration
}

// Carregar lê e valida as variáveis de ambiente:
//
//	SVV_PROJETO_ID           (ou GOOGLE_CLOUD_PROJECT)   obrigatória
//	SVV_TOPICO_VALIDACOES    nome do tópico Pub/Sub       obrigatória
//	SVV_CHAVE_HMAC           chave em hex (Secret Manager) obrigatória, ≥ 16 bytes
//	SVV_FIRESTORE_BANCO      padrão "(default)"
//	SVV_PRAZO_VALIDACAO      padrão 15m
//	SVV_ATRASO_RECONCILIACAO padrão 2m
//	SVV_LOTE_RECONCILIACAO   padrão 50
//	SVV_NIVEL_LOG            debug|info|warn|error, padrão info
//	SVV_TEMPO_LIMITE_PUBLICACAO padrão 2s
func Carregar() (Config, error) {
	c := Config{
		ProjetoId:          primeiro(os.Getenv("SVV_PROJETO_ID"), os.Getenv("GOOGLE_CLOUD_PROJECT")),
		BancoFirestore:     primeiro(os.Getenv("SVV_FIRESTORE_BANCO"), "(default)"),
		TopicoValidacoes:   os.Getenv("SVV_TOPICO_VALIDACOES"),
		PrazoValidacao:     duracao("SVV_PRAZO_VALIDACAO", 15*time.Minute),
		AtrasoReconciliar:  duracao("SVV_ATRASO_RECONCILIACAO", 2*time.Minute),
		LoteReconciliar:    50,
		NivelLog:           nivel(os.Getenv("SVV_NIVEL_LOG")),
		TempoLimitePublica: duracao("SVV_TEMPO_LIMITE_PUBLICACAO", 2*time.Second),
	}
	if lote := os.Getenv("SVV_LOTE_RECONCILIACAO"); lote != "" {
		if _, err := fmt.Sscanf(lote, "%d", &c.LoteReconciliar); err != nil || c.LoteReconciliar <= 0 {
			return c, errors.New("SVV_LOTE_RECONCILIACAO deve ser um inteiro positivo")
		}
	}
	if c.ProjetoId == "" {
		return c, errors.New("SVV_PROJETO_ID (ou GOOGLE_CLOUD_PROJECT) é obrigatória")
	}
	if c.TopicoValidacoes == "" {
		return c, errors.New("SVV_TOPICO_VALIDACOES é obrigatória")
	}
	chave := strings.TrimSpace(os.Getenv("SVV_CHAVE_HMAC"))
	if chave == "" {
		return c, errors.New("SVV_CHAVE_HMAC é obrigatória")
	}
	if bytes, err := hex.DecodeString(chave); err == nil {
		c.ChaveHmac = bytes
	} else {
		c.ChaveHmac = []byte(chave)
	}
	if len(c.ChaveHmac) < 16 {
		return c, errors.New("SVV_CHAVE_HMAC deve ter pelo menos 16 bytes")
	}
	return c, nil
}

func primeiro(valores ...string) string {
	for _, v := range valores {
		if v != "" {
			return v
		}
	}
	return ""
}

func duracao(variavel string, padrao time.Duration) time.Duration {
	if v := os.Getenv(variavel); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			return d
		}
	}
	return padrao
}

func nivel(v string) slog.Level {
	switch strings.ToLower(v) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
