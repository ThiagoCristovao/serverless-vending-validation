// Comando proxy-rede sobe o proxy de falhas de rede entre o aplicativo e o
// API Gateway (Sprint 9). O celular, na mesma rede Wi-Fi, usa um APK
// compilado com --dart-define=SVV_API_BASE=http://<ip-deste-computador>:8099/v1
// (make app-apk-injecao).
//
//	go run ./cmd/proxy-rede -destino https://<host-do-gateway> -modo intermitente
//	curl -X PUT localhost:8099/admin/modo -d '{"modo":"cortar-resposta"}'
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ThiagoCristovao/serverless-vending-validation/ferramentas/injecao-falhas/internal/proxy"
)

func main() {
	var (
		escuta    = flag.String("escuta", ":8099", "endereço de escuta")
		destino   = flag.String("destino", os.Getenv("SVV_PROXY_DESTINO"), "URL do gateway, ex.: https://svv-dev-gateway-xxxx.ue.gateway.dev")
		modo      = flag.String("modo", "normal", "normal | cortar | atrasar | indisponivel | cortar-resposta | intermitente")
		proporcao = flag.Float64("proporcao", 1, "proporção das requisições afetadas (0 a 1)")
		atraso    = flag.Duration("atraso", 25*time.Second, "atraso do modo atrasar (o aplicativo desiste após 20 s)")
	)
	flag.Parse()
	if *destino == "" {
		fmt.Fprintln(os.Stderr, "erro: informe -destino (ou SVV_PROXY_DESTINO)")
		os.Exit(1)
	}
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	p, err := proxy.Novo(*destino, proxy.Configuracao{Modo: proxy.Modo(*modo), Proporcao: *proporcao, Atraso: *atraso}, log)
	if err != nil {
		fmt.Fprintln(os.Stderr, "erro:", err)
		os.Exit(1)
	}

	ctx, parar := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer parar()
	servidor := &http.Server{Addr: *escuta, Handler: p.Handler(), ReadHeaderTimeout: 10 * time.Second}
	go func() {
		if err := servidor.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("proxy encerrado", "erro", err.Error())
			parar()
		}
	}()
	log.Info("proxy de rede no ar", "escuta", *escuta, "destino", *destino, "modo", *modo, "proporcao", *proporcao, "atraso", atraso.String(), "ipLocal", ipLocal())
	<-ctx.Done()
	ctxEncerrar, cancelar := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancelar()
	_ = servidor.Shutdown(ctxEncerrar)
}

// ipLocal descobre o IP usado para sair à rede (o que o celular deve apontar).
func ipLocal() string {
	conn, err := net.Dial("udp", "8.8.8.8:53")
	if err != nil {
		return "?"
	}
	defer conn.Close()
	return conn.LocalAddr().(*net.UDPAddr).IP.String()
}
