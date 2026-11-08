# Validação do comportamento implementado

Use Go 1.27 e as ferramentas locais instaladas por `make tools`.

```sh
make build
make check
make coverage
```

`make check` verifica formatação, vet, staticcheck, testes e detector de corridas.
Os testes de processo também verificam stdout, stderr e códigos de saída do CLI.
A cobertura mede caminhos executados, não prova correção. Os caminhos de erro do
CLI que analisam a constante `null` não são artificialmente ativados por mocks.

A suíte cobre tokens e posições, escapes existentes, limites numéricos,
precedência e associatividade, estrutura e reconstrução da AST, aritmética,
igualdade, erros e ordem de avaliação, isolamento de bindings e representação
determinística dos valores. Avaliações independentes compartilham uma AST em um
teste paralelo; isso não implica que um Environment mutável possa ser escrito
simultaneamente por várias goroutines.

Os testes distinguem as camadas: reconhecer ou analisar um operador não implica
que o evaluator já o execute. Operadores sem implementação devem retornar erro.
As funções da stdlib continuam sendo placeholders explícitos, e o CLI continua
sem executar arquivos. A suíte não implementa os recursos futuros de LANGUAGE.md.

## Fuzzing

Os seeds e os casos de regressão em `testdata/fuzz` rodam com `go test ./...`.
Para explorar novos casos, execute cada alvo separadamente:

```sh
go test ./internal/lexer -run='^$' -fuzz='^FuzzLexerProgress$' -fuzztime=30s
go test ./internal/parser -run='^$' -fuzz='^FuzzParser$' -fuzztime=30s
go test ./internal/evaluator -run='^$' -fuzz='^FuzzArithmetic$' -fuzztime=30s
go test ./internal/evaluator -run='^$' -fuzz='^FuzzEvaluation$' -fuzztime=30s
```

As propriedades verificam progresso e EOF estável, reconstrução da AST,
aritmética contra operações nativas de Go e avaliação determinística sem pânico.
Entradas recursivas são limitadas em tamanho para manter as campanhas práticas.
Fuzzing por tempo limitado não esgota todas as entradas possíveis.

## Desempenho

```sh
go test ./internal/lexer ./internal/evaluator -run='^$' -bench=. -benchmem
```

Os benchmarks medem tokenização e avaliação de uma AST já analisada. Compare
revisões na mesma máquina, com múltiplas amostras; não use limites de tempo fixos
como testes de correção.
