package armazenamento

import (
	"fmt"
	"time"

	"cloud.google.com/go/firestore"

	"github.com/ThiagoCristovao/serverless-vending-validation/servico/internal/dominio"
)

// Formas dos documentos do Firestore (docs/modelo-dados.md). Ponteiros nulos
// viram `null` no documento, como o modelo prevê.

type docLocalizacao struct {
	Id   string `firestore:"id"`
	Nome string `firestore:"nome"`
}

type docMaquina struct {
	Modelo            string         `firestore:"modelo"`
	Localizacao       docLocalizacao `firestore:"localizacao"`
	Ativa             bool           `firestore:"ativa"`
	ContadorCodigo    int64          `firestore:"contadorCodigo"`
	ValidacaoAbertaId *string        `firestore:"validacaoAbertaId"`
	UltimaValidacaoId *string        `firestore:"ultimaValidacaoId"`
	UltimaValidacaoEm *time.Time     `firestore:"ultimaValidacaoEm"`
	UltimoMedidor     *int64         `firestore:"ultimoMedidor"`
	CriadaEm          time.Time      `firestore:"criadaEm"`
	AtualizadaEm      time.Time      `firestore:"atualizadaEm"`
}

type docOperador struct {
	Nome     string    `firestore:"nome"`
	Email    string    `firestore:"email"`
	Rota     *string   `firestore:"rota"`
	Ativo    bool      `firestore:"ativo"`
	CriadoEm time.Time `firestore:"criadoEm"`
}

type docChaveQr struct {
	Algoritmo    string    `firestore:"algoritmo"`
	ChavePublica string    `firestore:"chavePublica"`
	Ativa        bool      `firestore:"ativa"`
	ValidaDe     time.Time `firestore:"validaDe"`
	ValidaAte    time.Time `firestore:"validaAte"`
}

type docOrigemQr struct {
	Kid string `firestore:"kid"`
	Exp string `firestore:"exp"`
	Loc string `firestore:"loc"`
	Mod string `firestore:"mod"`
}

type docLeituras struct {
	Medidor          int64 `firestore:"medidor"`
	UnidadesVendidas int64 `firestore:"unidadesVendidas"`
}

type docDispositivo struct {
	Plataforma string `firestore:"plataforma"`
	VersaoApp  string `firestore:"versaoApp"`
}

type docEventos struct {
	IniciadaPublicadaEm  *time.Time `firestore:"iniciadaPublicadaEm"`
	ConcluidaPublicadaEm *time.Time `firestore:"concluidaPublicadaEm"`
}

type docValidacao struct {
	IdMaquina         string          `firestore:"idMaquina"`
	IdOperador        string          `firestore:"idOperador"`
	Status            string          `firestore:"status"`
	IniciadaEm        time.Time       `firestore:"iniciadaEm"`
	ExpiraEm          time.Time       `firestore:"expiraEm"`
	ConcluidaEm       *time.Time      `firestore:"concluidaEm"`
	ContadorCodigo    int64           `firestore:"contadorCodigo"`
	ChaveIdempotencia string          `firestore:"chaveIdempotencia"`
	HashRequisicao    string          `firestore:"hashRequisicao"`
	Qr                docOrigemQr     `firestore:"qr"`
	Leituras          *docLeituras    `firestore:"leituras"`
	CodigoRotacionado *bool           `firestore:"codigoRotacionado"`
	Observacoes       *string         `firestore:"observacoes"`
	Avisos            []string        `firestore:"avisos"`
	Eventos           docEventos      `firestore:"eventos"`
	Dispositivo       *docDispositivo `firestore:"dispositivo"`
}

func valor(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func ponteiroSeNaoVazio(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func maquinaDe(snap *firestore.DocumentSnapshot) (*dominio.Maquina, error) {
	var d docMaquina
	if err := snap.DataTo(&d); err != nil {
		return nil, fmt.Errorf("firestore: decodificar máquina %s: %w", snap.Ref.ID, err)
	}
	if d.ContadorCodigo < 0 {
		return nil, fmt.Errorf("firestore: máquina %s com contador negativo", snap.Ref.ID)
	}
	return &dominio.Maquina{
		Id:                snap.Ref.ID,
		Modelo:            d.Modelo,
		Localizacao:       dominio.Localizacao{Id: d.Localizacao.Id, Nome: d.Localizacao.Nome},
		Ativa:             d.Ativa,
		ContadorCodigo:    uint64(d.ContadorCodigo),
		ValidacaoAbertaId: valor(d.ValidacaoAbertaId),
		UltimaValidacaoId: valor(d.UltimaValidacaoId),
		UltimaValidacaoEm: d.UltimaValidacaoEm,
		UltimoMedidor:     d.UltimoMedidor,
		CriadaEm:          d.CriadaEm,
		AtualizadaEm:      d.AtualizadaEm,
	}, nil
}

func docDeMaquina(m *dominio.Maquina) docMaquina {
	return docMaquina{
		Modelo:            m.Modelo,
		Localizacao:       docLocalizacao{Id: m.Localizacao.Id, Nome: m.Localizacao.Nome},
		Ativa:             m.Ativa,
		ContadorCodigo:    int64(m.ContadorCodigo),
		ValidacaoAbertaId: ponteiroSeNaoVazio(m.ValidacaoAbertaId),
		UltimaValidacaoId: ponteiroSeNaoVazio(m.UltimaValidacaoId),
		UltimaValidacaoEm: m.UltimaValidacaoEm,
		UltimoMedidor:     m.UltimoMedidor,
		CriadaEm:          m.CriadaEm,
		AtualizadaEm:      m.AtualizadaEm,
	}
}

func validacaoDe(snap *firestore.DocumentSnapshot) (*dominio.Validacao, error) {
	var d docValidacao
	if err := snap.DataTo(&d); err != nil {
		return nil, fmt.Errorf("firestore: decodificar validação %s: %w", snap.Ref.ID, err)
	}
	v := &dominio.Validacao{
		Id:                snap.Ref.ID,
		IdMaquina:         d.IdMaquina,
		IdOperador:        d.IdOperador,
		Status:            dominio.StatusValidacao(d.Status),
		IniciadaEm:        d.IniciadaEm,
		ExpiraEm:          d.ExpiraEm,
		ConcluidaEm:       d.ConcluidaEm,
		ContadorCodigo:    uint64(max(d.ContadorCodigo, 0)),
		ChaveIdempotencia: d.ChaveIdempotencia,
		HashRequisicao:    d.HashRequisicao,
		Qr:                dominio.OrigemQr{Kid: d.Qr.Kid, Exp: d.Qr.Exp, Loc: d.Qr.Loc, Mod: d.Qr.Mod},
		CodigoRotacionado: d.CodigoRotacionado,
		Observacoes:       valor(d.Observacoes),
		Eventos:           dominio.EventosPublicados{IniciadaEm: d.Eventos.IniciadaPublicadaEm, ConcluidaEm: d.Eventos.ConcluidaPublicadaEm},
	}
	if d.Leituras != nil {
		v.Leituras = &dominio.Leituras{Medidor: d.Leituras.Medidor, UnidadesVendidas: d.Leituras.UnidadesVendidas}
	}
	if d.Dispositivo != nil {
		v.Dispositivo = &dominio.Dispositivo{Plataforma: d.Dispositivo.Plataforma, VersaoApp: d.Dispositivo.VersaoApp}
	}
	for _, a := range d.Avisos {
		v.Avisos = append(v.Avisos, dominio.Aviso(a))
	}
	return v, nil
}

func docDeValidacao(v *dominio.Validacao) docValidacao {
	d := docValidacao{
		IdMaquina:         v.IdMaquina,
		IdOperador:        v.IdOperador,
		Status:            string(v.Status),
		IniciadaEm:        v.IniciadaEm,
		ExpiraEm:          v.ExpiraEm,
		ConcluidaEm:       v.ConcluidaEm,
		ContadorCodigo:    int64(v.ContadorCodigo),
		ChaveIdempotencia: v.ChaveIdempotencia,
		HashRequisicao:    v.HashRequisicao,
		Qr:                docOrigemQr{Kid: v.Qr.Kid, Exp: v.Qr.Exp, Loc: v.Qr.Loc, Mod: v.Qr.Mod},
		CodigoRotacionado: v.CodigoRotacionado,
		Observacoes:       ponteiroSeNaoVazio(v.Observacoes),
		Avisos:            []string{},
		Eventos:           docEventos{IniciadaPublicadaEm: v.Eventos.IniciadaEm, ConcluidaPublicadaEm: v.Eventos.ConcluidaEm},
	}
	if v.Leituras != nil {
		d.Leituras = &docLeituras{Medidor: v.Leituras.Medidor, UnidadesVendidas: v.Leituras.UnidadesVendidas}
	}
	if v.Dispositivo != nil {
		d.Dispositivo = &docDispositivo{Plataforma: v.Dispositivo.Plataforma, VersaoApp: v.Dispositivo.VersaoApp}
	}
	for _, a := range v.Avisos {
		d.Avisos = append(d.Avisos, string(a))
	}
	return d
}
