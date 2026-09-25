package aplicacao_test

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/ThiagoCristovao/serverless-vending-validation/servico/internal/adaptadores/memoria"
	"github.com/ThiagoCristovao/serverless-vending-validation/servico/internal/aplicacao"
	"github.com/ThiagoCristovao/serverless-vending-validation/servico/internal/dominio"
)

var inicio = time.Date(2026, 9, 17, 13, 5, 12, 0, time.UTC)

type ambiente struct {
	t       *testing.T
	arm     *memoria.Armazenamento
	pub     *memoria.Publicador
	rel     *memoria.Relogio
	srv     *aplicacao.Servico
	priv    ed25519.PrivateKey
	gerador *dominio.GeradorContrassenha
}

func novoAmbiente(t *testing.T) *ambiente {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	arm := memoria.Novo()
	arm.SemearMaquina(dominio.Maquina{Id: "VM-2047", Modelo: "CN168", Localizacao: dominio.Localizacao{Id: "BLA-T", Nome: "Bloco A - Térreo"}, Ativa: true, ContadorCodigo: 12, CriadaEm: inicio, AtualizadaEm: inicio})
	arm.SemearMaquina(dominio.Maquina{Id: "VM-0009", Modelo: "CN168", Localizacao: dominio.Localizacao{Id: "BLB-1", Nome: "Bloco B"}, Ativa: false})
	medidorAnterior := int64(20000)
	arm.SemearMaquina(dominio.Maquina{Id: "VM-3000", Modelo: "CN168", Localizacao: dominio.Localizacao{Id: "BLC-1", Nome: "Bloco C"}, Ativa: true, ContadorCodigo: 3, UltimoMedidor: &medidorAnterior})
	arm.SemearOperador(dominio.Operador{Uid: "op-1", Nome: "Operador Um", Ativo: true})
	arm.SemearOperador(dominio.Operador{Uid: "op-2", Nome: "Operador Dois", Ativo: true})
	arm.SemearOperador(dominio.Operador{Uid: "op-inativo", Nome: "Inativo", Ativo: false})
	arm.SemearChaveQr(dominio.ChaveQr{Kid: "k1", ChavePublica: pub, Ativa: true})
	arm.SemearChaveQr(dominio.ChaveQr{Kid: "k0", ChavePublica: pub, Ativa: false})

	rel := memoria.NovoRelogio(inicio)
	publicador := &memoria.Publicador{}
	gerador, err := dominio.NovoGeradorContrassenha([]byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	srv, err := aplicacao.Novo(aplicacao.Dependencias{
		Maquinas: arm, Operadores: arm, Chaves: arm, Validacoes: arm,
		Publicador:   publicador,
		Relogio:      rel,
		Ids:          memoria.GeradorUlid{Relogio: rel},
		Contrassenha: gerador,
		Registrador:  slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	if err != nil {
		t.Fatal(err)
	}
	return &ambiente{t: t, arm: arm, pub: publicador, rel: rel, srv: srv, priv: priv, gerador: gerador}
}

// payload monta um adesivo assinado; ajuste, quando informado, é aplicado antes
// da assinatura (o adesivo continua autêntico).
func (a *ambiente) payload(ajuste func(p *dominio.PayloadQr)) []byte {
	a.t.Helper()
	p := dominio.PayloadQr{V: 1, Maq: "VM-2047", Mod: "CN168", Loc: "BLA-T", Exp: "2027-12-31", Kid: "k1"}
	if ajuste != nil {
		ajuste(&p)
	}
	p.Assinar(a.priv)
	b, err := json.Marshal(p)
	if err != nil {
		a.t.Fatal(err)
	}
	return b
}

func (a *ambiente) iniciar(uid, chave string, bruto []byte) (*aplicacao.ResultadoValidacao, error) {
	return a.srv.IniciarValidacao(context.Background(), aplicacao.EntradaIniciarValidacao{Uid: uid, ChaveIdempotencia: chave, PayloadQr: bruto})
}

func (a *ambiente) concluir(uid, id string, leituras dominio.Leituras, rotacionado bool) (*aplicacao.ResultadoConclusao, error) {
	return a.srv.ConcluirValidacao(context.Background(), aplicacao.EntradaConcluirValidacao{Uid: uid, IdValidacao: id, Leituras: leituras, CodigoRotacionado: rotacionado})
}

func (a *ambiente) iniciarOk(uid, chave string, bruto []byte) *aplicacao.ResultadoValidacao {
	a.t.Helper()
	r, err := a.iniciar(uid, chave, bruto)
	if err != nil {
		a.t.Fatalf("iniciar: %v", err)
	}
	return r
}

func (a *ambiente) concluirOk(uid, id string, leituras dominio.Leituras, rotacionado bool) *aplicacao.ResultadoConclusao {
	a.t.Helper()
	c, err := a.concluir(uid, id, leituras, rotacionado)
	if err != nil {
		a.t.Fatalf("concluir: %v", err)
	}
	return c
}

func (a *ambiente) consultarOk(uid, id string) *aplicacao.ResultadoValidacao {
	a.t.Helper()
	q, err := a.srv.ConsultarValidacao(context.Background(), uid, id)
	if err != nil {
		a.t.Fatalf("consultar: %v", err)
	}
	return q
}

func esperaCodigo(t *testing.T, err error, codigo dominio.Codigo) *dominio.Erro {
	t.Helper()
	if err == nil {
		t.Fatalf("esperava erro %s", codigo)
	}
	if got := dominio.CodigoDe(err); got != codigo {
		t.Fatalf("codigo = %s, esperava %s (%v)", got, codigo, err)
	}
	return dominio.ErroDe(err)
}

var leiturasPadrao = dominio.Leituras{Medidor: 14832, UnidadesVendidas: 137}

func TestFluxoPrincipal(t *testing.T) {
	amb := novoAmbiente(t)

	r := amb.iniciarOk("op-1", "chave-1", amb.payload(nil))
	if r.Validacao.Status != dominio.StatusAberta || r.Reproduzida {
		t.Fatalf("resultado inesperado: %+v", r)
	}
	if r.Contrassenha != amb.gerador.Derivar("VM-2047", 12) || r.ProximaContrassenha != amb.gerador.Derivar("VM-2047", 13) {
		t.Fatalf("contrassenhas = %s/%s", r.Contrassenha, r.ProximaContrassenha)
	}
	if r.Maquina.Id != "VM-2047" || r.Validacao.Eventos.IniciadaEm == nil {
		t.Fatalf("máquina ou marcador de evento ausentes: %+v", r)
	}

	amb.rel.Avancar(10 * time.Minute)
	c := amb.concluirOk("op-1", r.Validacao.Id, leiturasPadrao, true)
	if c.Validacao.Status != dominio.StatusConcluida || c.Maquina.ContadorCodigo != 13 || c.Maquina.ValidacaoAbertaId != "" {
		t.Fatalf("conclusão inesperada: %+v %+v", c.Validacao, c.Maquina)
	}

	eventos := amb.pub.Publicados()
	if len(eventos) != 2 || eventos[0].Tipo != dominio.EventoValidacaoIniciada || eventos[1].Tipo != dominio.EventoValidacaoConcluida {
		t.Fatalf("eventos publicados: %+v", eventos)
	}
	if eventos[1].Validacao.Leituras == nil || eventos[1].Validacao.Leituras.Medidor != 14832 || eventos[1].Atributos()["idMaquina"] != "VM-2047" {
		t.Fatalf("evento concluida sem leituras ou atributos: %+v", eventos[1])
	}

	q := amb.consultarOk("op-1", r.Validacao.Id)
	if q.Validacao.Status != dominio.StatusConcluida || q.Contrassenha != "" || q.ProximaContrassenha != "" {
		t.Fatalf("consulta após conclusão não deve expor contrassenhas: %+v", q)
	}

	r2 := amb.iniciarOk("op-1", "chave-2", amb.payload(nil))
	if r2.Contrassenha != amb.gerador.Derivar("VM-2047", 13) {
		t.Fatal("a visita seguinte deve usar o contador rotacionado")
	}
}

func TestIdempotencia(t *testing.T) {
	amb := novoAmbiente(t)
	bruto := amb.payload(nil)

	r1 := amb.iniciarOk("op-1", "chave-1", bruto)
	r2 := amb.iniciarOk("op-1", "chave-1", bruto)
	if !r2.Reproduzida || r2.Validacao.Id != r1.Validacao.Id || r2.Contrassenha != r1.Contrassenha {
		t.Fatalf("reenvio deveria reproduzir a resposta: %+v", r2)
	}
	if len(amb.pub.Publicados()) != 1 {
		t.Fatal("reenvio não deve publicar novo evento")
	}

	outro := amb.payload(func(p *dominio.PayloadQr) { p.Exp = "2028-01-01" })
	e := esperaCodigo(t, errAo(amb.iniciar("op-1", "chave-1", outro)), dominio.CodigoChaveIdempotenciaConflitante)
	if e.IdValidacao != r1.Validacao.Id {
		t.Fatalf("erro deveria apontar a validação existente: %+v", e)
	}
	esperaCodigo(t, errAo(amb.iniciar("op-2", "chave-1", bruto)), dominio.CodigoChaveIdempotenciaConflitante)
}

func TestValidacaoEmAndamento(t *testing.T) {
	amb := novoAmbiente(t)
	r1 := amb.iniciarOk("op-1", "chave-1", amb.payload(nil))
	e := esperaCodigo(t, errAo(amb.iniciar("op-2", "chave-2", amb.payload(nil))), dominio.CodigoValidacaoEmAndamento)
	if e.IdValidacao != r1.Validacao.Id {
		t.Fatalf("erro deveria apontar a validação aberta: %+v", e)
	}
}

func TestExpiracao(t *testing.T) {
	amb := novoAmbiente(t)
	r1 := amb.iniciarOk("op-1", "chave-1", amb.payload(nil))

	amb.rel.Avancar(15 * time.Minute)
	esperaCodigo(t, errAo(amb.concluir("op-1", r1.Validacao.Id, leiturasPadrao, true)), dominio.CodigoValidacaoExpirada)

	q := amb.consultarOk("op-1", r1.Validacao.Id)
	if q.Validacao.Status != dominio.StatusExpirada || q.Contrassenha != "" {
		t.Fatalf("consulta deveria mostrar expirada sem contrassenha: %+v", q)
	}

	r2 := amb.iniciarOk("op-2", "chave-3", amb.payload(nil))
	if r2.Validacao.Id == r1.Validacao.Id || r2.Contrassenha != amb.gerador.Derivar("VM-2047", 12) {
		t.Fatal("após expirar, nova validação deve ser aceita sem rotacionar o contador")
	}
}

func TestOperadorNaoAutorizado(t *testing.T) {
	amb := novoAmbiente(t)
	for _, uid := range []string{"op-inativo", "desconhecido", ""} {
		esperaCodigo(t, errAo(amb.iniciar(uid, "chave-x", amb.payload(nil))), dominio.CodigoOperadorNaoAutorizado)
	}
	r := amb.iniciarOk("op-1", "chave-1", amb.payload(nil))
	esperaCodigo(t, errAo(amb.concluir("op-2", r.Validacao.Id, leiturasPadrao, true)), dominio.CodigoOperadorNaoAutorizado)
	esperaCodigo(t, errAo(amb.srv.ConsultarValidacao(context.Background(), "op-2", r.Validacao.Id)), dominio.CodigoOperadorNaoAutorizado)
	esperaCodigo(t, errAo(amb.srv.ConsultarValidacao(context.Background(), "op-1", "abc")), dominio.CodigoValidacaoNaoEncontrada)
	esperaCodigo(t, errAo(amb.srv.ConsultarValidacao(context.Background(), "op-1", "01J8ZK3V9Q7XW2N4M6P8R0T2Y4")), dominio.CodigoValidacaoNaoEncontrada)
}

func TestRejeicoesDoAdesivo(t *testing.T) {
	amb := novoAmbiente(t)

	adulterado := amb.payload(nil)
	var p dominio.PayloadQr
	if err := json.Unmarshal(adulterado, &p); err != nil {
		t.Fatal(err)
	}
	p.Loc = "BLB-1"
	adulterado, _ = json.Marshal(p)

	casos := []struct {
		nome   string
		bruto  []byte
		chave  string
		codigo dominio.Codigo
	}{
		{"máquina desconhecida", amb.payload(func(p *dominio.PayloadQr) { p.Maq = "VM-9999" }), "c1", dominio.CodigoMaquinaDesconhecida},
		{"máquina inativa", amb.payload(func(p *dominio.PayloadQr) { p.Maq = "VM-0009"; p.Loc = "BLB-1" }), "c2", dominio.CodigoMaquinaInativa},
		{"adesivo trocado de máquina", amb.payload(func(p *dominio.PayloadQr) { p.Loc = "BLB-1" }), "c3", dominio.CodigoAssinaturaInvalida},
		{"adesivo vencido", amb.payload(func(p *dominio.PayloadQr) { p.Exp = "2025-12-31" }), "c4", dominio.CodigoQrExpirado},
		{"kid desconhecido", amb.payload(func(p *dominio.PayloadQr) { p.Kid = "k9" }), "c5", dominio.CodigoChaveQrDesconhecida},
		{"kid inativo", amb.payload(func(p *dominio.PayloadQr) { p.Kid = "k0" }), "c6", dominio.CodigoChaveQrDesconhecida},
		{"assinatura adulterada", adulterado, "c7", dominio.CodigoAssinaturaInvalida},
		{"sem Idempotency-Key", amb.payload(nil), "", dominio.CodigoPayloadInvalido},
		{"json inválido", []byte("{"), "c8", dominio.CodigoPayloadInvalido},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			esperaCodigo(t, errAo(amb.iniciar("op-1", c.chave, c.bruto)), c.codigo)
		})
	}
	if len(amb.pub.Publicados()) != 0 {
		t.Fatal("rejeições não devem publicar eventos")
	}
}

func TestPublicadorIndisponivelNaoBloqueiaOperador(t *testing.T) {
	amb := novoAmbiente(t)
	amb.pub.Falha = errors.New("pub/sub indisponível")

	r := amb.iniciarOk("op-1", "chave-1", amb.payload(nil))
	if r.Contrassenha == "" || r.Validacao.Eventos.IniciadaEm != nil {
		t.Fatalf("a validação deve prosseguir e ficar com publicação pendente: %+v", r)
	}
	if len(amb.pub.Publicados()) != 0 {
		t.Fatal("nada deveria ter sido publicado")
	}

	amb.pub.Falha = nil
	c := amb.concluirOk("op-1", r.Validacao.Id, leiturasPadrao, true)
	if c.Validacao.Eventos.ConcluidaEm == nil {
		t.Fatal("com o publicador de volta, a conclusão deve registrar a publicação")
	}
}

func TestConclusaoIdempotente(t *testing.T) {
	amb := novoAmbiente(t)
	r := amb.iniciarOk("op-1", "chave-1", amb.payload(nil))

	c1 := amb.concluirOk("op-1", r.Validacao.Id, leiturasPadrao, true)
	c2 := amb.concluirOk("op-1", r.Validacao.Id, leiturasPadrao, true)
	if !c2.Reproduzida || c1.Maquina.ContadorCodigo != 13 || c2.Maquina.ContadorCodigo != 13 {
		t.Fatalf("reenvio da conclusão não deve rotacionar de novo: %+v", c2.Maquina)
	}
	if len(amb.pub.Publicados()) != 2 {
		t.Fatal("reenvio não deve publicar novo evento")
	}
	esperaCodigo(t, errAo(amb.concluir("op-1", r.Validacao.Id, dominio.Leituras{Medidor: 1}, true)), dominio.CodigoValidacaoJaConcluida)
}

func TestSemRotacaoMantemCodigos(t *testing.T) {
	amb := novoAmbiente(t)
	r := amb.iniciarOk("op-1", "chave-1", amb.payload(nil))
	c := amb.concluirOk("op-1", r.Validacao.Id, leiturasPadrao, false)
	if c.Maquina.ContadorCodigo != 12 {
		t.Fatalf("contador = %d, não deveria rotacionar", c.Maquina.ContadorCodigo)
	}
	r2 := amb.iniciarOk("op-1", "chave-2", amb.payload(nil))
	if r2.Contrassenha != r.Contrassenha {
		t.Fatal("sem rotação, a visita seguinte deve devolver o mesmo código")
	}
}

func TestMedidorMenorQueAnteriorGeraAviso(t *testing.T) {
	amb := novoAmbiente(t)
	outraMaquina := func(p *dominio.PayloadQr) { p.Maq = "VM-3000"; p.Loc = "BLC-1" }

	r := amb.iniciarOk("op-1", "chave-1", amb.payload(outraMaquina))
	c := amb.concluirOk("op-1", r.Validacao.Id, dominio.Leituras{Medidor: 14832, UnidadesVendidas: 10}, true)
	if len(c.Validacao.Avisos) != 1 || c.Validacao.Avisos[0] != dominio.AvisoMedidorMenorQueAnterior {
		t.Fatalf("avisos = %v, esperava medidor_menor_que_anterior", c.Validacao.Avisos)
	}
	eventos := amb.pub.Publicados()
	if ultimo := eventos[len(eventos)-1]; len(ultimo.Validacao.Avisos) != 1 {
		t.Fatalf("o evento concluida deveria carregar o aviso: %+v", ultimo.Validacao)
	}
	if c.Maquina.UltimoMedidor == nil || *c.Maquina.UltimoMedidor != 14832 {
		t.Fatal("a conclusão com aviso ainda atualiza o último medidor")
	}

	r2 := amb.iniciarOk("op-1", "chave-2", amb.payload(outraMaquina))
	c2 := amb.concluirOk("op-1", r2.Validacao.Id, dominio.Leituras{Medidor: 15000, UnidadesVendidas: 10}, true)
	if len(c2.Validacao.Avisos) != 0 {
		t.Fatalf("medidor maior não deve gerar aviso: %v", c2.Validacao.Avisos)
	}
}

// errAo descarta o valor e devolve só o erro, para uso com esperaCodigo.
func errAo[T any](_ T, err error) error { return err }
