# GopherFlow

GopherFlow é um interpretador experimental, escrito em Go, para uma linguagem
funcional orientada a fluxos de dados. O projeto ainda está em desenvolvimento.

## Estado atual

O interpretador implementa:

- análise léxica e sintática com posição de linha e coluna;
- valores `Number`, `String`, `Bool`, `Null`, listas e registros;
- vínculos imutáveis, expressões condicionais, funções e chamadas recursivas;
- operadores aritméticos, relacionais, de igualdade e lógicos;
- curto-circuito para `and` e `or`;
- execução de arquivos e pipelines com leitura e gravação de CSV;
- otimização da AST e avaliação concorrente de expressões independentes;
- testes unitários, de regressão, fuzzing e corrida de dados.

As operações da biblioteca padrão materializam os fluxos em memória e
executam seus estágios de forma síncrona. O CLI aceita um arquivo por
invocação. A gramática e a semântica estão em [LANGUAGE.md](./LANGUAGE.md).

## Requisitos

- Go 1.27;
- `make`;
- acesso à internet apenas para instalar o `staticcheck` com `make tools`.

## Uso

Compile o programa e execute o exemplo:

```sh
make build
./bin/gopherflow examples/hello.gf
```

O exemplo imprime `"Olá, mundo"`. Sem argumentos, o programa avalia a
expressão `null`.

## Exemplo da linguagem

```gopherflow
let temperatura = 31.5 in
    if temperatura > 30 then "quente" else "amena";
```

Salve a expressão em um arquivo `.gf` e passe seu caminho ao CLI para obter
`"quente"`.

## Estrutura do projeto

```text
cmd/gopherflow/       ponto de entrada do programa
internal/token/       tipos de token e posições no código-fonte
internal/lexer/       análise léxica
internal/ast/         nós da árvore sintática
internal/parser/      análise sintática e precedência de operadores
internal/evaluator/   avaliação de expressões e ambientes léxicos
internal/runtime/     valores usados durante a execução
internal/stdlib/      operações de fluxo de dados e CSV
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
