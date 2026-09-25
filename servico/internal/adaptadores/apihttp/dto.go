package apihttp

import (
	"encoding/json"
	"time"

	"github.com/ThiagoCristovao/serverless-vending-validation/servico/internal/aplicacao"
	"github.com/ThiagoCristovao/serverless-vending-validation/servico/internal/dominio"
)

// Formas de requisição e resposta espelham api/openapi.yaml.

type requisicaoIniciar struct {
	Qr          json.RawMessage `json:"qr"`
	LidoEm      *time.Time      `json:"lidoEm"`
	Dispositivo *dispositivoDto `json:"dispositivo"`
}

type dispositivoDto struct {
	Plataforma string `json:"plataforma"`
	VersaoApp  string `json:"versaoApp"`
}

type requisicaoConcluir struct {
	Leituras          *leiturasDto `json:"leituras"`
	CodigoRotacionado *bool        `json:"codigoRotacionado"`
	Observacoes       string       `json:"observacoes"`
}

type leiturasDto struct {
	Medidor          int64 `json:"medidor"`
	UnidadesVendidas int64 `json:"unidadesVendidas"`
}

type localizacaoDto struct {
	Id   string `json:"id"`
	Nome string `json:"nome"`
}

type maquinaDto struct {
	Id                string         `json:"id"`
	Modelo            string         `json:"modelo"`
	Localizacao       localizacaoDto `json:"localizacao"`
	UltimaValidacaoEm *time.Time     `json:"ultimaValidacaoEm"`
}

type validacaoDto struct {
	IdValidacao         string       `json:"idValidacao"`
	Status              string       `json:"status"`
	Contrassenha        string       `json:"contrassenha,omitempty"`
	ProximaContrassenha string       `json:"proximaContrassenha,omitempty"`
	IniciadaEm          time.Time    `json:"iniciadaEm"`
	ExpiraEm            time.Time    `json:"expiraEm"`
	ConcluidaEm         *time.Time   `json:"concluidaEm"`
	Maquina             maquinaDto   `json:"maquina"`
	Leituras            *leiturasDto `json:"leituras,omitempty"`
	CodigoRotacionado   *bool        `json:"codigoRotacionado,omitempty"`
	Avisos              []string     `json:"avisos,omitempty"`
}

type conclusaoDto struct {
	IdValidacao       string    `json:"idValidacao"`
	Status            string    `json:"status"`
	ConcluidaEm       time.Time `json:"concluidaEm"`
	CodigoRotacionado bool      `json:"codigoRotacionado"`
	Avisos            []string  `json:"avisos"`
}

func utc(t time.Time) time.Time { return t.UTC() }

func utcOuNulo(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	u := t.UTC()
	return &u
}

func avisosDto(avisos []dominio.Aviso) []string {
	saida := make([]string, 0, len(avisos))
	for _, a := range avisos {
		saida = append(saida, string(a))
	}
	return saida
}

func maquinaParaDto(m *dominio.Maquina) maquinaDto {
	return maquinaDto{
		Id:                m.Id,
		Modelo:            m.Modelo,
		Localizacao:       localizacaoDto{Id: m.Localizacao.Id, Nome: m.Localizacao.Nome},
		UltimaValidacaoEm: utcOuNulo(m.UltimaValidacaoEm),
	}
}

func validacaoParaDto(r *aplicacao.ResultadoValidacao) validacaoDto {
	v := r.Validacao
	d := validacaoDto{
		IdValidacao:         v.Id,
		Status:              string(v.Status),
		Contrassenha:        r.Contrassenha,
		ProximaContrassenha: r.ProximaContrassenha,
		IniciadaEm:          utc(v.IniciadaEm),
		ExpiraEm:            utc(v.ExpiraEm),
		ConcluidaEm:         utcOuNulo(v.ConcluidaEm),
		Maquina:             maquinaParaDto(r.Maquina),
		CodigoRotacionado:   v.CodigoRotacionado,
	}
	if v.Leituras != nil {
		d.Leituras = &leiturasDto{Medidor: v.Leituras.Medidor, UnidadesVendidas: v.Leituras.UnidadesVendidas}
	}
	if len(v.Avisos) > 0 {
		d.Avisos = avisosDto(v.Avisos)
	}
	return d
}

func conclusaoParaDto(r *aplicacao.ResultadoConclusao) conclusaoDto {
	v := r.Validacao
	d := conclusaoDto{
		IdValidacao: v.Id,
		Status:      string(v.Status),
		Avisos:      avisosDto(v.Avisos),
	}
	if v.ConcluidaEm != nil {
		d.ConcluidaEm = utc(*v.ConcluidaEm)
	}
	if v.CodigoRotacionado != nil {
		d.CodigoRotacionado = *v.CodigoRotacionado
	}
	return d
}
