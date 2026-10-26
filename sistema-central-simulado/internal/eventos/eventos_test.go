package eventos_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/ThiagoCristovao/serverless-vending-validation/sistema-central-simulado/internal/eventos"
)

const concluida = `{
  "tipo": "validacao.concluida",
  "versao": 1,
  "idValidacao": "01J8ZK3V9Q7XW2N4M6P8R0T2Y4",
  "ocorridoEm": "2026-09-17T13:19:44Z",
  "maquina": { "id": "VM-2047", "modelo": "CN168", "localizacao": { "id": "BLA-T", "nome": "Bloco A - Térreo" } },
  "operador": { "id": "uid-do-firebase", "nome": "Operador Exemplo" },
  "validacao": {
    "iniciadaEm": "2026-09-17T13:05:12Z",
    "concluidaEm": "2026-09-17T13:19:44Z",
    "leituras": { "medidor": 14832, "unidadesVendidas": 137 },
    "codigoRotacionado": true,
    "avisos": ["medidor_menor_que_anterior"],
    "campoFuturo": "tolerado"
  }
}`

const iniciada = `{
  "tipo": "validacao.iniciada",
  "versao": 1,
  "idValidacao": "01J8ZK3V9Q7XW2N4M6P8R0T2Y4",
  "ocorridoEm": "2026-09-17T13:05:12Z",
  "maquina": { "id": "VM-2047", "modelo": "CN168", "localizacao": { "id": "BLA-T", "nome": "Bloco A - Térreo" } },
  "operador": { "id": "uid-do-firebase", "nome": "Operador Exemplo" },
  "validacao": { "iniciadaEm": "2026-09-17T13:05:12Z", "expiraEm": "2026-09-17T13:20:12Z" }
}`

func TestDecodificarConcluida(t *testing.T) {
	e, err := eventos.Decodificar([]byte(concluida))
	if err != nil {
		t.Fatal(err)
	}
	if e.Chave() != (eventos.Chave{IdValidacao: "01J8ZK3V9Q7XW2N4M6P8R0T2Y4", Tipo: eventos.TipoValidacaoConcluida}) {
		t.Fatalf("chave: %+v", e.Chave())
	}
	if e.Maquina.Localizacao.Nome != "Bloco A - Térreo" || e.Operador.Id != "uid-do-firebase" {
		t.Fatalf("campos: %+v", e)
	}
	if e.Validacao.Leituras == nil || e.Validacao.Leituras.Medidor != 14832 || e.Validacao.CodigoRotacionado == nil || !*e.Validacao.CodigoRotacionado {
		t.Fatalf("validacao: %+v", e.Validacao)
	}
	if len(e.Validacao.Avisos) != 1 || e.Validacao.Avisos[0] != "medidor_menor_que_anterior" {
		t.Fatalf("avisos: %v", e.Validacao.Avisos)
	}
}

func TestDecodificarIniciada(t *testing.T) {
	e, err := eventos.Decodificar([]byte(iniciada))
	if err != nil {
		t.Fatal(err)
	}
	if e.Validacao.ExpiraEm == nil || e.Validacao.Leituras != nil || e.Validacao.ConcluidaEm != nil {
		t.Fatalf("validacao: %+v", e.Validacao)
	}
}

func TestDecodificarRejeitaContratoViolado(t *testing.T) {
	casos := map[string]string{
		"JSON mal formado":        `{"tipo":`,
		"tipo desconhecido":       strings.Replace(iniciada, "validacao.iniciada", "validacao.cancelada", 1),
		"versão não suportada":    strings.Replace(iniciada, `"versao": 1`, `"versao": 2`, 1),
		"idValidacao inválido":    strings.Replace(iniciada, "01J8ZK3V9Q7XW2N4M6P8R0T2Y4", "abc", 1),
		"máquina ausente":         strings.Replace(iniciada, `"id": "VM-2047"`, `"id": ""`, 1),
		"operador ausente":        strings.Replace(iniciada, `"id": "uid-do-firebase"`, `"id": ""`, 1),
		"concluída sem leituras":  strings.Replace(concluida, `"leituras": { "medidor": 14832, "unidadesVendidas": 137 },`, ``, 1),
		"concluída sem concluída": strings.Replace(concluida, `"concluidaEm": "2026-09-17T13:19:44Z",`, ``, 1),
	}
	for nome, corpo := range casos {
		t.Run(nome, func(t *testing.T) {
			if _, err := eventos.Decodificar([]byte(corpo)); !errors.Is(err, eventos.ErrInvalido) {
				t.Fatalf("esperava ErrInvalido, veio %v", err)
			}
		})
	}
}
