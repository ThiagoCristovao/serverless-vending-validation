// Package armazenamento guarda os eventos recebidos pelo sistema central em um
// arquivo JSON Lines, com índice em memória por (idValidacao, tipo). É o
// mecanismo de idempotência do consumidor (RF-24, CE-17): um evento já
// registrado é reconhecido como duplicata mesmo após reinício do processo.
package armazenamento

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/ThiagoCristovao/serverless-vending-validation/sistema-central-simulado/internal/eventos"
)

// Item é um evento registrado com os metadados da entrega.
type Item struct {
	IdValidacao string            `json:"idValidacao"`
	Tipo        string            `json:"tipo"`
	IdMaquina   string            `json:"idMaquina"`
	IdMensagem  string            `json:"idMensagem"`
	PublicadaEm time.Time         `json:"publicadaEm"`
	RecebidaEm  time.Time         `json:"recebidaEm"`
	Tentativa   int               `json:"tentativa"`
	Atributos   map[string]string `json:"atributos,omitempty"`
	Evento      eventos.Evento    `json:"evento"`
	Corpo       json.RawMessage   `json:"corpo"`
}

// Chave devolve a chave de idempotência do item.
func (i Item) Chave() eventos.Chave { return eventos.Chave{IdValidacao: i.IdValidacao, Tipo: i.Tipo} }

// Filtro restringe a listagem; campos vazios não filtram.
type Filtro struct {
	IdValidacao string
	IdMaquina   string
	Tipo        string
}

// Registro é o armazenamento: arquivo JSONL (opcional) + índice em memória.
type Registro struct {
	mu      sync.Mutex
	caminho string
	arquivo *os.File
	indice  map[eventos.Chave]int
	itens   []Item
}

// Abrir carrega o arquivo (criando-o se não existir) e reconstrói o índice.
// Com caminho vazio, o registro fica apenas em memória.
func Abrir(caminho string) (*Registro, error) {
	r := &Registro{caminho: caminho, indice: map[eventos.Chave]int{}}
	if caminho == "" {
		return r, nil
	}
	if err := os.MkdirAll(filepath.Dir(caminho), 0o755); err != nil {
		return nil, err
	}
	arquivo, err := os.OpenFile(caminho, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0o644)
	if err != nil {
		return nil, err
	}
	leitor := bufio.NewScanner(arquivo)
	leitor.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	linha := 0
	for leitor.Scan() {
		linha++
		if len(leitor.Bytes()) == 0 {
			continue
		}
		var item Item
		if err := json.Unmarshal(leitor.Bytes(), &item); err != nil {
			_ = arquivo.Close()
			return nil, fmt.Errorf("%s, linha %d: %w", caminho, linha, err)
		}
		r.adicionar(item)
	}
	if err := leitor.Err(); err != nil {
		_ = arquivo.Close()
		return nil, err
	}
	r.arquivo = arquivo
	return r, nil
}

func (r *Registro) adicionar(item Item) {
	r.indice[item.Chave()] = len(r.itens)
	r.itens = append(r.itens, item)
}

// Registrar grava o item se a chave (idValidacao, tipo) ainda não existir.
// Devolve false (sem erro) para duplicatas. A gravação é sincronizada em disco
// antes de responder, para que o ack só aconteça com o evento persistido.
func (r *Registro) Registrar(item Item) (novo bool, err error) {
	if item.IdValidacao == "" || item.Tipo == "" {
		return false, errors.New("item sem idValidacao ou tipo")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, existe := r.indice[item.Chave()]; existe {
		return false, nil
	}
	if r.arquivo != nil {
		linha, err := json.Marshal(item)
		if err != nil {
			return false, err
		}
		if _, err := r.arquivo.Write(append(linha, '\n')); err != nil {
			return false, err
		}
		if err := r.arquivo.Sync(); err != nil {
			return false, err
		}
	}
	r.adicionar(item)
	return true, nil
}

// Obter devolve o item de uma chave, se registrado.
func (r *Registro) Obter(chave eventos.Chave) (Item, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	i, ok := r.indice[chave]
	if !ok {
		return Item{}, false
	}
	return r.itens[i], true
}

// Listar devolve os itens que satisfazem o filtro, na ordem de registro.
func (r *Registro) Listar(f Filtro) []Item {
	r.mu.Lock()
	defer r.mu.Unlock()
	var saida []Item
	for _, item := range r.itens {
		if f.IdValidacao != "" && item.IdValidacao != f.IdValidacao {
			continue
		}
		if f.IdMaquina != "" && item.IdMaquina != f.IdMaquina {
			continue
		}
		if f.Tipo != "" && item.Tipo != f.Tipo {
			continue
		}
		saida = append(saida, item)
	}
	return saida
}

// Total é o número de itens registrados.
func (r *Registro) Total() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.itens)
}

// Caminho do arquivo em uso (vazio em memória).
func (r *Registro) Caminho() string { return r.caminho }

// Fechar libera o arquivo.
func (r *Registro) Fechar() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.arquivo == nil {
		return nil
	}
	err := r.arquivo.Close()
	r.arquivo = nil
	return err
}
