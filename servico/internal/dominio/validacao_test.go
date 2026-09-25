package dominio

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

var instante = time.Date(2026, 9, 17, 13, 5, 12, 0, time.UTC)

func fixtureMaquina() *Maquina {
	return &Maquina{Id: "VM-2047", Modelo: "CN168", Localizacao: Localizacao{Id: "BLA-T", Nome: "Bloco A - Térreo"}, Ativa: true, ContadorCodigo: 12}
}

func fixtureOperador() *Operador {
	return &Operador{Uid: "op-1", Nome: "Operador Exemplo", Ativo: true}
}

func fixturePayload() *PayloadQr {
	return &PayloadQr{V: 1, Maq: "VM-2047", Mod: "CN168", Loc: "BLA-T", Exp: "2027-12-31", Kid: "k1", Sig: strings.Repeat("A", 86)}
}

func TestValidacao_CicloDeVida(t *testing.T) {
	m, o := fixtureMaquina(), fixtureOperador()
	v := NovaValidacao("01J8ZK3V9Q7XW2N4M6P8R0T2Y4", m, o, fixturePayload(), "chave-1", nil, instante, 0)

	if v.Status != StatusAberta || !v.Aberta() {
		t.Fatalf("status inicial = %s", v.Status)
	}
	if !v.ExpiraEm.Equal(instante.Add(PrazoValidacaoPadrao)) || v.ContadorCodigo != 12 || v.Qr.Kid != "k1" {
		t.Fatalf("campos iniciais inesperados: %+v", v)
	}
	if v.Expirou(instante.Add(14 * time.Minute)) {
		t.Fatal("não deveria ter expirado antes do prazo")
	}
	if !v.Expirou(instante.Add(15 * time.Minute)) {
		t.Fatal("deveria expirar exatamente no prazo")
	}

	if got := CodigoDe(v.Concluir(Leituras{Medidor: -1}, true, "", instante)); got != CodigoPayloadInvalido {
		t.Fatalf("leitura negativa: codigo = %s", got)
	}
	if got := CodigoDe(v.Concluir(Leituras{}, true, strings.Repeat("x", 501), instante)); got != CodigoPayloadInvalido {
		t.Fatalf("observações longas: codigo = %s", got)
	}

	fim := instante.Add(10 * time.Minute)
	leituras := Leituras{Medidor: 14832, UnidadesVendidas: 137}
	if err := v.Concluir(leituras, true, "ok", fim); err != nil {
		t.Fatalf("concluir: %v", err)
	}
	if v.Status != StatusConcluida || v.ConcluidaEm == nil || !v.ConcluidaEm.Equal(fim) {
		t.Fatalf("estado após conclusão: %+v", v)
	}
	if !v.MesmaConclusao(leituras, true) || v.MesmaConclusao(leituras, false) {
		t.Fatal("MesmaConclusao inconsistente")
	}
	if got := CodigoDe(v.Concluir(Leituras{}, false, "", fim)); got != CodigoValidacaoJaConcluida {
		t.Fatalf("segunda conclusão: codigo = %s", got)
	}

	m.RegistrarConclusao(v, fim)
	if m.ContadorCodigo != 13 {
		t.Fatalf("contador = %d, esperava 13 (rotação)", m.ContadorCodigo)
	}
	if m.ValidacaoAbertaId != "" || m.UltimaValidacaoId != v.Id || m.UltimoMedidor == nil || *m.UltimoMedidor != 14832 {
		t.Fatalf("efeitos na máquina inesperados: %+v", m)
	}
}

func TestValidacao_ExpiradaNaoConclui(t *testing.T) {
	v := NovaValidacao("01J8ZK3V9Q7XW2N4M6P8R0T2Y4", fixtureMaquina(), fixtureOperador(), fixturePayload(), "chave-1", nil, instante, 0)
	if got := CodigoDe(v.Concluir(Leituras{}, true, "", instante.Add(20*time.Minute))); got != CodigoValidacaoExpirada {
		t.Fatalf("codigo = %s", got)
	}
	v.MarcarExpirada()
	if v.Aberta() || v.Status != StatusExpirada {
		t.Fatal("MarcarExpirada não encerrou a validação")
	}
}

func TestValidacao_RegistrarAvisos(t *testing.T) {
	novaConcluida := func(medidor int64) *Validacao {
		v := NovaValidacao("01J8ZK3V9Q7XW2N4M6P8R0T2Y4", fixtureMaquina(), fixtureOperador(), fixturePayload(), "chave-1", nil, instante, 0)
		if err := v.Concluir(Leituras{Medidor: medidor, UnidadesVendidas: 1}, true, "", instante); err != nil {
			t.Fatal(err)
		}
		return v
	}
	semHistorico := fixtureMaquina()
	anterior := int64(20000)
	comHistorico := fixtureMaquina()
	comHistorico.UltimoMedidor = &anterior

	casos := []struct {
		nome    string
		maquina *Maquina
		medidor int64
		avisos  int
	}{
		{"sem histórico", semHistorico, 100, 0},
		{"medidor menor que o anterior", comHistorico, 19999, 1},
		{"medidor igual ao anterior", comHistorico, 20000, 0},
		{"medidor maior que o anterior", comHistorico, 20001, 0},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			v := novaConcluida(c.medidor)
			v.RegistrarAvisos(c.maquina)
			if len(v.Avisos) != c.avisos {
				t.Fatalf("avisos = %v, esperava %d", v.Avisos, c.avisos)
			}
			if c.avisos == 1 && v.Avisos[0] != AvisoMedidorMenorQueAnterior {
				t.Fatalf("aviso inesperado: %s", v.Avisos[0])
			}
		})
	}
}

func TestMaquina_SemRotacaoMantemContador(t *testing.T) {
	m := fixtureMaquina()
	v := NovaValidacao("01J8ZK3V9Q7XW2N4M6P8R0T2Y4", m, fixtureOperador(), fixturePayload(), "chave-1", nil, instante, 0)
	if err := v.Concluir(Leituras{Medidor: 1, UnidadesVendidas: 1}, false, "", instante); err != nil {
		t.Fatal(err)
	}
	m.RegistrarConclusao(v, instante)
	if m.ContadorCodigo != 12 {
		t.Fatalf("contador = %d, não deveria rotacionar", m.ContadorCodigo)
	}
}

func TestEventos(t *testing.T) {
	m, o := fixtureMaquina(), fixtureOperador()
	v := NovaValidacao("01J8ZK3V9Q7XW2N4M6P8R0T2Y4", m, o, fixturePayload(), "chave-1", nil, instante, 0)

	ini := NovoEventoIniciada(v, m, o)
	if ini.Tipo != EventoValidacaoIniciada || ini.Validacao.ExpiraEm == nil || ini.Validacao.Leituras != nil {
		t.Fatalf("evento iniciada inesperado: %+v", ini)
	}
	attrs := ini.Atributos()
	if attrs["tipo"] != "validacao.iniciada" || attrs["idMaquina"] != "VM-2047" || attrs["versao"] != "1" || ini.ChaveOrdenacao() != "VM-2047" {
		t.Fatalf("atributos inesperados: %v", attrs)
	}
	corpo, err := json.Marshal(ini)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.ToLower(string(corpo)), "contrassenha") {
		t.Fatal("a contrassenha nunca deve trafegar no evento")
	}

	fim := instante.Add(9 * time.Minute)
	_ = v.Concluir(Leituras{Medidor: 14832, UnidadesVendidas: 137}, true, "", fim)
	con := NovoEventoConcluida(v, m, o)
	if con.Tipo != EventoValidacaoConcluida || !con.OcorridoEm.Equal(fim) || con.Validacao.Leituras == nil || con.Validacao.Leituras.Medidor != 14832 {
		t.Fatalf("evento concluida inesperado: %+v", con)
	}
}
