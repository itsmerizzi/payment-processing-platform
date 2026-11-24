# ADR 0001 — Estados do pagamento

Status: aceita

## Contexto

O plano (`docs/plano.md`, seção 4) define cinco estados e deixa as transições em aberto. O processamento é assíncrono e a SQS entrega mensagens *at least once*, então o mesmo pagamento pode ser processado mais de uma vez, e o provedor simulado pode falhar de forma transitória.

## Decisão

```text
PENDING ──► PROCESSING ──► APPROVED
               │  ▲  │
               └──┘  ├────► DECLINED
        (reprocesso) └────► FAILED
```

- `PENDING → PROCESSING`: o worker iniciou o processamento.
- `PROCESSING → PROCESSING`: permitido sem erro. É o reprocessamento (mensagem duplicada ou nova tentativa após falha transitória).
- `PROCESSING → APPROVED | DECLINED | FAILED`: resultado final.
- `APPROVED`, `DECLINED` e `FAILED` são finais; qualquer outra transição retorna `ErrInvalidTransition`.

Diferença entre os resultados:

| Situação | Estado | Natureza |
|---|---|---|
| `DECLINED` | final | Decisão de negócio do provedor (ex.: saldo insuficiente). Não adianta tentar de novo. |
| `FAILED` | final | Falha técnica definitiva: as tentativas se esgotaram (a mensagem vai para a DLQ). |
| Falha transitória | continua `PROCESSING` | Timeout, erro 5xx etc. Não muda o estado; a mensagem volta à fila e o pagamento é reprocessado. |

## Consequências

- Não existe estado `RETRYING`: o número de tentativas e o backoff pertencem à mensageria/worker, não ao domínio.
- Reprocessar a partir de `PROCESSING` é seguro para a máquina de estados, mas **não** garante idempotência dos efeitos (chamada ao provedor, webhook). Isso será tratado na etapa de resiliência (idempotência do worker).
- Um pagamento já final recebendo nova mensagem gera `ErrInvalidTransition`; o worker deve tratar isso como duplicata e descartar a mensagem, não como falha.
- Se `PROCESSING` ficar preso (worker morreu), a recuperação depende de a mensagem reaparecer na fila (visibility timeout).
