# Constituição do sdd-kit

Princípios que toda spec e todo PR devem respeitar. Mudá-los exige uma spec própria.

1. **Fonte única.** A skill `sdd-delivery` e os modelos de processo vivem só aqui. Os repositórios que adotam o kit consomem cópias geradas, nunca editadas à mão como fonte.
2. **Nunca destruir trabalho alheio.** Adotar ou sincronizar o kit nunca sobrescreve um arquivo existente do repositório de destino sem opt-in explícito (`--force` ou revisão de PR).
3. **Idempotência.** Rodar a adoção duas vezes seguidas produz o mesmo estado; a segunda execução não muda nada.
4. **Agnóstico de linguagem no núcleo.** O processo (skill, specs, templates de issue/PR) não depende de linguagem; o que é específico fica em `template/<linguagem>/`.
5. **Portável.** Todo script roda em Linux, macOS e Windows (sh e PowerShell), sem dependências além do que vem no sistema e do `git`.
6. **Testado no CI.** Todo critério de aceite tem verificação automatizada; o CI bloqueia merge vermelho; shell passa no shellcheck e Markdown no markdownlint.
7. **O processo se aplica a si mesmo.** O sdd-kit é desenvolvido com o processo que ele distribui: specs, épicos, um PR por ticket, CI verde.
8. **Português primeiro.** Specs, issues, PRs e documentação em português; um `README.en.md` para quem não lê português; código e commits em inglês.
