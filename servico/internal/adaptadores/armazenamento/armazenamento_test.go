//go:build integracao

// Testes de integração do repositório contra o emulador do Firestore.
// Execute com o emulador no ar (make emuladores-subir) e:
//
//	FIRESTORE_EMULATOR_HOST=localhost:8081 go test -tags integracao ./internal/adaptadores/armazenamento/
package armazenamento_test

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"cloud.google.com/go/firestore"

	"github.com/ThiagoCristovao/serverless-vending-validation/servico/internal/adaptadores/armazenamento"
	"github.com/ThiagoCristovao/serverless-vending-validation/servico/internal/adaptadores/sistema"
	"github.com/ThiagoCristovao/serverless-vending-validation/servico/internal/dominio"
	"github.com/ThiagoCristovao/serverless-vending-validation/servico/internal/portas"
)

var inicio = time.Date(2026, 9, 17, 13, 5, 12, 0, time.UTC)

func novoRepositorio(t *testing.T) *armazenamento.Repositorio {
	t.Helper()
	if os.Getenv("FIRESTORE_EMULATOR_HOST") == "" {
		t.Skip("FIRESTORE_EMULATOR_HOST não definido; emulador ausente")
	}
	ctx := context.Background()
	// Um projeto por execução isola os dados entre testes.
	projeto := fmt.Sprintf("svv-teste-%d", time.Now().UnixNano())
	cli, err := firestore.NewClient(ctx, projeto)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cli.Close() })
	repo := armazenamento.Novo(cli)

	pub, _, _ := ed25519.GenerateKey(rand.Reader)
	err = repo.Semear(ctx,
		[]dominio.Maquina{{Id: "VM-2047", Modelo: "CN168", Localizacao: dominio.Localizacao{Id: "BLA-T", Nome: "Bloco A"}, Ativa: true, ContadorCodigo: 12, CriadaEm: inicio, AtualizadaEm: inicio}},
		[]dominio.Operador{{Uid: "op-1", Nome: "Operador Um", Email: "op1@exemplo.invalid", Ativo: true, CriadoEm: inicio}},
		[]dominio.ChaveQr{{Kid: "k1", ChavePublica: pub, Ativa: true, ValidaDe: inicio, ValidaAte: inicio.AddDate(2, 0, 0)}},
	)
	if err != nil {
		t.Fatal(err)
	}
	return repo
}

func novaValidacao(t *testing.T, m *dominio.Maquina, o *dominio.Operador, chave string, agora time.Time) *dominio.Validacao {
	t.Helper()
	id, err := sistema.GeradorUlid{}.NovoId()
	if err != nil {
		t.Fatal(err)
	}
	p := &dominio.PayloadQr{V: 1, Maq: m.Id, Mod: m.Modelo, Loc: m.Localizacao.Id, Exp: "2027-12-31", Kid: "k1", Sig: "x"}
	return dominio.NovaValidacao(id, m, o, p, chave, nil, agora, 0)
}

func TestLeituras(t *testing.T) {
	repo := novoRepositorio(t)
	ctx := context.Background()

	m, err := repo.ObterMaquina(ctx, "VM-2047")
	if err != nil || m.ContadorCodigo != 12 || m.Localizacao.Nome != "Bloco A" || m.ValidacaoAbertaId != "" {
		t.Fatalf("máquina: %+v %v", m, err)
	}
	if _, err := repo.ObterMaquina(ctx, "VM-9999"); !errors.Is(err, portas.ErrNaoEncontrado) {
		t.Fatalf("máquina inexistente: %v", err)
	}
	o, err := repo.ObterOperador(ctx, "op-1")
	if err != nil || !o.Ativo || o.Nome != "Operador Um" {
		t.Fatalf("operador: %+v %v", o, err)
	}
	c, err := repo.ObterChaveQr(ctx, "k1")
	if err != nil || len(c.ChavePublica) != ed25519.PublicKeySize || !c.Ativa {
		t.Fatalf("chave: %+v %v", c, err)
	}
}

func TestAbrirConcluirEExpirar(t *testing.T) {
	repo := novoRepositorio(t)
	ctx := context.Background()
	m, _ := repo.ObterMaquina(ctx, "VM-2047")
	o, _ := repo.ObterOperador(ctx, "op-1")

	v1 := novaValidacao(t, m, o, "chave-1", inicio)
	if err := repo.Abrir(ctx, v1, inicio); err != nil {
		t.Fatalf("abrir: %v", err)
	}
	m, _ = repo.ObterMaquina(ctx, "VM-2047")
	if m.ValidacaoAbertaId != v1.Id {
		t.Fatalf("máquina deveria apontar a validação aberta: %+v", m)
	}

	lida, err := repo.ObterPorChaveIdempotencia(ctx, "chave-1")
	if err != nil || lida.Id != v1.Id || lida.Status != dominio.StatusAberta {
		t.Fatalf("idempotência: %+v %v", lida, err)
	}

	v2 := novaValidacao(t, m, o, "chave-2", inicio.Add(time.Minute))
	var emAndamento *portas.ErrValidacaoEmAndamento
	if err := repo.Abrir(ctx, v2, inicio.Add(time.Minute)); !errors.As(err, &emAndamento) || emAndamento.IdValidacao != v1.Id {
		t.Fatalf("segunda abertura deveria conflitar: %v", err)
	}

	fim := inicio.Add(10 * time.Minute)
	if err := v1.Concluir(dominio.Leituras{Medidor: 14832, UnidadesVendidas: 137}, true, "ok", fim); err != nil {
		t.Fatal(err)
	}
	v1.RegistrarAvisos(m)
	atualizada, err := repo.Concluir(ctx, v1, fim)
	if err != nil || atualizada.ContadorCodigo != 13 || atualizada.ValidacaoAbertaId != "" || atualizada.UltimoMedidor == nil || *atualizada.UltimoMedidor != 14832 {
		t.Fatalf("concluir: %+v %v", atualizada, err)
	}
	relida, _ := repo.ObterValidacao(ctx, v1.Id)
	if relida.Status != dominio.StatusConcluida || relida.Leituras == nil || relida.CodigoRotacionado == nil || !*relida.CodigoRotacionado {
		t.Fatalf("validação relida: %+v", relida)
	}

	if err := repo.MarcarEventoPublicado(ctx, v1.Id, dominio.EventoValidacaoIniciada, fim); err != nil {
		t.Fatal(err)
	}
	relida, _ = repo.ObterValidacao(ctx, v1.Id)
	if relida.Eventos.IniciadaEm == nil || relida.Eventos.ConcluidaEm != nil {
		t.Fatalf("marcadores: %+v", relida.Eventos)
	}

	pendentes, err := repo.PublicacoesPendentes(ctx, fim.Add(time.Hour), 10)
	if err != nil || len(pendentes) != 1 || pendentes[0].Id != v1.Id {
		t.Fatalf("pendentes (concluida sem marcador): %v %v", pendentes, err)
	}
	if err := repo.MarcarEventoPublicado(ctx, v1.Id, dominio.EventoValidacaoConcluida, fim); err != nil {
		t.Fatal(err)
	}
	pendentes, _ = repo.PublicacoesPendentes(ctx, fim.Add(time.Hour), 10)
	if len(pendentes) != 0 {
		t.Fatalf("não deveria haver pendências: %v", pendentes)
	}

	// Nova validação aberta e depois expirada: a abertura seguinte a supera.
	v3 := novaValidacao(t, atualizada, o, "chave-3", fim)
	if err := repo.Abrir(ctx, v3, fim); err != nil {
		t.Fatal(err)
	}
	depois := fim.Add(20 * time.Minute)
	v4 := novaValidacao(t, atualizada, o, "chave-4", depois)
	if err := repo.Abrir(ctx, v4, depois); err != nil {
		t.Fatalf("abertura após expiração deveria passar: %v", err)
	}
	v3Relida, _ := repo.ObterValidacao(ctx, v3.Id)
	if v3Relida.Status != dominio.StatusExpirada {
		t.Fatalf("v3 deveria estar expirada: %s", v3Relida.Status)
	}
	if err := repo.MarcarExpirada(ctx, v4.Id); err != nil {
		t.Fatal(err)
	}
	m, _ = repo.ObterMaquina(ctx, "VM-2047")
	if m.ValidacaoAbertaId != "" {
		t.Fatalf("MarcarExpirada deveria liberar a máquina: %+v", m)
	}
}
