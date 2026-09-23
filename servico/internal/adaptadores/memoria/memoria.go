// Package memoria implementa as portas em memória, para testes de unidade e
// execução local sem credenciais. Não é usado em produção.
package memoria

import (
	"context"
	"crypto/rand"
	"io"
	"sync"
	"time"

	"github.com/ThiagoCristovao/serverless-vending-validation/servico/internal/dominio"
	"github.com/ThiagoCristovao/serverless-vending-validation/servico/internal/portas"
)

// Armazenamento guarda máquinas, operadores, chaves e validações em mapas
// protegidos por um único mutex, o que torna Abrir e Concluir atômicos.
type Armazenamento struct {
	mu         sync.Mutex
	maquinas   map[string]*dominio.Maquina
	operadores map[string]*dominio.Operador
	chaves     map[string]*dominio.ChaveQr
	validacoes map[string]*dominio.Validacao
}

// Novo cria um armazenamento vazio.
func Novo() *Armazenamento {
	return &Armazenamento{
		maquinas:   map[string]*dominio.Maquina{},
		operadores: map[string]*dominio.Operador{},
		chaves:     map[string]*dominio.ChaveQr{},
		validacoes: map[string]*dominio.Validacao{},
	}
}

// SemearMaquina cadastra (ou substitui) uma máquina.
func (a *Armazenamento) SemearMaquina(m dominio.Maquina) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.maquinas[m.Id] = &m
}

// SemearOperador cadastra (ou substitui) um operador.
func (a *Armazenamento) SemearOperador(o dominio.Operador) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.operadores[o.Uid] = &o
}

// SemearChaveQr cadastra (ou substitui) uma chave de assinatura.
func (a *Armazenamento) SemearChaveQr(c dominio.ChaveQr) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.chaves[c.Kid] = &c
}

// ObterMaquina implementa portas.RepositorioMaquinas.
func (a *Armazenamento) ObterMaquina(_ context.Context, id string) (*dominio.Maquina, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	m, ok := a.maquinas[id]
	if !ok {
		return nil, portas.ErrNaoEncontrado
	}
	c := *m
	return &c, nil
}

// ObterOperador implementa portas.RepositorioOperadores.
func (a *Armazenamento) ObterOperador(_ context.Context, uid string) (*dominio.Operador, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	o, ok := a.operadores[uid]
	if !ok {
		return nil, portas.ErrNaoEncontrado
	}
	c := *o
	return &c, nil
}

// ObterChaveQr implementa portas.ProvedorChavesQr.
func (a *Armazenamento) ObterChaveQr(_ context.Context, kid string) (*dominio.ChaveQr, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	k, ok := a.chaves[kid]
	if !ok {
		return nil, portas.ErrNaoEncontrado
	}
	c := *k
	return &c, nil
}

// ObterValidacao implementa portas.RepositorioValidacoes.
func (a *Armazenamento) ObterValidacao(_ context.Context, id string) (*dominio.Validacao, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	v, ok := a.validacoes[id]
	if !ok {
		return nil, portas.ErrNaoEncontrado
	}
	c := *v
	return &c, nil
}

// ObterPorChaveIdempotencia implementa portas.RepositorioValidacoes.
func (a *Armazenamento) ObterPorChaveIdempotencia(_ context.Context, chave string) (*dominio.Validacao, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, v := range a.validacoes {
		if v.ChaveIdempotencia == chave {
			c := *v
			return &c, nil
		}
	}
	return nil, portas.ErrNaoEncontrado
}

// Abrir implementa portas.RepositorioValidacoes de forma atômica.
func (a *Armazenamento) Abrir(_ context.Context, v *dominio.Validacao, agora time.Time) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	m, ok := a.maquinas[v.IdMaquina]
	if !ok {
		return portas.ErrNaoEncontrado
	}
	if m.ValidacaoAbertaId != "" {
		if aberta, existe := a.validacoes[m.ValidacaoAbertaId]; existe && aberta.Aberta() {
			if !aberta.Expirou(agora) {
				return &portas.ErrValidacaoEmAndamento{IdValidacao: aberta.Id, IdOperador: aberta.IdOperador}
			}
			aberta.MarcarExpirada()
		}
	}
	c := *v
	a.validacoes[v.Id] = &c
	m.ValidacaoAbertaId = v.Id
	m.AtualizadaEm = agora
	return nil
}

// Concluir implementa portas.RepositorioValidacoes de forma atômica.
func (a *Armazenamento) Concluir(_ context.Context, v *dominio.Validacao, agora time.Time) (*dominio.Maquina, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	m, ok := a.maquinas[v.IdMaquina]
	if !ok {
		return nil, portas.ErrNaoEncontrado
	}
	c := *v
	a.validacoes[v.Id] = &c
	m.RegistrarConclusao(&c, agora)
	mc := *m
	return &mc, nil
}

// MarcarExpirada implementa portas.RepositorioValidacoes.
func (a *Armazenamento) MarcarExpirada(_ context.Context, id string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	v, ok := a.validacoes[id]
	if !ok {
		return portas.ErrNaoEncontrado
	}
	v.MarcarExpirada()
	if m, existe := a.maquinas[v.IdMaquina]; existe && m.ValidacaoAbertaId == id {
		m.ValidacaoAbertaId = ""
	}
	return nil
}

// MarcarEventoPublicado implementa portas.RepositorioValidacoes.
func (a *Armazenamento) MarcarEventoPublicado(_ context.Context, id string, tipo dominio.TipoEvento, quando time.Time) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	v, ok := a.validacoes[id]
	if !ok {
		return portas.ErrNaoEncontrado
	}
	q := quando
	switch tipo {
	case dominio.EventoValidacaoIniciada:
		v.Eventos.IniciadaEm = &q
	case dominio.EventoValidacaoConcluida:
		v.Eventos.ConcluidaEm = &q
	}
	return nil
}

// Publicador registra os eventos publicados; Falha, quando definida, simula
// indisponibilidade do Pub/Sub.
type Publicador struct {
	mu      sync.Mutex
	Eventos []dominio.Evento
	Falha   error
}

// Publicar implementa portas.PublicadorEventos.
func (p *Publicador) Publicar(_ context.Context, evento dominio.Evento) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.Falha != nil {
		return p.Falha
	}
	p.Eventos = append(p.Eventos, evento)
	return nil
}

// Publicados devolve uma cópia dos eventos registrados.
func (p *Publicador) Publicados() []dominio.Evento {
	p.mu.Lock()
	defer p.mu.Unlock()
	c := make([]dominio.Evento, len(p.Eventos))
	copy(c, p.Eventos)
	return c
}

// Relogio é um relógio controlável pelos testes.
type Relogio struct {
	mu    sync.Mutex
	agora time.Time
}

// NovoRelogio cria um relógio parado no instante dado.
func NovoRelogio(instante time.Time) *Relogio { return &Relogio{agora: instante} }

// Agora implementa portas.Relogio.
func (r *Relogio) Agora() time.Time {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.agora
}

// Avancar move o relógio para a frente.
func (r *Relogio) Avancar(d time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.agora = r.agora.Add(d)
}

// GeradorUlid gera ULIDs com o instante do relógio e entropia criptográfica.
type GeradorUlid struct {
	Relogio   portas.Relogio
	Aleatorio io.Reader
}

// NovoId implementa portas.GeradorIdentificador.
func (g GeradorUlid) NovoId() (string, error) {
	fonte := g.Aleatorio
	if fonte == nil {
		fonte = rand.Reader
	}
	return dominio.NovoUlid(g.Relogio.Agora(), fonte)
}
