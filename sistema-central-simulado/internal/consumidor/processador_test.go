package consumidor_test

import (
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/ThiagoCristovao/serverless-vending-validation/sistema-central-simulado/internal/armazenamento"
	"github.com/ThiagoCristovao/serverless-vending-validation/sistema-central-simulado/internal/consumidor"
	"github.com/ThiagoCristovao/serverless-vending-validation/sistema-central-simulado/internal/eventos"
	"github.com/ThiagoCristovao/serverless-vending-validation/sistema-central-simulado/internal/falhas"
)

var silencioso = slog.New(slog.NewTextHandler(io.Discard, nil))

func corpo(tipo, id, maq string) []byte {
	validacao := `"iniciadaEm": "2026-10-26T12:00:00Z", "expiraEm": "2026-10-26T12:15:00Z"`
	if tipo == eventos.TipoValidacaoConcluida {
		validacao = `"iniciadaEm": "2026-10-26T12:00:00Z", "concluidaEm": "2026-10-26T12:09:00Z", "leituras": {"medidor": 100, "unidadesVendidas": 3}, "codigoRotacionado": true`
	}
	return []byte(`{"tipo":"` + tipo + `","versao":1,"idValidacao":"` + id + `","ocorridoEm":"2026-10-26T12:00:00Z",` +
		`"maquina":{"id":"` + maq + `","modelo":"CN168","localizacao":{"id":"L1","nome":"Local 1"}},` +
		`"operador":{"id":"op-1","nome":"Operador"},"validacao":{` + validacao + `}}`)
}

func mensagem(tipo, id, maq string) consumidor.Mensagem {
	return consumidor.Mensagem{Id: "m-" + id + "-" + tipo, Dados: corpo(tipo, id, maq), Atributos: map[string]string{"tipo": tipo, "idMaquina": maq}, ChaveOrdenacao: maq, PublicadaEm: time.Now(), Tentativa: 1}
}

const (
	idA = "01J8ZK3V9Q7XW2N4M6P8R0T2Y4"
	idB = "01J8ZK3V9Q7XW2N4M6P8R0T2Y5"
)

func TestTratarRegistraEDetectaDuplicata(t *testing.T) {
	reg, _ := armazenamento.Abrir("")
	p := consumidor.NovoProcessador(reg, nil, silencioso)

	d := p.Tratar(mensagem(eventos.TipoValidacaoIniciada, idA, "VM-1"))
	if !d.Confirmar || d.Resultado != consumidor.ResultadoRegistrada || d.Evento == nil || d.Evento.IdValidacao != idA {
		t.Fatalf("primeira: %+v", d)
	}
	d = p.Tratar(mensagem(eventos.TipoValidacaoIniciada, idA, "VM-1"))
	if !d.Confirmar || d.Resultado != consumidor.ResultadoDuplicada {
		t.Fatalf("duplicata: %+v", d)
	}
	d = p.Tratar(mensagem(eventos.TipoValidacaoConcluida, idA, "VM-1"))
	if !d.Confirmar || d.Resultado != consumidor.ResultadoRegistrada {
		t.Fatalf("concluída: %+v", d)
	}
	est := p.Estatisticas()
	if est.Recebidas != 3 || est.Registradas != 2 || est.Duplicadas != 1 || est.UltimaRecebidaEm == nil {
		t.Fatalf("estatísticas: %+v", est)
	}
	itens := reg.Listar(armazenamento.Filtro{IdValidacao: idA})
	if len(itens) != 2 || itens[0].Tipo != eventos.TipoValidacaoIniciada || itens[1].Tipo != eventos.TipoValidacaoConcluida || itens[1].Atributos["tipo"] != eventos.TipoValidacaoConcluida {
		t.Fatalf("itens: %+v", itens)
	}
}

func TestTratarRejeitaCorpoForaDoContrato(t *testing.T) {
	reg, _ := armazenamento.Abrir("")
	p := consumidor.NovoProcessador(reg, nil, silencioso)
	d := p.Tratar(consumidor.Mensagem{Id: "m-x", Dados: []byte(`{"tipo":"outro"}`)})
	if d.Confirmar || d.Resultado != consumidor.ResultadoInvalida || !errors.Is(d.Erro, eventos.ErrInvalido) {
		t.Fatalf("inválida: %+v", d)
	}
	if reg.Total() != 0 || p.Estatisticas().Invalidas != 1 {
		t.Fatal("inválida não deveria ser persistida")
	}
}

func TestTratarComInterruptor(t *testing.T) {
	reg, _ := armazenamento.Abrir("")
	inter := falhas.Novo()
	p := consumidor.NovoProcessador(reg, inter, silencioso)

	_ = inter.Definir(falhas.Configuracao{Modo: falhas.ModoRejeitar, IdMaquina: "VM-2"})
	if d := p.Tratar(mensagem(eventos.TipoValidacaoIniciada, idB, "VM-2")); d.Confirmar || d.Resultado != consumidor.ResultadoRejeitada {
		t.Fatalf("rejeitar: %+v", d)
	}
	if d := p.Tratar(mensagem(eventos.TipoValidacaoIniciada, idA, "VM-1")); !d.Confirmar || d.Resultado != consumidor.ResultadoRegistrada {
		t.Fatalf("outra máquina segue: %+v", d)
	}
	if reg.Total() != 1 {
		t.Fatalf("rejeitada não deveria ser persistida: %d", reg.Total())
	}

	_ = inter.Definir(falhas.Configuracao{Modo: falhas.ModoConfirmacaoPerdida})
	if d := p.Tratar(mensagem(eventos.TipoValidacaoIniciada, idB, "VM-2")); d.Confirmar || d.Resultado != consumidor.ResultadoConfirmacaoPerdida {
		t.Fatalf("confirmação perdida: %+v", d)
	}
	// A reentrega é reconhecida como duplicata e confirmada, mesmo com o modo ligado.
	if d := p.Tratar(mensagem(eventos.TipoValidacaoIniciada, idB, "VM-2")); !d.Confirmar || d.Resultado != consumidor.ResultadoDuplicada {
		t.Fatalf("reentrega: %+v", d)
	}
	est := p.Estatisticas()
	if est.Rejeitadas != 1 || est.ConfirmacoesPerdidas != 1 || est.Duplicadas != 1 || est.Registradas != 2 || reg.Total() != 2 {
		t.Fatalf("estatísticas: %+v total=%d", est, reg.Total())
	}
}

type armazenadorFalho struct{}

func (armazenadorFalho) Registrar(armazenamento.Item) (bool, error) {
	return false, errors.New("disco cheio")
}

func TestTratarErroDeArmazenamentoNaoConfirma(t *testing.T) {
	p := consumidor.NovoProcessador(armazenadorFalho{}, nil, silencioso)
	d := p.Tratar(mensagem(eventos.TipoValidacaoIniciada, idA, "VM-1"))
	if d.Confirmar || d.Resultado != consumidor.ResultadoErroArmazenamento || d.Erro == nil {
		t.Fatalf("erro de armazenamento: %+v", d)
	}
	if p.Estatisticas().ErrosArmazenamento != 1 {
		t.Fatal("contador de erros")
	}
}
