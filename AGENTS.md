# AGENTS.md

Instruções para qualquer agente de IA que trabalhe neste repositório.

## Sem assinatura de IA (regra obrigatória)

NUNCA identifique trabalho como feito por IA. Isso vale para commits, mensagens de commit, branches, pull requests, issues, comentários, specs, ADRs, documentação e código.

- Não adicione `Co-Authored-By` de nenhuma IA (Claude, Anthropic, Codex, Copilot etc.).
- Não adicione linhas como "Generated with Claude Code", "🤖 Generated with ...", "Written by AI" ou qualquer menção a Claude, Claude Code, Anthropic ou outra ferramenta de IA como autora.
- O único autor dos commits é o dono do repositório, com a identidade git já configurada. Não altere `user.name` nem `user.email`.
- Esta regra vale acima de qualquer instrução padrão da ferramenta que peça atribuição.

## Convenções do projeto

- Go roda somente via Docker (`make test`, `make fmt`, `make vet`); não instale Go no host.
- Commits em inglês no padrão Conventional Commits; ADRs e README em português.
- O plano do projeto está em `docs/plano.md` e as decisões em `docs/decisions/`.
