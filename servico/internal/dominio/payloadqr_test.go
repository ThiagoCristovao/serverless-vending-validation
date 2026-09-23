package dominio

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func novoParDeChaves(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("gerar chaves: %v", err)
	}
	return pub, priv
}

func payloadAssinado(t *testing.T, priv ed25519.PrivateKey) PayloadQr {
	t.Helper()
	p := PayloadQr{V: 1, Maq: "VM-2047", Mod: "CN168", Loc: "BLA-T", Exp: "2027-12-31", Kid: "k1"}
	p.Assinar(priv)
	return p
}

func jsonDe(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("serializar: %v", err)
	}
	return b
}

func TestDecodificarPayloadQr_Valido(t *testing.T) {
	_, priv := novoParDeChaves(t)
	p := payloadAssinado(t, priv)

	got, err := DecodificarPayloadQr(jsonDe(t, p))
	if err != nil {
		t.Fatalf("payload válido rejeitado: %v", err)
	}
	if *got != p {
		t.Fatalf("decodificado = %+v, esperava %+v", *got, p)
	}
	if len(p.Sig) != 86 {
		t.Fatalf("assinatura com %d caracteres, esperava 86", len(p.Sig))
	}
}

func TestDecodificarPayloadQr_Rejeicoes(t *testing.T) {
	_, priv := novoParDeChaves(t)
	base := payloadAssinado(t, priv)
	altera := func(f func(p *PayloadQr)) []byte {
		p := base
		f(&p)
		return jsonDe(t, p)
	}
	comExtra := strings.TrimSuffix(string(jsonDe(t, base)), "}") + `,"extra":1}`

	casos := []struct {
		nome   string
		bruto  []byte
		codigo Codigo
	}{
		{"json inválido", []byte(`{"v":1,`), CodigoPayloadInvalido},
		{"campo desconhecido", []byte(comExtra), CodigoPayloadInvalido},
		{"conteúdo além do objeto", append(jsonDe(t, base), []byte(` {}`)...), CodigoPayloadInvalido},
		{"versão não suportada", altera(func(p *PayloadQr) { p.V = 2 }), CodigoPayloadInvalido},
		{"maq em minúsculas", altera(func(p *PayloadQr) { p.Maq = "vm-1" }), CodigoPayloadInvalido},
		{"loc vazio", altera(func(p *PayloadQr) { p.Loc = "" }), CodigoPayloadInvalido},
		{"kid em maiúsculas", altera(func(p *PayloadQr) { p.Kid = "K1" }), CodigoPayloadInvalido},
		{"sig curta", altera(func(p *PayloadQr) { p.Sig = "abc" }), CodigoPayloadInvalido},
		{"exp fora do formato", altera(func(p *PayloadQr) { p.Exp = "31/12/2027" }), CodigoPayloadInvalido},
		{"modelo não suportado", altera(func(p *PayloadQr) { p.Mod = "CN167" }), CodigoModeloNaoSuportado},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			_, err := DecodificarPayloadQr(c.bruto)
			if err == nil {
				t.Fatal("esperava erro")
			}
			if got := CodigoDe(err); got != c.codigo {
				t.Fatalf("codigo = %s, esperava %s (%v)", got, c.codigo, err)
			}
		})
	}
}

func TestVerificarAssinatura(t *testing.T) {
	pub, priv := novoParDeChaves(t)
	outraPub, _ := novoParDeChaves(t)
	p := payloadAssinado(t, priv)

	if err := p.VerificarAssinatura(pub); err != nil {
		t.Fatalf("assinatura válida rejeitada: %v", err)
	}

	adulterado := p
	adulterado.Loc = "BLB-T"
	if got := CodigoDe(adulterado.VerificarAssinatura(pub)); got != CodigoAssinaturaInvalida {
		t.Fatalf("campo adulterado: codigo = %s", got)
	}
	if got := CodigoDe(p.VerificarAssinatura(outraPub)); got != CodigoAssinaturaInvalida {
		t.Fatalf("outra chave: codigo = %s", got)
	}
	if got := CodigoDe(p.VerificarAssinatura(ed25519.PublicKey([]byte{1, 2, 3}))); got != CodigoChaveQrDesconhecida {
		t.Fatalf("chave pública inválida: codigo = %s", got)
	}
	malFormada := p
	malFormada.Sig = strings.Repeat("A", 85) + "!"
	if got := CodigoDe(malFormada.VerificarAssinatura(pub)); got != CodigoAssinaturaInvalida {
		t.Fatalf("assinatura mal formada: codigo = %s", got)
	}
}

func TestVencido(t *testing.T) {
	p := PayloadQr{Exp: "2027-12-31"}
	casos := []struct {
		nome    string
		ref     time.Time
		vencido bool
	}{
		{"véspera", time.Date(2027, 12, 30, 12, 0, 0, 0, time.UTC), false},
		{"último dia, 23:59 UTC", time.Date(2027, 12, 31, 23, 59, 59, 0, time.UTC), false},
		{"dia seguinte, 00:00 UTC", time.Date(2028, 1, 1, 0, 0, 0, 0, time.UTC), true},
		{"muito depois", time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC), true},
	}
	for _, c := range casos {
		if got := p.Vencido(c.ref); got != c.vencido {
			t.Errorf("%s: vencido = %v, esperava %v", c.nome, got, c.vencido)
		}
	}
	if !(&PayloadQr{Exp: "inválida"}).Vencido(time.Now()) {
		t.Error("exp inválido deve contar como vencido")
	}
}

func TestHashRequisicao(t *testing.T) {
	_, priv := novoParDeChaves(t)
	p := payloadAssinado(t, priv)
	compacto := jsonDe(t, p)
	espacado := []byte(strings.ReplaceAll(string(compacto), ",", ", "))

	a, err := DecodificarPayloadQr(compacto)
	if err != nil {
		t.Fatal(err)
	}
	b, err := DecodificarPayloadQr(espacado)
	if err != nil {
		t.Fatal(err)
	}
	if a.HashRequisicao() != b.HashRequisicao() {
		t.Fatal("mesmo conteúdo com espaçamento diferente deve ter o mesmo hash")
	}
	outro := p
	outro.Sig = strings.Repeat("A", 86)
	if outro.HashRequisicao() == p.HashRequisicao() {
		t.Fatal("assinaturas diferentes devem ter hashes diferentes")
	}
}
