// Comando local executa o serviço de validação na máquina do desenvolvedor,
// com as portas em memória e dados de exemplo, sem credenciais da GCP.
//
//	go run ./cmd/local            # porta 8080
//	PORT=9090 go run ./cmd/local
//
// Identifique o operador pelo cabeçalho X-Operador-Teste (ex.: op-1). O
// programa imprime, ao iniciar, um payload de QR assinado pronto para uso.
package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/GoogleCloudPlatform/functions-framework-go/funcframework"

	"github.com/ThiagoCristovao/serverless-vending-validation/servico/internal/adaptadores/apihttp"
	"github.com/ThiagoCristovao/serverless-vending-validation/servico/internal/adaptadores/memoria"
	"github.com/ThiagoCristovao/serverless-vending-validation/servico/internal/adaptadores/sistema"
	"github.com/ThiagoCristovao/serverless-vending-validation/servico/internal/aplicacao"
	"github.com/ThiagoCristovao/serverless-vending-validation/servico/internal/dominio"
	"github.com/ThiagoCristovao/serverless-vending-validation/servico/internal/registro"
)

// CabecalhoOperadorDeTeste identifica o operador em execução local.
const CabecalhoOperadorDeTeste = "X-Operador-Teste"

func main() {
	log := registro.Novo(os.Stderr, slog.LevelInfo)

	publica, privada, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		log.Error("gerar chave de exemplo", "erro", err)
		os.Exit(1)
	}
	agora := time.Now().UTC()
	arm := memoria.Novo()
	arm.SemearMaquina(dominio.Maquina{Id: "VM-2047", Modelo: "CN168", Localizacao: dominio.Localizacao{Id: "BLA-T", Nome: "Bloco A - Térreo"}, Ativa: true, ContadorCodigo: 12, CriadaEm: agora, AtualizadaEm: agora})
	arm.SemearMaquina(dominio.Maquina{Id: "VM-0009", Modelo: "CN168", Localizacao: dominio.Localizacao{Id: "BLB-1", Nome: "Bloco B"}, Ativa: false, CriadaEm: agora, AtualizadaEm: agora})
	arm.SemearOperador(dominio.Operador{Uid: "op-1", Nome: "Operador Um", Email: "op1@exemplo.invalid", Ativo: true, CriadoEm: agora})
	arm.SemearOperador(dominio.Operador{Uid: "op-2", Nome: "Operador Dois", Email: "op2@exemplo.invalid", Ativo: true, CriadoEm: agora})
	arm.SemearChaveQr(dominio.ChaveQr{Kid: "k1", ChavePublica: publica, Ativa: true, ValidaDe: agora, ValidaAte: agora.AddDate(2, 0, 0)})

	gerador, err := dominio.NovoGeradorContrassenha([]byte("chave-hmac-local-nao-usar-em-producao-0000"))
	if err != nil {
		log.Error("gerador de contrassenha", "erro", err)
		os.Exit(1)
	}
	relogio := sistema.Relogio{}
	servico, err := aplicacao.Novo(aplicacao.Dependencias{
		Maquinas: arm, Operadores: arm, Chaves: arm, Validacoes: arm,
		Publicador:   &publicadorEco{log: log},
		Relogio:      relogio,
		Ids:          sistema.GeradorUlid{Relogio: relogio},
		Contrassenha: gerador,
		Registrador:  log,
	})
	if err != nil {
		log.Error("montar serviço", "erro", err)
		os.Exit(1)
	}

	exemplo := dominio.PayloadQr{V: 1, Maq: "VM-2047", Mod: "CN168", Loc: "BLA-T", Exp: agora.AddDate(1, 0, 0).Format("2006-01-02"), Kid: "k1"}
	exemplo.Assinar(privada)
	payload, _ := json.Marshal(exemplo)
	fmt.Fprintf(os.Stderr, "\nPayload de QR assinado para testes:\n%s\n\n", payload)
	fmt.Fprintf(os.Stderr, "Exemplo:\n  curl -s -X POST localhost:%s/v1/validacoes -H '%s: op-1' -H 'Idempotency-Key: 6f1c2a4e-3b7d-4e8f-9a0b-1c2d3e4f5a6b' -H 'Content-Type: application/json' -d '{\"qr\":%s}'\n\n", porta(), CabecalhoOperadorDeTeste, payload)

	manipulador := apihttp.NovoManipulador(servico, apihttp.Opcoes{Registrador: log, CabecalhoOperadorDeTeste: CabecalhoOperadorDeTeste})
	if err := funcframework.RegisterHTTPFunctionContext(context.Background(), "/", manipulador.ServeHTTP); err != nil {
		log.Error("registrar função", "erro", err)
		os.Exit(1)
	}
	log.Info("serviço local pronto", "porta", porta())
	if err := funcframework.Start(porta()); err != nil {
		log.Error("servidor encerrado", "erro", err)
		os.Exit(1)
	}
}

func porta() string {
	if p := os.Getenv("PORT"); p != "" {
		return p
	}
	return "8080"
}

// publicadorEco imprime os eventos em vez de publicá-los.
type publicadorEco struct{ log *slog.Logger }

func (p *publicadorEco) Publicar(ctx context.Context, e dominio.Evento) error {
	corpo, _ := json.Marshal(e)
	p.log.InfoContext(ctx, "evento (eco local)", "tipo", string(e.Tipo), "idValidacao", e.IdValidacao, "corpo", string(corpo))
	return nil
}
