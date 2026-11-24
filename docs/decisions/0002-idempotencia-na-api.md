# ADR 0002 — Idempotência na API

Status: aceita

## Contexto

O cliente pode repetir `POST /payments` por timeout ou falha de rede, e requisições idênticas podem chegar em paralelo. Criar dois pagamentos para a mesma intenção é o pior erro possível neste sistema. A idempotência do worker (mensagens duplicadas na SQS) é outro problema e fica para a etapa de resiliência.

## Decisão

- `Idempotency-Key` é obrigatório no `POST /payments` (1 a 255 caracteres; senão `400`).
- A chave e um hash da requisição ficam na própria tabela `payments` (`idempotency_key` com constraint `UNIQUE`, `request_hash`). O hash é o SHA-256 dos valores já normalizados (`valor|MOEDA`), então `brl` e `BRL` são a mesma requisição.
- A garantia vem do banco, não de checagem na aplicação: o repositório faz `INSERT ... ON CONFLICT (idempotency_key) DO NOTHING`. Inserções concorrentes com a mesma chave se serializam no Postgres e só uma grava; as outras leem o registro existente. Não há "SELECT antes de INSERT" (que teria janela de corrida).
- Resultado:
  - chave nova: `201` com o pagamento `PENDING`;
  - mesma chave e mesmo corpo: `200` com o mesmo pagamento (o estado atual, que pode ter avançado), sem criar outro;
  - mesma chave e corpo diferente: `422` com código `idempotency_conflict`.
- `422` e não `409`: a requisição é bem formada, mas semanticamente inválida para aquela chave (reuso indevido). `409` fica reservado para o caso "requisição original ainda em andamento", que não existe aqui porque a criação é uma única instrução atômica.
- Erros seguem `{"error": {"code": "...", "message": "..."}}`. Validação de corpo, valor, moeda e chave retorna `400`.
- O ID do pagamento é UUID v7, gerado fora do domínio.

## Consequências

- Uma única tabela e uma única instrução tornam a criação atômica. Na etapa da Outbox, o evento será inserido na mesma transação, apenas quando `created = true`.
- As chaves são globais: ainda não existe o conceito de cliente. Quando houver autenticação, a unicidade deve passar a ser `(cliente, chave)`.
- Chaves nunca expiram. Uma política de expiração (ex.: 24 h) exigiria separar a chave em tabela própria; foi adiado por não ser necessário agora.
- O hash cobre só `amount` e `currency`. Novos campos do corpo precisam entrar no hash.
- A réplica devolve `200`, não `201`, para o cliente distinguir repetição de criação; o corpo é idêntico.
