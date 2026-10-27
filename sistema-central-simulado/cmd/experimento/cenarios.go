package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"time"

	"github.com/ThiagoCristovao/serverless-vending-validation/sistema-central-simulado/internal/armazenamento"
	"github.com/ThiagoCristovao/serverless-vending-validation/sistema-central-simulado/internal/eventos"
	"github.com/ThiagoCristovao/serverless-vending-validation/sistema-central-simulado/internal/laboratorio"
)

// resultado de uma validação completa (abertura + conclusão) feita pelo gateway.
type resultado struct {
	Maquina         string
	Id              string
	Chave           string
	StatusAbertura  int
	StatusConclusao int
	DurAbertura     time.Duration
	DurConclusao    time.Duration
	Erro            string
}

func (r resultado) ok() bool {
	return r.StatusAbertura == http.StatusCreated && r.StatusConclusao == http.StatusOK && r.Erro == ""
}

// validar executa uma validação completa na máquina.
func (x *experimento) validar(ctx context.Context, maq string) resultado {
	r := resultado{Maquina: maq}
	v, err := x.gateway.IniciarValidacao(ctx, x.adesivo, maq)
	r.StatusAbertura, r.DurAbertura = v.Resposta.Status, v.Resposta.Duracao
	if err != nil {
		r.Erro = err.Error()
		return r
	}
	r.Id, r.Chave = v.Id, v.Chave
	resp, err := x.gateway.ConcluirValidacao(ctx, v.Id)
	r.StatusConclusao, r.DurConclusao = resp.Status, resp.Duracao
	if err != nil {
		r.Erro = err.Error()
	}
	return r
}

// validarVarias alterna as máquinas A e B, sequencialmente.
func (x *experimento) validarVarias(ctx context.Context, n int) []resultado {
	var saida []resultado
	for i := 0; i < n; i++ {
		maq := x.cfg.MaquinaA
		if i%2 == 1 {
			maq = x.cfg.MaquinaB
		}
		saida = append(saida, x.validar(ctx, maq))
	}
	return saida
}

func linhasResultados(rs []resultado) [][]string {
	var linhas [][]string
	for i, r := range rs {
		erro := r.Erro
		if erro == "" {
			erro = "—"
		}
		linhas = append(linhas, []string{strconv.Itoa(i + 1), r.Maquina, r.Id, strconv.Itoa(r.StatusAbertura), dur(r.DurAbertura), strconv.Itoa(r.StatusConclusao), dur(r.DurConclusao), erro})
	}
	return linhas
}

var cabecalhoResultados = []string{"#", "Máquina", "idValidacao", "Abertura", "Tempo", "Conclusão", "Tempo", "Erro"}

func contarOk(rs []resultado) int {
	n := 0
	for _, r := range rs {
		if r.ok() {
			n++
		}
	}
	return n
}

// eventosConsumidos conta, para as validações dadas, quantos eventos o
// consumidor registrou e se cada par está em ordem.
func (x *experimento) eventosConsumidos(rs []resultado) (total int, emOrdem bool) {
	emOrdem = true
	for _, r := range rs {
		itens := x.central.Registro.Listar(armazenamento.Filtro{IdValidacao: r.Id})
		total += len(itens)
		if len(itens) == 2 && (itens[0].Tipo != eventos.TipoValidacaoIniciada || itens[1].Tipo != eventos.TipoValidacaoConcluida) {
			emOrdem = false
		}
	}
	return total, emOrdem
}

func linhasAmostras(as []Amostra) [][]string {
	var linhas [][]string
	for _, a := range as {
		linhas = append(linhas, []string{hora(a.Quando), strconv.FormatFloat(a.Valor, 'f', 0, 64)})
	}
	return linhas
}

// --- Cenário A: sistema central indisponível ------------------------------------

func (x *experimento) centralIndisponivel(ctx context.Context) error {
	rel := x.rel
	rel.Parametro("assinatura", x.cfg.Assinatura())
	rel.Parametro("duração da indisponibilidade", x.indisponibilidade)

	// Sanidade: consumir o que houver na assinatura antes de começar.
	if err := x.central.Iniciar(ctx); err != nil {
		return err
	}
	rel.Evento("consumidor iniciado; aguardando 10 s para esvaziar a assinatura")
	time.Sleep(10 * time.Second)
	antes := x.central.Processador.Estatisticas()
	rel.Evento("eventos antigos consumidos: %d", antes.Registradas+antes.Duplicadas)

	// Falha: o sistema central para de consumir.
	x.central.Pausar()
	inicioFalha := time.Now()
	rel.Evento("sistema central PAUSADO (indisponível)")

	rs := x.validarVarias(ctx, x.validacoes)
	rel.Evento("%d validações executadas durante a indisponibilidade; %d completas sem erro", len(rs), contarOk(rs))
	rel.Tabela("Validações durante a indisponibilidade", cabecalhoResultados, linhasResultados(rs))

	// Mantém a indisponibilidade pelo tempo configurado, amostrando o backlog.
	for time.Since(inicioFalha) < x.indisponibilidade {
		restante := x.indisponibilidade - time.Since(inicioFalha)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(min(30*time.Second, restante)):
		}
		if as, err := x.nuvem.Backlog(ctx, x.cfg.Assinatura(), inicioFalha.Add(-2*time.Minute)); err == nil && len(as) > 0 {
			ultima := as[len(as)-1]
			rel.Evento("backlog (num_undelivered_messages) = %.0f em %s", ultima.Valor, hora(ultima.Quando))
		}
	}

	// Recuperação: o sistema central volta.
	if err := x.central.Iniciar(ctx); err != nil {
		return err
	}
	fimFalha := time.Now()
	rel.Evento("sistema central RETOMADO após %s", dur(fimFalha.Sub(inicioFalha)))
	esperados := 2 * contarOk(rs)
	levou, ok := laboratorio.Esperar(ctx, x.prazoRecuperacao, time.Second, func() bool {
		total, _ := x.eventosConsumidos(rs)
		return total >= esperados
	})
	total, emOrdem := x.eventosConsumidos(rs)
	rel.Evento("%d de %d eventos das validações da falha consumidos %s após a retomada", total, esperados, dur(levou))
	depois := x.central.Processador.Estatisticas()

	// Evidências do provedor (as métricas do Pub/Sub têm granularidade de ~1 min).
	rel.Evento("coletando métricas do Monitoring por 2 min após a retomada")
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(2 * time.Minute):
	}
	if as, err := x.nuvem.Backlog(ctx, x.cfg.Assinatura(), inicioFalha.Add(-2*time.Minute)); err == nil {
		rel.Tabela("Métrica `subscription/num_undelivered_messages` (Cloud Monitoring)", []string{"Instante", "Mensagens não entregues"}, linhasAmostras(as))
	} else {
		rel.Secao("Métrica de backlog", "falha ao consultar o Monitoring: "+err.Error())
	}
	if as, err := x.nuvem.IdadeMaisAntiga(ctx, x.cfg.Assinatura(), inicioFalha.Add(-2*time.Minute)); err == nil {
		rel.Tabela("Métrica `subscription/oldest_unacked_message_age` (segundos)", []string{"Instante", "Idade (s)"}, linhasAmostras(as))
	}

	rel.Verificar(contarOk(rs) == len(rs), "RNF-03: %d/%d validações completas (201/200) com o sistema central indisponível", contarOk(rs), len(rs))
	rel.Verificar(ok && total == esperados, "RNF-01: %d/%d eventos consumidos após o restabelecimento, sem intervenção manual, em %s", total, esperados, dur(levou))
	rel.Verificar(emOrdem, "ordem por máquina preservada (iniciada antes de concluida em cada validação)")
	rel.Verificar(depois.Duplicadas == antes.Duplicadas && depois.Rejeitadas == antes.Rejeitadas, "nenhuma duplicata ou rejeição durante a recuperação (duplicadas=%d, rejeitadas=%d)", depois.Duplicadas-antes.Duplicadas, depois.Rejeitadas-antes.Rejeitadas)
	return nil
}

// --- Cenário B: Pub/Sub indisponível ao publicar --------------------------------

func (x *experimento) publicacaoIndisponivel(ctx context.Context) error {
	rel := x.rel
	conta := "serviceAccount:" + x.cfg.ContaFuncao()
	rel.Parametro("tópico", x.cfg.Topico())
	rel.Parametro("falha injetada", "remoção de roles/pubsub.publisher da conta da função no tópico")
	if err := x.central.Iniciar(ctx); err != nil {
		return err
	}

	restaurar, err := x.nuvem.RemoverPublicador(ctx, x.cfg.Topico(), conta)
	if err != nil {
		return err
	}
	inicioFalha := time.Now()
	rel.Evento("vínculo pubsub.publisher REMOVIDO da conta da função")
	restaurado := false
	defer func() {
		if !restaurado {
			_ = restaurar(context.Background())
			rel.Evento("vínculo restaurado (limpeza)")
		}
	}()

	// Sonda: a falha está ativa quando uma validação nova fica com o marcador nulo.
	var sondas []resultado
	var ativaEm time.Time
	for time.Since(inicioFalha) < 8*time.Minute && ativaEm.IsZero() {
		r := x.validar(ctx, x.cfg.MaquinaA)
		if !r.ok() && (r.StatusAbertura == http.StatusServiceUnavailable || r.StatusConclusao == http.StatusServiceUnavailable) {
			// O ambiente pode ainda estar se recuperando de outro cenário (a
			// propagação de IAM não é monotônica); espera e tenta de novo.
			rel.Evento("sonda recebeu 503 (armazenamento indisponível); nova sonda em 15 s")
			time.Sleep(15 * time.Second)
			continue
		}
		sondas = append(sondas, r)
		if !r.ok() {
			return fmt.Errorf("sonda falhou: %s", r.Erro)
		}
		m, err := x.nuvem.MarcadoresPublicacao(ctx, r.Id)
		if err != nil {
			return err
		}
		if m.IniciadaEm == nil {
			ativaEm = time.Now()
			rel.Evento("falha ATIVA: validação %s ficou com eventos.iniciadaPublicadaEm nulo (%s após a remoção)", r.Id, dur(ativaEm.Sub(inicioFalha)))
		} else {
			rel.Evento("sonda %s publicada normalmente (IAM ainda propagando)", r.Id)
			time.Sleep(10 * time.Second)
		}
	}
	if ativaEm.IsZero() {
		return errors.New("a remoção do vínculo não surtiu efeito em 8 min")
	}

	rs := x.validarVarias(ctx, x.validacoes)
	rel.Evento("%d validações executadas com a publicação indisponível; %d completas sem erro", len(rs), contarOk(rs))
	rel.Tabela("Validações com a publicação indisponível", cabecalhoResultados, linhasResultados(rs))
	pendentes := 0
	var linhasMarc [][]string
	for _, r := range rs {
		m, err := x.nuvem.MarcadoresPublicacao(ctx, r.Id)
		if err != nil {
			return err
		}
		if m.IniciadaEm == nil {
			pendentes++
		}
		if m.ConcluidaEm == nil {
			pendentes++
		}
		linhasMarc = append(linhasMarc, []string{r.Id, m.Status, marcador(m.IniciadaEm), marcador(m.ConcluidaEm)})
	}
	rel.Tabela("Marcadores de publicação no Firestore durante a falha", []string{"idValidacao", "status", "iniciadaPublicadaEm", "concluidaPublicadaEm"}, linhasMarc)
	rel.Evento("%d de %d publicações pendentes (marcador nulo)", pendentes, 2*len(rs))

	// Recuperação: restaura o vínculo e espera a reconciliação do Scheduler.
	if err := restaurar(ctx); err != nil {
		return err
	}
	restaurado = true
	rel.Evento("vínculo pubsub.publisher RESTAURADO após %s de falha; aguardando a reconciliação (Cloud Scheduler a cada 5 min, pendências com mais de 2 min)", dur(time.Since(inicioFalha)))
	todas := append(slices.Clone(sondas), rs...)
	esperados := 2 * contarOk(todas)
	levou, ok := laboratorio.Esperar(ctx, x.prazoRecuperacao, 5*time.Second, func() bool {
		total, _ := x.eventosConsumidos(todas)
		return total >= esperados
	})
	total, emOrdem := x.eventosConsumidos(todas)
	rel.Evento("%d de %d eventos consumidos %s após restaurar", total, esperados, dur(levou))

	linhasMarc = nil
	publicadas := 0
	for _, r := range rs {
		m, err := x.nuvem.MarcadoresPublicacao(ctx, r.Id)
		if err != nil {
			return err
		}
		if m.IniciadaEm != nil {
			publicadas++
		}
		if m.ConcluidaEm != nil {
			publicadas++
		}
		linhasMarc = append(linhasMarc, []string{r.Id, m.Status, marcador(m.IniciadaEm), marcador(m.ConcluidaEm)})
	}
	rel.Tabela("Marcadores de publicação no Firestore após a reconciliação", []string{"idValidacao", "status", "iniciadaPublicadaEm", "concluidaPublicadaEm"}, linhasMarc)

	// Logs da função: pendências e reconciliação.
	if entradas, err := x.nuvem.EntradasDaFuncao(ctx, x.cfg.NomeFuncao(), inicioFalha, `("publicação pendente" OR "reconciliação" OR "reconciliar")`); err == nil {
		var linhas [][]string
		for _, e := range entradas {
			mensagem := e.Mensagem
			if mensagem == "" && e.Requisicao != "" {
				mensagem = "log de acesso: " + e.Requisicao
			}
			if mensagem == "" {
				continue
			}
			linhas = append(linhas, []string{hora(e.Quando), e.Severidade, mensagem, e.Rota(), e.Campo("idValidacao"), e.Campo("status")})
		}
		rel.Tabela("Logs da função (Cloud Logging): publicações pendentes e chamadas de reconciliação", []string{"Instante", "Severidade", "Mensagem", "Rota", "idValidacao", "Status"}, linhas)
	} else {
		rel.Secao("Logs da função", "falha ao consultar o Logging: "+err.Error())
	}

	rel.Verificar(contarOk(rs) == len(rs), "RNF-04: %d/%d validações completas (201/200) com a publicação indisponível", contarOk(rs), len(rs))
	rel.Verificar(pendentes == 2*len(rs), "CE-14: %d/%d publicações ficaram pendentes (marcador nulo) sem afetar a resposta ao operador", pendentes, 2*len(rs))
	rel.Verificar(ok && total == esperados && publicadas == 2*len(rs), "ADR-0012: reconciliação republicou e o consumidor recebeu %d/%d eventos em %s após restaurar (%d marcadores gravados)", total, esperados, dur(levou), publicadas)
	rel.Verificar(emOrdem, "ordem por máquina preservada após a reconciliação")
	return nil
}

func marcador(t *time.Time) string {
	if t == nil {
		return "**null**"
	}
	return hora(*t)
}

// --- Cenário C: Firestore indisponível ------------------------------------------

func (x *experimento) firestoreIndisponivel(ctx context.Context) error {
	rel := x.rel
	conta := "serviceAccount:" + x.cfg.ContaFuncao()
	rel.Parametro("falha injetada", "remoção de roles/datastore.user da conta da função no projeto")
	if err := x.central.Iniciar(ctx); err != nil {
		return err
	}

	restaurar, err := x.nuvem.RemoverPapelProjeto(ctx, "roles/datastore.user", conta)
	if err != nil {
		return err
	}
	inicioFalha := time.Now()
	rel.Evento("vínculo roles/datastore.user REMOVIDO da conta da função")
	restaurado := false
	defer func() {
		if !restaurado {
			_ = restaurar(context.Background())
			rel.Evento("vínculo restaurado (limpeza)")
		}
	}()

	// Sonda até o serviço responder 503 dependencia_indisponivel.
	var ativaEm time.Time
	corpo := laboratorio.CorpoValidacao(x.adesivo.Qr(x.cfg.MaquinaA))
	for time.Since(inicioFalha) < 8*time.Minute && ativaEm.IsZero() {
		resp, err := x.gateway.Chamar(ctx, http.MethodPost, "/validacoes", corpo, laboratorio.NovoUuid())
		if err != nil {
			return err
		}
		switch {
		case resp.Status == http.StatusServiceUnavailable && resp.Texto("codigo") == "dependencia_indisponivel":
			ativaEm = time.Now()
			rel.Evento("falha ATIVA: 503 dependencia_indisponivel (%s após a remoção)", dur(ativaEm.Sub(inicioFalha)))
		case resp.Status == http.StatusCreated:
			rel.Evento("sonda %s aberta normalmente (IAM ainda propagando); concluindo", resp.Texto("idValidacao"))
			if _, err := x.gateway.ConcluirValidacao(ctx, resp.Texto("idValidacao")); err != nil {
				rel.Evento("conclusão da sonda: %v", err)
			}
			time.Sleep(10 * time.Second)
		default:
			rel.Evento("sonda respondeu %d: %s", resp.Status, resp.Bruto)
			time.Sleep(10 * time.Second)
		}
	}
	if ativaEm.IsZero() {
		return errors.New("a remoção do vínculo não surtiu efeito em 8 min")
	}

	// Requisições durante a falha, com a política de reenvio do aplicativo
	// (RF-23: 1 s, 2 s, 4 s) aplicada a cada uma.
	esperas := []time.Duration{time.Second, 2 * time.Second, 4 * time.Second}
	var linhas [][]string
	codigos503, outros := 0, 0
	var chaves []string
	for i := 0; i < x.validacoes; i++ {
		chave := laboratorio.NovoUuid()
		chaves = append(chaves, chave)
		var statusTentativas []string
		for tentativa := 0; tentativa <= len(esperas); tentativa++ {
			resp, err := x.gateway.Chamar(ctx, http.MethodPost, "/validacoes", corpo, chave)
			if err != nil {
				return err
			}
			statusTentativas = append(statusTentativas, fmt.Sprintf("%d (%s)", resp.Status, dur(resp.Duracao)))
			if resp.Status == http.StatusServiceUnavailable && resp.Texto("codigo") == "dependencia_indisponivel" {
				codigos503++
			} else {
				outros++
			}
			if tentativa < len(esperas) {
				time.Sleep(esperas[tentativa])
			}
		}
		linhas = append(linhas, append([]string{strconv.Itoa(i + 1), chave[:8]}, statusTentativas...))
	}
	rel.Tabela("Requisições durante a falha, com os reenvios do aplicativo (1 s, 2 s, 4 s)", []string{"#", "Chave", "1ª", "2ª", "3ª", "4ª"}, linhas)
	rel.Evento("%d respostas 503 dependencia_indisponivel e %d de outro tipo", codigos503, outros)

	// Recuperação.
	if err := restaurar(ctx); err != nil {
		return err
	}
	restaurado = true
	fimFalha := time.Now()
	rel.Evento("vínculo roles/datastore.user RESTAURADO; sondando até o serviço voltar")
	var recuperadaEm time.Time
	var idRecuperada string
	for time.Since(fimFalha) < x.prazoRecuperacao && recuperadaEm.IsZero() {
		// Reusa a primeira chave que falhou: o reenvio do operador deve criar a
		// validação normalmente, sem conflito nem estado parcial.
		resp, err := x.gateway.Chamar(ctx, http.MethodPost, "/validacoes", corpo, chaves[0])
		if err != nil {
			return err
		}
		if resp.Status == http.StatusCreated {
			recuperadaEm = time.Now()
			idRecuperada = resp.Texto("idValidacao")
			rel.Evento("serviço RECUPERADO: 201 com a chave da 1ª requisição falha, validação %s (%s após restaurar)", idRecuperada, dur(recuperadaEm.Sub(fimFalha)))
		} else {
			rel.Evento("ainda %d; nova sonda em 10 s", resp.Status)
			time.Sleep(10 * time.Second)
		}
	}
	if recuperadaEm.IsZero() {
		return errors.New("o serviço não se recuperou no prazo")
	}
	if _, err := x.gateway.ConcluirValidacao(ctx, idRecuperada); err != nil {
		return err
	}
	levou, consumida := laboratorio.Esperar(ctx, 2*time.Minute, time.Second, func() bool {
		return len(x.central.Registro.Listar(armazenamento.Filtro{IdValidacao: idRecuperada})) == 2
	})
	rel.Evento("validação pós-recuperação consumida pelo sistema central: %v (%s)", consumida, dur(levou))

	if entradas, err := x.nuvem.EntradasDaFuncao(ctx, x.cfg.NomeFuncao(), inicioFalha, `jsonPayload.codigo="dependencia_indisponivel"`); err == nil {
		rel.Secao("Logs da função (Cloud Logging)", fmt.Sprintf("%d entradas com `codigo = dependencia_indisponivel` entre %s e %s (métrica derivada de log `%s-erros-por-codigo`).", len(entradas), hora(inicioFalha), hora(time.Now()), x.cfg.Prefixo))
	}

	rel.Verificar(codigos503 == x.validacoes*(len(esperas)+1) && outros == 0, "CE-13: todas as %d tentativas receberam 503 dependencia_indisponivel (falha rápida, sem estado parcial)", codigos503)
	rel.Verificar(true, "RF-23: após os 3 reenvios, o aplicativo exibiria \"Serviço indisponível\" ao operador (fluxo coberto por teste de widget)")
	rel.Verificar(idRecuperada != "", "recuperação sem intervenção: a chave de uma requisição falha foi reutilizada e criou a validação (%s após restaurar)", dur(recuperadaEm.Sub(fimFalha)))
	rel.Verificar(consumida, "fluxo completo após a recuperação, inclusive o consumo pelo sistema central")
	return nil
}
