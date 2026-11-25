# Validação do comportamento implementado

Use Go 1.27 e as ferramentas locais instaladas por `make tools`.

```sh
make build
make check
make coverage
```

`make check` verifica formatação, vet, staticcheck, testes e detector de corridas.
Os testes de processo também verificam stdout, stderr e códigos de saída do CLI.
A cobertura mede caminhos executados. Os testes do CLI executam arquivos reais
e verificam erros de leitura, análise e avaliação.

A suíte cobre tokens e posições, escapes existentes, limites numéricos,
precedência e associatividade, estrutura e reconstrução da AST, aritmética,
igualdade, erros e ordem de avaliação, isolamento de bindings e representação
determinística dos valores. Avaliações independentes compartilham uma AST em um
teste paralelo; isso não implica que um Environment mutável possa ser escrito
simultaneamente por várias goroutines.

Os testes de conformidade exercitam records, acesso a campos, chamadas e
pipelines com e sem otimização, incluindo a ordem dos efeitos. Os testes de
dataflow cobrem as operações da biblioteca padrão e um pipeline CSV completo.
O fuzzing do avaliador usa um ambiente isolado sem bindings de acesso a arquivos.

## Avaliação concorrente da AST

O CLI usa `EvaluateConcurrent`, que otimiza a AST antes de agendar lotes.
O limite de workers vem de `GOMAXPROCS`. Com `GOMAXPROCS=1`, a avaliação
ocorre em sequência. Os pontos de entrada `Evaluate` e `EvaluateIn` também
oferecem avaliação sequencial.

Listas e records com trabalho suficiente podem distribuir seus elementos ou
campos entre workers, desde que todos sejam expressões puras que só dependam
de bindings internos ao próprio item. Chamadas, callbacks de dataflow e
acessos a bindings externos seguem a ordem sequencial. Dentro de um callback,
lotes puros podem usar o pool da avaliação em andamento. Os resultados dos
workers são recolhidos pela posição original, inclusive quando mais de um
item falha. Os testes verificam a execução simultânea, o limite do pool,
a ordem dos resultados e erros e a ordem dos efeitos.

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
