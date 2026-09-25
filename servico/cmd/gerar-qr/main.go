// Comando gerar-qr cria o par de chaves Ed25519 do administrador da rede e
// assina payloads de adesivo (docs/payload-qr.md), opcionalmente gerando o PNG
// do QR para colar em uma máquina de testes.
//
//	go run ./cmd/gerar-qr -gerar-chaves -saida ./chaves
//	go run ./cmd/gerar-qr -chave-privada ./chaves/chave-privada-qr.txt -maq VM-2047 -png vm-2047.png
package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	qrcode "github.com/skip2/go-qrcode"

	"github.com/ThiagoCristovao/serverless-vending-validation/servico/internal/dominio"
)

func main() {
	var (
		gerarChaves  = flag.Bool("gerar-chaves", false, "gera um par de chaves Ed25519 em -saida e encerra")
		saida        = flag.String("saida", ".", "diretório dos arquivos de chave gerados")
		chavePrivada = flag.String("chave-privada", os.Getenv("SVV_CHAVE_PRIVADA_QR"), "chave privada Ed25519 em base64url (64 bytes) ou caminho de arquivo com ela")
		maq          = flag.String("maq", "VM-2047", "identificador da máquina")
		mod          = flag.String("mod", "CN168", "modelo da máquina")
		loc          = flag.String("loc", "BLA-T", "identificador da localização")
		kid          = flag.String("kid", "k1", "identificador da chave de assinatura")
		exp          = flag.String("exp", time.Now().UTC().AddDate(1, 0, 0).Format("2006-01-02"), "validade do adesivo (AAAA-MM-DD)")
		png          = flag.String("png", "", "grava o QR em PNG neste caminho")
		nivel        = flag.String("nivel", "M", "correção de erro do QR: L, M, Q ou H")
		tamanho      = flag.Int("tamanho", 512, "lado do PNG em pixels")
	)
	flag.Parse()

	if *gerarChaves {
		if err := gerarParDeChaves(*saida); err != nil {
			falhar(err)
		}
		return
	}
	privada, err := carregarPrivada(*chavePrivada)
	if err != nil {
		falhar(err)
	}
	p := dominio.PayloadQr{V: dominio.VersaoPayloadQr, Maq: *maq, Mod: *mod, Loc: *loc, Exp: *exp, Kid: *kid}
	p.Assinar(privada)
	if err := p.Validar(); err != nil {
		falhar(fmt.Errorf("payload inválido: %w", err))
	}
	corpo, err := json.Marshal(p)
	if err != nil {
		falhar(err)
	}
	fmt.Println(string(corpo))

	if *png != "" {
		niveis := map[string]qrcode.RecoveryLevel{"L": qrcode.Low, "M": qrcode.Medium, "Q": qrcode.High, "H": qrcode.Highest}
		n, ok := niveis[strings.ToUpper(*nivel)]
		if !ok {
			falhar(errors.New("nível de correção deve ser L, M, Q ou H"))
		}
		if err := qrcode.WriteFile(string(corpo), n, *tamanho, *png); err != nil {
			falhar(fmt.Errorf("gerar PNG: %w", err))
		}
		fmt.Fprintf(os.Stderr, "QR gravado em %s (%d bytes de conteúdo, correção %s)\n", *png, len(corpo), strings.ToUpper(*nivel))
	}
}

func gerarParDeChaves(dir string) error {
	publica, privada, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	caminhoPrivada := filepath.Join(dir, "chave-privada-qr.txt")
	caminhoPublica := filepath.Join(dir, "chave-publica-qr.txt")
	if err := os.WriteFile(caminhoPrivada, []byte(base64.RawURLEncoding.EncodeToString(privada)+"\n"), 0o600); err != nil {
		return err
	}
	if err := os.WriteFile(caminhoPublica, []byte(base64.RawURLEncoding.EncodeToString(publica)+"\n"), 0o644); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "chave privada: %s (guarde fora da nuvem e do git)\nchave pública: %s (cadastre em chavesQr e embuta no aplicativo)\n", caminhoPrivada, caminhoPublica)
	fmt.Println(base64.RawURLEncoding.EncodeToString(publica))
	return nil
}

func carregarPrivada(valor string) (ed25519.PrivateKey, error) {
	if valor == "" {
		return nil, errors.New("informe -chave-privada (base64url ou arquivo) ou use -gerar-chaves")
	}
	if conteudo, err := os.ReadFile(valor); err == nil {
		valor = strings.TrimSpace(string(conteudo))
	}
	bytes, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(valor))
	if err != nil || len(bytes) != ed25519.PrivateKeySize {
		return nil, errors.New("chave privada deve ser Ed25519 (64 bytes) em base64url")
	}
	return ed25519.PrivateKey(bytes), nil
}

func falhar(err error) {
	fmt.Fprintln(os.Stderr, "erro:", err)
	os.Exit(1)
}
