package dominio

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"time"
)

// VersaoPayloadQr é a única versão de esquema aceita nesta versão do serviço.
const VersaoPayloadQr = 1

// ModelosSuportados lista os modelos de máquina aceitos (RN-08).
var ModelosSuportados = map[string]bool{"CN168": true}

var (
	padraoMaq = regexp.MustCompile(`^[A-Z0-9-]{3,32}$`)
	padraoLoc = regexp.MustCompile(`^[A-Z0-9-]{1,32}$`)
	padraoKid = regexp.MustCompile(`^[a-z0-9]{1,8}$`)
	padraoSig = regexp.MustCompile(`^[A-Za-z0-9_-]{86}$`)
)

// PayloadQr é o conteúdo decodificado do adesivo fixado na máquina
// (docs/payload-qr.md). É estático: nada nele muda com o tempo.
type PayloadQr struct {
	V   int    `json:"v"`
	Maq string `json:"maq"`
	Mod string `json:"mod"`
	Loc string `json:"loc"`
	Exp string `json:"exp"`
	Kid string `json:"kid"`
	Sig string `json:"sig"`
}

// DecodificarPayloadQr converte o JSON lido do QR em PayloadQr. Rejeita campos
// desconhecidos, conteúdo além do objeto e violações de formato.
func DecodificarPayloadQr(bruto []byte) (*PayloadQr, error) {
	dec := json.NewDecoder(bytes.NewReader(bruto))
	dec.DisallowUnknownFields()

	var p PayloadQr
	if err := dec.Decode(&p); err != nil {
		return nil, NovoErro(CodigoPayloadInvalido, "o conteúdo do QR não segue o esquema esperado").ComCausa(err)
	}
	if dec.More() {
		return nil, NovoErro(CodigoPayloadInvalido, "o conteúdo do QR tem dados além do objeto JSON")
	}
	if err := p.Validar(); err != nil {
		return nil, err
	}
	return &p, nil
}

// Validar confere estrutura e formato dos campos. Não confere validade
// temporal nem assinatura; ver Vencido e VerificarAssinatura.
func (p *PayloadQr) Validar() error {
	switch {
	case p.V != VersaoPayloadQr:
		return NovoErro(CodigoPayloadInvalido, fmt.Sprintf("versão %d do payload não é suportada", p.V))
	case !padraoMaq.MatchString(p.Maq):
		return NovoErro(CodigoPayloadInvalido, "campo maq fora do formato esperado")
	case !padraoLoc.MatchString(p.Loc):
		return NovoErro(CodigoPayloadInvalido, "campo loc fora do formato esperado")
	case !padraoKid.MatchString(p.Kid):
		return NovoErro(CodigoPayloadInvalido, "campo kid fora do formato esperado")
	case !padraoSig.MatchString(p.Sig):
		return NovoErro(CodigoPayloadInvalido, "campo sig fora do formato esperado")
	}
	if _, err := p.DataValidade(); err != nil {
		return NovoErro(CodigoPayloadInvalido, "campo exp deve ser uma data AAAA-MM-DD").ComCausa(err)
	}
	if !ModelosSuportados[p.Mod] {
		return NovoErro(CodigoModeloNaoSuportado, fmt.Sprintf("modelo %q não é suportado", p.Mod))
	}
	return nil
}

// DataValidade interpreta exp (AAAA-MM-DD) como uma data em UTC.
func (p *PayloadQr) DataValidade() (time.Time, error) {
	return time.Parse("2006-01-02", p.Exp)
}

// Vencido informa se o adesivo já venceu na data de referência. O adesivo vale
// até o fim do dia indicado em exp (UTC).
func (p *PayloadQr) Vencido(referencia time.Time) bool {
	exp, err := p.DataValidade()
	if err != nil {
		return true
	}
	r := referencia.UTC()
	dia := time.Date(r.Year(), r.Month(), r.Day(), 0, 0, 0, 0, time.UTC)
	return dia.After(exp)
}

// MensagemCanonica é o texto coberto pela assinatura: v|maq|mod|loc|exp|kid.
func (p *PayloadQr) MensagemCanonica() []byte {
	return []byte(fmt.Sprintf("%d|%s|%s|%s|%s|%s", p.V, p.Maq, p.Mod, p.Loc, p.Exp, p.Kid))
}

// VerificarAssinatura confere sig contra a mensagem canônica usando a chave
// pública Ed25519 identificada por kid.
func (p *PayloadQr) VerificarAssinatura(chavePublica ed25519.PublicKey) error {
	if len(chavePublica) != ed25519.PublicKeySize {
		return NovoErro(CodigoChaveQrDesconhecida, "chave pública do adesivo inválida")
	}
	sig, err := base64.RawURLEncoding.DecodeString(p.Sig)
	if err != nil || len(sig) != ed25519.SignatureSize {
		return NovoErro(CodigoAssinaturaInvalida, "assinatura do adesivo mal formada")
	}
	if !ed25519.Verify(chavePublica, p.MensagemCanonica(), sig) {
		return NovoErro(CodigoAssinaturaInvalida, "a assinatura do adesivo não corresponde aos campos")
	}
	return nil
}

// Assinar preenche sig com a assinatura Ed25519 da mensagem canônica. Usado
// pela ferramenta gerar-qr e pelos testes; o serviço nunca assina.
func (p *PayloadQr) Assinar(chavePrivada ed25519.PrivateKey) {
	p.Sig = base64.RawURLEncoding.EncodeToString(ed25519.Sign(chavePrivada, p.MensagemCanonica()))
}

// HashRequisicao resume o conteúdo relevante de uma requisição de validação.
// Serve à idempotência: a mesma Idempotency-Key com hash diferente é conflito.
func (p *PayloadQr) HashRequisicao() string {
	texto := append(p.MensagemCanonica(), []byte("|"+p.Sig)...)
	soma := sha256.Sum256(texto)
	return hex.EncodeToString(soma[:])
}

// ChaveQr é uma chave pública usada para assinar adesivos (coleção chavesQr).
type ChaveQr struct {
	Kid          string
	ChavePublica ed25519.PublicKey
	Ativa        bool
	ValidaDe     time.Time
	ValidaAte    time.Time
}
