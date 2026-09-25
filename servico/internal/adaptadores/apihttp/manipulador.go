package apihttp

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strings"

	"github.com/ThiagoCristovao/serverless-vending-validation/servico/internal/aplicacao"
	"github.com/ThiagoCristovao/serverless-vending-validation/servico/internal/dominio"
)

// TamanhoMaximoCorpo limita o corpo das requisições (o payload do QR tem
// poucas centenas de bytes).
const TamanhoMaximoCorpo = 64 << 10

var padraoUuid = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// Opcoes configura o manipulador HTTP.
type Opcoes struct {
	// Registrador recebe uma linha por requisição; nil usa slog.Default().
	Registrador *slog.Logger
	// ProjetoId compõe o campo de trace do Cloud Logging; opcional.
	ProjetoId string
	// CabecalhoOperadorDeTeste, quando não vazio, permite identificar o operador
	// por um cabeçalho simples em execução local. Nunca definir em produção.
	CabecalhoOperadorDeTeste string
}

type manipulador struct {
	s *aplicacao.Servico
	o Opcoes
}

// NovoManipulador monta as rotas do contrato sobre o serviço de aplicação.
func NovoManipulador(s *aplicacao.Servico, o Opcoes) http.Handler {
	if o.Registrador == nil {
		o.Registrador = slog.Default()
	}
	m := &manipulador{s: s, o: o}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /saude", m.saude)
	mux.HandleFunc("POST /v1/validacoes", m.iniciar)
	mux.HandleFunc("GET /v1/validacoes/{idValidacao}", m.consultar)
	mux.HandleFunc("POST /v1/validacoes/{idValidacao}/conclusao", m.concluir)
	// Fora do contrato público: o gateway não roteia /interno; só invocadores com
	// papel run.invoker (Cloud Scheduler) chegam aqui (ADR-0012).
	mux.HandleFunc("POST /interno/reconciliar-publicacoes", m.reconciliar)
	return registrarRequisicoes(o.Registrador, o.ProjetoId, mux)
}

func (m *manipulador) saude(w http.ResponseWriter, _ *http.Request) {
	escreverJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (m *manipulador) reconciliar(w http.ResponseWriter, r *http.Request) {
	relatorio, err := m.s.ReconciliarPublicacoes(r.Context())
	if err != nil {
		escreverErro(w, r, err)
		return
	}
	escreverJSON(w, http.StatusOK, relatorio)
}

func (m *manipulador) iniciar(w http.ResponseWriter, r *http.Request) {
	uid, err := m.uidDe(r)
	if err != nil {
		escreverNaoAutenticado(w, r, err.Error())
		return
	}
	chave := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if !padraoUuid.MatchString(chave) {
		escreverErro(w, r, dominio.NovoErro(dominio.CodigoPayloadInvalido, "o cabeçalho Idempotency-Key é obrigatório e deve ser um UUID"))
		return
	}
	var req requisicaoIniciar
	if err := lerJSON(r, &req); err != nil {
		escreverErro(w, r, err)
		return
	}
	if len(bytes.TrimSpace(req.Qr)) == 0 {
		escreverErro(w, r, dominio.NovoErro(dominio.CodigoPayloadInvalido, "o campo qr é obrigatório"))
		return
	}
	entrada := aplicacao.EntradaIniciarValidacao{Uid: uid, ChaveIdempotencia: chave, PayloadQr: req.Qr, LidoEm: req.LidoEm}
	if req.Dispositivo != nil {
		entrada.Dispositivo = &dominio.Dispositivo{Plataforma: req.Dispositivo.Plataforma, VersaoApp: req.Dispositivo.VersaoApp}
	}
	resultado, err := m.s.IniciarValidacao(r.Context(), entrada)
	if err != nil {
		escreverErro(w, r, err)
		return
	}
	anotar(r, resultado.Validacao.Id, resultado.Validacao.IdMaquina, uid)
	escreverJSON(w, http.StatusCreated, validacaoParaDto(resultado))
}

func (m *manipulador) consultar(w http.ResponseWriter, r *http.Request) {
	uid, err := m.uidDe(r)
	if err != nil {
		escreverNaoAutenticado(w, r, err.Error())
		return
	}
	id := r.PathValue("idValidacao")
	anotar(r, id, "", uid)
	resultado, err := m.s.ConsultarValidacao(r.Context(), uid, id)
	if err != nil {
		escreverErro(w, r, err)
		return
	}
	anotar(r, id, resultado.Validacao.IdMaquina, uid)
	escreverJSON(w, http.StatusOK, validacaoParaDto(resultado))
}

func (m *manipulador) concluir(w http.ResponseWriter, r *http.Request) {
	uid, err := m.uidDe(r)
	if err != nil {
		escreverNaoAutenticado(w, r, err.Error())
		return
	}
	id := r.PathValue("idValidacao")
	anotar(r, id, "", uid)
	var req requisicaoConcluir
	if err := lerJSON(r, &req); err != nil {
		escreverErro(w, r, err)
		return
	}
	if req.Leituras == nil || req.CodigoRotacionado == nil {
		escreverErro(w, r, dominio.NovoErro(dominio.CodigoPayloadInvalido, "os campos leituras e codigoRotacionado são obrigatórios"))
		return
	}
	resultado, err := m.s.ConcluirValidacao(r.Context(), aplicacao.EntradaConcluirValidacao{
		Uid:               uid,
		IdValidacao:       id,
		Leituras:          dominio.Leituras{Medidor: req.Leituras.Medidor, UnidadesVendidas: req.Leituras.UnidadesVendidas},
		CodigoRotacionado: *req.CodigoRotacionado,
		Observacoes:       req.Observacoes,
	})
	if err != nil {
		escreverErro(w, r, err)
		return
	}
	anotar(r, id, resultado.Validacao.IdMaquina, uid)
	escreverJSON(w, http.StatusOK, conclusaoParaDto(resultado))
}

// lerJSON decodifica o corpo com limite de tamanho e esquema fechado.
func lerJSON(r *http.Request, destino any) error {
	corpo, err := io.ReadAll(http.MaxBytesReader(nil, r.Body, TamanhoMaximoCorpo))
	if err != nil {
		var grande *http.MaxBytesError
		if errors.As(err, &grande) {
			return dominio.NovoErro(dominio.CodigoPayloadInvalido, "corpo da requisição excede o tamanho máximo")
		}
		return dominio.NovoErro(dominio.CodigoPayloadInvalido, "não foi possível ler o corpo da requisição").ComCausa(err)
	}
	dec := json.NewDecoder(bytes.NewReader(corpo))
	dec.DisallowUnknownFields()
	if err := dec.Decode(destino); err != nil {
		return dominio.NovoErro(dominio.CodigoPayloadInvalido, "corpo da requisição não segue o contrato").ComCausa(err)
	}
	if dec.More() {
		return dominio.NovoErro(dominio.CodigoPayloadInvalido, "corpo da requisição tem dados além do objeto JSON")
	}
	return nil
}
