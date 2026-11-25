# GopherFlow

GopherFlow é um interpretador experimental, escrito em Go, para uma linguagem
funcional orientada a fluxos de dados. O projeto ainda está em desenvolvimento.

## Estado atual

A base implementa:

- análise léxica com posição de linha e coluna;
- análise de uma expressão por entrada;
- árvore sintática abstrata (AST);
- valores `Number`, `String`, `Bool` e `Null`;
- vínculos imutáveis com `let ... in ...`;
- expressões condicionais com `if ... then ... else ...`;
- operadores aritméticos, relacionais, de igualdade e lógicos;
- curto-circuito para `and` e `or`;
- testes unitários, de regressão, fuzzing e corrida de dados.

Ainda não estão disponíveis:

- execução de arquivos ou leitura interativa pelo programa de linha de comando;
- funções, chamadas, listas e registros no avaliador;
- execução do operador de fluxo `|>`;
- funções da biblioteca padrão (`source`, `map`, `filter` e `sink`);
- processamento concorrente de fluxos.

A sintaxe planejada está descrita em [LANGUAGE.md](./LANGUAGE.md). Esse documento
inclui recursos que ainda não foram implementados.

## Requisitos

- Go 1.27;
- `make`;
- acesso à internet apenas para instalar o `staticcheck` com `make tools`.

## Uso

Compile o programa:

```sh
make build
./bin/gopherflow
```

No estado atual, o programa avalia internamente a expressão `null` e imprime:

```text
null
```

Qualquer argumento é recusado porque a execução de arquivos ainda não foi
implementada.

## Exemplo da linguagem

```gopherflow
let temperatura = 31.5 in
    if temperatura > 30 then "quente" else "amena";
```

O exemplo representa a sintaxe já aceita pelo analisador, mas ainda não pode ser
executado diretamente pelo programa de linha de comando.

## Estrutura do projeto

```text
cmd/gopherflow/       ponto de entrada do programa
internal/token/       tipos de token e posições no código-fonte
internal/lexer/       análise léxica
internal/ast/         nós da árvore sintática
internal/parser/      análise sintática e precedência de operadores
internal/evaluator/   avaliação de expressões e ambientes léxicos
internal/runtime/     valores usados durante a execução
internal/stdlib/      funções padrão ainda não implementadas
docs/                 documentação de validação
examples/             exemplos de código GopherFlow
```

Os pacotes em `internal` são detalhes da implementação e não formam uma API
pública estável.

## Desenvolvimento

Comandos principais:

```sh
make help        # lista os alvos disponíveis
make test        # executa os testes
make fmt         # formata os arquivos Go
make lint        # verifica formatação e análise estática
make check       # executa todas as verificações
make coverage    # gera o resumo de cobertura
make clean       # remove artefatos gerados
```

`make tools` instala o `staticcheck` em `.tools/bin`, sem exigir instalação
global.

Ao alterar o código:

- formate os arquivos com `gofmt`;
- mantenha cada pacote com uma responsabilidade clara;
- retorne erros em vez de usar `panic` para falhas esperadas;
- prefira a biblioteca padrão do Go;
- adicione testes para comportamentos válidos e inválidos.

Detalhes sobre a suíte estão em [docs/TESTING.md](./docs/TESTING.md).
