package laboratorio

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// Relatorio acumula parâmetros, linha do tempo, tabelas e conclusão de um
// experimento e os emite em Markdown (docs/avaliacao/evidencias).
type Relatorio struct {
	titulo     string
	inicio     time.Time
	parametros [][2]string
	linhaTempo []string
	secoes     []string
	conclusoes []string
	ok         bool
}

// NovoRelatorio inicia um relatório.
func NovoRelatorio(titulo string) *Relatorio {
	return &Relatorio{titulo: titulo, inicio: time.Now(), ok: true}
}

// Parametro registra um parâmetro do experimento.
func (r *Relatorio) Parametro(nome string, valor any) {
	r.parametros = append(r.parametros, [2]string{nome, fmt.Sprint(valor)})
}

// Evento acrescenta uma linha à linha do tempo (e ecoa no stderr).
func (r *Relatorio) Evento(formato string, args ...any) {
	linha := fmt.Sprintf("%s  %s", time.Now().Format("15:04:05"), fmt.Sprintf(formato, args...))
	r.linhaTempo = append(r.linhaTempo, linha)
	fmt.Fprintln(os.Stderr, linha)
}

// Secao acrescenta uma seção em Markdown.
func (r *Relatorio) Secao(titulo, markdown string) {
	r.secoes = append(r.secoes, "### "+titulo+"\n\n"+strings.TrimRight(markdown, "\n")+"\n")
}

// Tabela acrescenta uma seção com uma tabela.
func (r *Relatorio) Tabela(titulo string, cabecalho []string, linhas [][]string) {
	var b strings.Builder
	b.WriteString("| " + strings.Join(cabecalho, " | ") + " |\n|")
	for range cabecalho {
		b.WriteString("---|")
	}
	b.WriteString("\n")
	for _, l := range linhas {
		b.WriteString("| " + strings.Join(l, " | ") + " |\n")
	}
	if len(linhas) == 0 {
		b.WriteString("| (sem dados) |\n")
	}
	r.Secao(titulo, b.String())
}

// Verificar registra um critério e seu resultado.
func (r *Relatorio) Verificar(ok bool, formato string, args ...any) {
	marca := "✅"
	if !ok {
		marca = "❌"
		r.ok = false
	}
	r.conclusoes = append(r.conclusoes, marca+" "+fmt.Sprintf(formato, args...))
	r.Evento("%s %s", marca, fmt.Sprintf(formato, args...))
}

// Ok informa se todos os critérios passaram.
func (r *Relatorio) Ok() bool { return r.ok }

// Markdown monta o documento.
func (r *Relatorio) Markdown() string {
	var b strings.Builder
	fmt.Fprintf(&b, "## %s\n\n", r.titulo)
	fmt.Fprintf(&b, "Executado em %s (duração %s). Resultado: ", r.inicio.Format("2006-01-02 15:04:05 MST"), time.Since(r.inicio).Round(time.Second))
	if r.ok {
		b.WriteString("**todos os critérios atendidos**.\n\n")
	} else {
		b.WriteString("**critérios não atendidos** (ver conclusão).\n\n")
	}
	b.WriteString("### Parâmetros\n\n| Parâmetro | Valor |\n|---|---|\n")
	for _, p := range r.parametros {
		fmt.Fprintf(&b, "| %s | %s |\n", p[0], p[1])
	}
	b.WriteString("\n### Linha do tempo\n\n```text\n")
	for _, l := range r.linhaTempo {
		b.WriteString(l + "\n")
	}
	b.WriteString("```\n\n")
	for _, s := range r.secoes {
		b.WriteString(s + "\n")
	}
	b.WriteString("### Conclusão\n\n")
	for _, c := range r.conclusoes {
		b.WriteString("- " + c + "\n")
	}
	return b.String()
}

func Dur(d time.Duration) string { return d.Round(time.Millisecond).String() }

func HoraDe(t time.Time) string {
	if t.IsZero() {
		return "—"
	}
	return t.Local().Format("15:04:05")
}
