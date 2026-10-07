package main

import (
	"flag"
	"fmt"
	"io"
	"strings"
)

// axyn decide answers the question a stopped run asked, from the terminal: the answer
// becomes a decision in the spec and the run resumes on its own.

// pendingQuestion is the question and its lettered options in the run's final message.
func pendingQuestion(msg string) (question string, options map[string]string) {
	options = map[string]string{}
	for _, l := range strings.Split(msg, "\n") {
		t := strings.TrimSpace(l)
		if i := strings.Index(t, "pergunta ao usuário"); i >= 0 {
			question = t[i:]
		}
		if len(t) > 3 && t[1] == ')' && t[0] >= 'A' && t[0] <= 'Z' {
			options[t[:1]] = strings.TrimSpace(t[2:])
		}
	}
	return question, options
}

func runDecideCmd(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("axyn decide", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("dir", ".", "raiz do repositório")
	noResume := fs.Bool("no-resume", false, "só grava a decisão, sem retomar a execução")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	answer := strings.TrimSpace(strings.Join(fs.Args(), " "))
	if answer == "" {
		_, _ = fmt.Fprintln(stderr, "axyn decide: informe a resposta, por exemplo: axyn decide B (ou axyn decide \"use só o pacote cmd\")")
		return exitUsage
	}
	st, err := loadRun(*dir, "")
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "axyn decide: %v\n", err)
		return exitFail
	}
	question, options := pendingQuestion(st.Message)
	s := &mcpServer{dir: *dir}
	pl, path, perr := s.loadPlan()
	_, open := s.openTicket()
	if question == "" {
		question = "pergunta do axyn na execução " + st.ID
		if open != nil {
			question = "como seguir no ticket «" + open.Title + "»"
		}
	}
	if opt, ok := options[strings.ToUpper(answer)]; ok {
		if strings.HasPrefix(opt, "outra instrução") {
			_, _ = fmt.Fprintln(stderr, "axyn decide: escreva a instrução entre aspas, por exemplo: axyn decide \"mexa só no main.go\"")
			return exitUsage
		}
		answer = strings.ToUpper(answer) + ") " + opt
	}
	msg, err := s.toolDecide(question, answer)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "axyn decide: %v\n", err)
		return exitFail
	}
	// With the decision, the open ticket starts the recovery ladder again: the earlier
	// attempts stay in the plan (axyn history), but no longer count.
	if perr == nil && open != nil {
		if pl, path, err = s.loadPlan(); err == nil {
			for i := range pl.Tickets {
				if pl.Tickets[i].ID == open.ID {
					pl.Tickets[i].Earlier = append(pl.Tickets[i].Earlier, pl.Tickets[i].Attempts...)
					pl.Tickets[i].Attempts = nil
				}
			}
			_ = s.savePlan(pl, path)
		}
	}
	_, _ = fmt.Fprintf(stdout, "%s: %s\n", msg, answer)
	if *noResume {
		_, _ = fmt.Fprintln(stdout, "para continuar: axyn run --resume")
		return exitOK
	}
	id, err := s.startRun("", true)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "axyn decide: a decisão foi gravada, mas não consegui retomar: %v; rode: axyn run --resume\n", err)
		return exitFail
	}
	_, _ = fmt.Fprintf(stdout, "execução %s retomada em segundo plano; para acompanhar: axyn status --watch\n", id)
	return exitOK
}
