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
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"slices"
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
		lote         = flag.String("lote", "", "arquivo JSON com listas de máquinas e operadores a semear em lote (testes de carga, Sprint 10)")
	)
	flag.Parse()
	if *projeto == "" || *chavePublica == "" || (*operadorUid == "" && *lote == "") {
		falhar(errors.New("-projeto, -chave-publica e -operador-uid (ou -lote) são obrigatórios"))
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
	var operadores []dominio.Operador
	if *lote == "" {
		maquinas = append(maquinas, novaMaquina(*maq, *mod, *loc, *locNome, *contador, agora))
		operadores = append(operadores, dominio.Operador{Uid: *operadorUid, Nome: *operadorNome, Email: *operadorMail, Ativo: true, CriadoEm: agora})
	} else {
		var l loteSemente
		conteudo, err := os.ReadFile(*lote)
		if err == nil {
			err = json.Unmarshal(conteudo, &l)
		}
		if err != nil {
			falhar(fmt.Errorf("lote: %w", err))
		}
		for _, m := range l.Maquinas {
			maquinas = append(maquinas, novaMaquina(m.Id, valorOu(m.Modelo, *mod), m.Loc, m.LocNome, 0, agora))
		}
		for _, o := range l.Operadores {
			operadores = append(operadores, dominio.Operador{Uid: o.Uid, Nome: o.Nome, Email: o.Email, Ativo: true, CriadoEm: agora})
		}
	}
	maquinas = slices.DeleteFunc(maquinas, func(m dominio.Maquina) bool {
		_, err := repo.ObterMaquina(ctx, m.Id)
		if errors.Is(err, portas.ErrNaoEncontrado) || *sobrescrever {
			return false
		}
		if err != nil {
			falhar(err)
		}
		fmt.Fprintf(os.Stderr, "máquina %s já existe; mantida (use -sobrescrever para substituir)\n", m.Id)
		return true
	})
	operadores = slices.DeleteFunc(operadores, func(o dominio.Operador) bool {
		_, err := repo.ObterOperador(ctx, o.Uid)
		if errors.Is(err, portas.ErrNaoEncontrado) || *sobrescrever {
			return false
		}
		if err != nil {
			falhar(err)
		}
		fmt.Fprintf(os.Stderr, "operador %s já existe; mantido\n", o.Uid)
		return true
	})
	chaves := []dominio.ChaveQr{{Kid: *kid, ChavePublica: publica, Ativa: true, ValidaDe: agora, ValidaAte: agora.AddDate(2, 0, 0)}}

	if err := repo.Semear(ctx, maquinas, operadores, chaves); err != nil {
		falhar(err)
	}
	fmt.Printf("semeado em %s/%s: %d máquina(s), %d operador(es), chave %s\n", *projeto, *banco, len(maquinas), len(operadores), *kid)
}

// loteSemente é o formato do arquivo de -lote.
type loteSemente struct {
	Maquinas []struct {
		Id      string `json:"id"`
		Modelo  string `json:"modelo"`
		Loc     string `json:"loc"`
		LocNome string `json:"locNome"`
	} `json:"maquinas"`
	Operadores []struct {
		Uid   string `json:"uid"`
		Nome  string `json:"nome"`
		Email string `json:"email"`
	} `json:"operadores"`
}

func novaMaquina(id, modelo, loc, locNome string, contador uint64, agora time.Time) dominio.Maquina {
	return dominio.Maquina{Id: id, Modelo: modelo, Localizacao: dominio.Localizacao{Id: loc, Nome: locNome}, Ativa: true, ContadorCodigo: contador, CriadaEm: agora, AtualizadaEm: agora}
}

func valorOu(v, padrao string) string {
	if v == "" {
		return padrao
	}
	return v
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
