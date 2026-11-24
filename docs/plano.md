# Payment Processing Platform

## 1. Objetivo do projeto

Construir uma plataforma **simulada de processamento de pagamentos**, usando Go e AWS, para praticar desenvolvimento backend, arquitetura de software, mensageria, infraestrutura como código e operação de serviços em nuvem.

O projeto não será um gateway real e não terá integração com bancos ou adquirentes comerciais na primeira versão. Um provedor simulado representará respostas de aprovação, recusa e falhas técnicas.

**Prioridade:** construir primeiro uma aplicação funcional localmente, com Docker. Depois, evoluir a infraestrutura e experimentar serviços AWS em ambientes temporários e sem custo sempre que possível. A certificação AWS Solutions Architect – Associate (SAA-C03) permanece como objetivo futuro, mas não é o foco inicial.

## 2. Escopo funcional

- Criar pagamentos por uma API REST em Go.
- Consultar o estado de um pagamento.
- Aceitar uma `Idempotency-Key` para evitar a criação duplicada de pagamentos.
- Persistir pagamentos em PostgreSQL.
- Processar pagamentos de forma assíncrona usando workers.
- Simular um provedor que pode aprovar, recusar ou apresentar falha técnica.
- Notificar o sistema solicitante por webhooks.
- Implementar retentativas para falhas transitórias e uma Dead Letter Queue (DLQ) para mensagens que não puderem ser processadas.
- Garantir que o processamento de mensagens duplicadas não provoque efeitos duplicados.

## 3. Arquitetura proposta

A arquitetura será baseada em **Hexagonal Architecture (Ports & Adapters)**, com conceitos de DDD e processamento orientado a eventos.

### Componentes da aplicação

| Componente | Responsabilidade |
|---|---|
| Payment API | Receber solicitações, validar dados, aplicar idempotência e registrar pagamentos |
| PostgreSQL | Persistir pagamentos, chaves de idempotência e eventos da Outbox |
| Outbox Publisher | Publicar na fila os eventos registrados na tabela Outbox |
| Amazon SQS | Transportar mensagens de processamento de forma assíncrona |
| Payment Worker | Consumir mensagens e executar o processamento do pagamento |
| Payment Provider Simulator | Simular aprovação, recusa e falhas técnicas |
| Webhook Dispatcher | Enviar o resultado do pagamento ao sistema solicitante |
| CloudWatch | Centralizar logs e métricas na implantação AWS |

### Fluxo principal

1. Um sistema externo envia uma solicitação para a Payment API.
2. A API valida a solicitação e verifica a chave de idempotência.
3. A API grava o pagamento como `PENDING` no PostgreSQL.
4. Na mesma transação, registra um evento na tabela Outbox.
5. O Outbox Publisher lê os eventos pendentes e os publica na SQS.
6. O Payment Worker consome a mensagem e chama a interface do provedor de pagamentos.
7. O provedor simulado retorna aprovação, recusa ou falha técnica.
8. O Worker atualiza o estado do pagamento no PostgreSQL.
9. O Webhook Dispatcher envia uma notificação ao sistema externo.
10. Falhas transitórias podem gerar novas tentativas; mensagens que excederem o limite de tentativas podem seguir para a DLQ.

## 4. Estados do pagamento

Estados iniciais propostos:

| Estado | Significado |
|---|---|
| `PENDING` | Pagamento registrado e aguardando processamento |
| `PROCESSING` | Pagamento em processamento |
| `APPROVED` | Pagamento aprovado pelo provedor |
| `DECLINED` | Pagamento recusado por uma decisão de negócio do provedor |
| `FAILED` | Falha técnica definitiva após as tentativas de recuperação previstas |

**Distinção importante:** `DECLINED` não é a mesma coisa que `FAILED`. Uma recusa é um resultado de negócio; uma falha técnica é um problema de execução. Falhas transitórias não devem necessariamente levar o pagamento a `FAILED`, pois podem ser recuperadas por retentativas.

As transições exatas entre estados serão detalhadas posteriormente.

## 5. Decisões e conceitos técnicos importantes

### Idempotência

A API deve reconhecer solicitações repetidas com a mesma chave de idempotência, evitando criar pagamentos duplicados. O Worker também precisa ser idempotente, pois a SQS trabalha com entrega *at least once*: uma mensagem pode ser entregue mais de uma vez.

### Transactional Outbox

O pagamento e o evento que solicita seu processamento serão gravados na mesma transação do banco. Isso reduz o risco de o pagamento ser salvo sem que o evento correspondente seja publicado na fila.

### Mensageria e resiliência

A SQS desacopla a API do processamento. O sistema poderá usar retentativas com *exponential backoff* e DLQ para mensagens que falharem repetidamente.

### Webhooks

O sistema notificará o solicitante quando houver um resultado relevante. O envio de webhooks também deverá considerar falhas, retentativas e prevenção de notificações duplicadas.

## 6. Tecnologias previstas

| Área | Tecnologia |
|---|---|
| Linguagem | Go |
| API | REST |
| Banco de dados | PostgreSQL |
| Containers | Docker e Docker Compose |
| Mensageria local | ElasticMQ ou LocalStack |
| Mensageria AWS | Amazon SQS |
| Computação AWS | Amazon ECS com Fargate |
| Registro de imagens | Amazon ECR |
| Banco gerenciado AWS | Amazon RDS for PostgreSQL |
| Observabilidade AWS | Amazon CloudWatch |
| Identidade e permissões | AWS IAM |
| Rede AWS | Amazon VPC |
| Infraestrutura como código | Terraform |
| CI/CD | GitHub Actions |

Não há necessidade de Kubernetes/EKS na primeira versão. ECS com Fargate é suficiente para executar os containers sem administrar diretamente instâncias EC2.

## 7. Containers e execução

No ambiente local, Docker Compose poderá executar:

- Payment API;
- Payment Worker;
- Outbox Publisher;
- Webhook Dispatcher;
- PostgreSQL;
- simulador local de SQS, como ElasticMQ ou LocalStack.

Os componentes da aplicação serão processos independentes. Inicialmente, eles poderão reutilizar a mesma imagem Docker, executando comandos diferentes conforme o papel de cada serviço.

Na AWS, a proposta é usar ECS/Fargate para os processos da aplicação, ECR para armazenar as imagens, SQS como serviço gerenciado e RDS para PostgreSQL. A fila SQS não precisa ser executada em um container próprio.

## 8. Estrutura inicial sugerida

```text
payment-processing-platform/
├── cmd/
│   ├── api/
│   ├── worker/
│   ├── outbox/
│   └── webhook-dispatcher/
├── internal/
│   ├── domain/
│   ├── application/
│   ├── infrastructure/
│   └── interfaces/
├── migrations/
├── deployments/
│   └── terraform/
├── docs/
│   ├── architecture/
│   └── decisions/
├── tests/
├── Dockerfile
├── docker-compose.yml
└── README.md
```

A estrutura é uma proposta inicial e poderá mudar conforme a implementação evoluir.

## 9. Estratégia de desenvolvimento

1. **Domínio e desenho:** definir entidades, regras, estados e contratos principais.
2. **Base local:** criar o projeto Go, PostgreSQL e ambiente Docker Compose.
3. **API e persistência:** implementar criação e consulta de pagamentos e idempotência.
4. **Processamento assíncrono:** adicionar Outbox, fila e Worker.
5. **Resiliência:** implementar idempotência no Worker, retentativas e DLQ.
6. **Webhooks:** implementar despacho e retentativas de notificações.
7. **Qualidade:** adicionar testes unitários e de integração, logs estruturados e métricas.
8. **CI/CD:** configurar pipeline no GitHub Actions.
9. **AWS:** mapear os componentes locais para serviços AWS e experimentar implantação em ambiente temporário.
10. **Terraform:** introduzir Infrastructure as Code gradualmente, depois de compreender os recursos e suas relações na AWS.

## 10. Estratégia de custo

A preferência é manter o desenvolvimento local com Docker, sem infraestrutura AWS persistente e paga durante a fase inicial.

O AWS Builder Center e seus ambientes de aprendizagem/sandbox poderão ser usados para experimentos, desde que o workshop disponível ofereça os serviços necessários. A disponibilidade e as limitações dependem do ambiente específico; não se deve presumir que todos os serviços AWS estarão disponíveis gratuitamente.

Antes de criar recursos persistentes na AWS, verificar preços, limites e possíveis cobranças. Não provisionar recursos pagos sem avaliar o custo.

## 11. Documentação de arquitetura

Durante o desenvolvimento, registrar decisões importantes como ADRs (*Architecture Decision Records*), por exemplo:

- Por que usar SQS em vez de processamento síncrono?
- Por que ECS/Fargate em vez de EC2 ou EKS?
- Como garantir idempotência na API e no Worker?
- Como evitar perda de eventos com Transactional Outbox?
- Como tratar falhas e reprocessamentos?
- Como proteger e repetir o envio de webhooks?

## 12. Rotina de estudo

Disponibilidade estimada: aproximadamente **7 horas por semana**.

A prioridade é aprender construindo o projeto. O estudo para a certificação SAA-C03 será conectado às decisões e aos serviços utilizados, sem substituir a implementação prática.

---

**Resumo:** o projeto será uma plataforma de pagamentos simulada em Go, inicialmente local e conteinerizada, com PostgreSQL, processamento assíncrono, SQS, Outbox, idempotência, retentativas, DLQ e webhooks. Depois, será evoluída para AWS com ECS/Fargate, RDS, ECR, CloudWatch e Terraform, priorizando experimentos sem custo e documentando as decisões arquiteturais.
