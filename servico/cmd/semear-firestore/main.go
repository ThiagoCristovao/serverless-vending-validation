// Comando semear-firestore cadastra uma máquina, um operador e uma chave de
// assinatura de adesivos no Firestore do projeto (ou no emulador, se
// FIRESTORE_EMULATOR_HOST estiver definido). Usa as credenciais do gcloud (ADC).
//
//	go run ./cmd/semear-firestore -projeto svv-dev -operador-uid <uid do Firebase Auth> \
//	   -chave-publica ./chaves/chave-publica-qr.txt
//
// Registros existentes não são sobrescritos (preserva o contador da máquina),
// salvo com -sobrescrever.
package main

import (
	"context"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/firestore"

	"github.com/ThiagoCristovao/serverless-vending-validation/servico/internal/adaptadores/armazenamento"
	"github.com/ThiagoCristovao/serverless-vending-validation/servico/internal/dominio"
	"github.com/ThiagoCristovao/serverless-vending-validation/servico/internal/portas"
)

func main() {
	var (
		projeto      = flag.String("projeto", os.Getenv("SVV_PROJETO_ID"), "ID do projeto GCP")
		banco        = flag.String("banco", "(default)", "banco Firestore")
		maq          = flag.String("maquina", "VM-2047", "identificador da máquina")
		mod          = flag.String("modelo", "CN168", "modelo")
		loc          = flag.String("loc", "BLA-T", "identificador da localização")
		locNome      = flag.String("loc-nome", "Bloco A - Térreo", "nome da localização")
		contador     = flag.Uint64("contador", 0, "contador inicial de rotação da contrassenha")
		operadorUid  = flag.String("operador-uid", "", "uid do operador no Firebase Authentication")
		operadorNome = flag.String("operador-nome", "Operador de Testes", "nome do operador")
		operadorMail = flag.String("operador-email", "", "e-mail do operador")
		kid          = flag.String("kid", "k1", "identificador da chave de assinatura")
		chavePublica = flag.String("chave-publica", os.Getenv("SVV_CHAVE_PUBLICA_QR"), "chave pública Ed25519 em base64url ou caminho de arquivo")
		sobrescrever = flag.Bool("sobrescrever", false, "substitui registros existentes (zera o contador da máquina)")
	)
	flag.Parse()
	if *projeto == "" || *operadorUid == "" || *chavePublica == "" {
		falhar(errors.New("-projeto, -operador-uid e -chave-publica são obrigatórios"))
	}
	publica, err := carregarPublica(*chavePublica)
	if err != nil {
		falhar(err)
	}

	ctx := context.Background()
	cli, err := firestore.NewClientWithDatabase(ctx, *projeto, *banco)
	if err != nil {
		falhar(err)
	}
	defer cli.Close()
	repo := armazenamento.Novo(cli)
	agora := time.Now().UTC()

	var maquinas []dominio.Maquina
	if _, err := repo.ObterMaquina(ctx, *maq); errors.Is(err, portas.ErrNaoEncontrado) || *sobrescrever {
		maquinas = append(maquinas, dominio.Maquina{Id: *maq, Modelo: *mod, Localizacao: dominio.Localizacao{Id: *loc, Nome: *locNome}, Ativa: true, ContadorCodigo: *contador, CriadaEm: agora, AtualizadaEm: agora})
	} else if err != nil {
		falhar(err)
	} else {
		fmt.Fprintf(os.Stderr, "máquina %s já existe; mantida (use -sobrescrever para substituir)\n", *maq)
	}
	var operadores []dominio.Operador
	if _, err := repo.ObterOperador(ctx, *operadorUid); errors.Is(err, portas.ErrNaoEncontrado) || *sobrescrever {
		operadores = append(operadores, dominio.Operador{Uid: *operadorUid, Nome: *operadorNome, Email: *operadorMail, Ativo: true, CriadoEm: agora})
	} else if err != nil {
		falhar(err)
	} else {
		fmt.Fprintf(os.Stderr, "operador %s já existe; mantido\n", *operadorUid)
	}
	chaves := []dominio.ChaveQr{{Kid: *kid, ChavePublica: publica, Ativa: true, ValidaDe: agora, ValidaAte: agora.AddDate(2, 0, 0)}}

	if err := repo.Semear(ctx, maquinas, operadores, chaves); err != nil {
		falhar(err)
	}
	fmt.Printf("semeado em %s/%s: %d máquina(s), %d operador(es), chave %s\n", *projeto, *banco, len(maquinas), len(operadores), *kid)
}

func carregarPublica(valor string) ([]byte, error) {
	if conteudo, err := os.ReadFile(valor); err == nil {
		valor = strings.TrimSpace(string(conteudo))
	}
	bytes, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(valor))
	if err != nil || len(bytes) != 32 {
		return nil, errors.New("chave pública deve ser Ed25519 (32 bytes) em base64url")
	}
	return bytes, nil
}

func falhar(err error) {
	fmt.Fprintln(os.Stderr, "erro:", err)
	os.Exit(1)
}
