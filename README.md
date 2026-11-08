# GopherFlow

GopherFlow é uma linguagem funcional, dinamicamente tipada e orientada a pipelines de processamento de dados. O interpretador/runtime será implementado em Go.

A especificação sintática da linguagem está em [`LANGUAGE.md`](./LANGUAGE.md).

## Objetivos do projeto

O projeto deve implementar gradualmente:

1. lexer;
2. parser;
3. AST;
4. runtime de valores dinamicamente tipados;
5. evaluator/interpretador;
6. funções e closures;
7. listas e records;
8. biblioteca funcional;
9. operadores de dataflow;
10. runtime concorrente usando goroutines e channels.

A implementação deve priorizar correção, simplicidade e separação clara entre front-end da linguagem e runtime.

---

# Convenções de desenvolvimento

## Versão de Go

O projeto deve usar **Go 1.27 ou superior dentro da série 1.27**, salvo decisão posterior explícita.

O `go.mod` deve declarar:

```text
go 1.27
```

## Formatação

Todo código Go deve ser formatado com a ferramenta padrão:

```bash
gofmt
```

Nenhum estilo manual alternativo deve ser introduzido.

Verificação:

```bash
make fmt-check
```

Correção automática:

```bash
make fmt
```

## Organização de pacotes

Estrutura inicial recomendada:

```text
gopherflow/
├── cmd/
│   └── gopherflow/
│       └── main.go
├── internal/
│   ├── token/
│   ├── lexer/
│   ├── ast/
│   ├── parser/
│   ├── runtime/
│   ├── evaluator/
│   └── stdlib/
├── examples/
├── testdata/
├── docs/
├── LANGUAGE.md
├── README.md
├── Makefile
├── go.mod
└── .gitignore
```

Pacotes internos devem possuir responsabilidade única e dependências direcionadas.

Dependência esperada:

```text
token
  ↑
lexer
  ↑
parser → ast
           ↓
       evaluator
           ↓
        runtime
           ↓
         stdlib
```

Evitar ciclos entre pacotes.

## Código Go

Regras gerais:

- preferir funções pequenas;
- evitar estado global mutável;
- propagar `error` explicitamente;
- não usar `panic` para erros normais da linguagem interpretada;
- usar `panic` somente para invariantes internas realmente impossíveis de recuperar;
- não criar interfaces antes de existir necessidade concreta de polimorfismo;
- evitar abstrações prematuras;
- preferir tipos concretos;
- manter o domínio dinâmico do GopherFlow concentrado no pacote `runtime`;
- não espalhar `any` por todo o código Go;
- erros de lexer, parser e runtime devem carregar posição de origem quando aplicável.

## Valores do GopherFlow

A tipagem dinâmica da linguagem interpretada não significa abandonar tipos no código Go.

O pacote `runtime` deve fornecer uma representação explícita dos valores da linguagem, por exemplo por tipos concretos que satisfaçam uma interface `Value`, ou por outra representação centralizada equivalente.

Categorias esperadas:

```text
Number
String
Bool
Null
List
Record
Function
Builtin
Stream
```

A decisão final da representação deve preservar:

- inspeção segura do tipo dinâmico;
- mensagens de erro claras;
- igualdade bem definida;
- extensibilidade;
- ausência de casts espalhados pelo projeto.

---

# Testes

## Biblioteca

Usar prioritariamente a biblioteca padrão:

```go
testing
```

Não adicionar frameworks externos de teste sem necessidade concreta.

## Convenções

- arquivos de teste: `*_test.go`;
- testes unitários próximos ao pacote testado;
- usar **table-driven tests** quando existirem múltiplos casos da mesma propriedade;
- nomes descritivos;
- testar casos válidos e inválidos;
- erros esperados devem ser testados explicitamente;
- lexer e parser devem possuir testes para posição/linha/coluna quando aplicável.

Exemplo de nome:

```go
func TestLexer_NumberLiteral(t *testing.T)
func TestParser_PipelineAssociativity(t *testing.T)
func TestEvaluator_TypeErrorOnInvalidAddition(t *testing.T)
```

Execução:

```bash
make test
```

---

# Ferramentas de qualidade

## `gofmt`

Formatter oficial do Go.

```bash
make fmt
make fmt-check
```

## `go vet`

Analisador estático distribuído com o toolchain oficial do Go. Detecta construções suspeitas que compilam, mas provavelmente representam erros.

```bash
make vet
```

## `staticcheck`

Analisador estático adicional para detectar bugs, código incorreto, problemas de API e construções não idiomáticas.

Instalação:

```bash
go install honnef.co/go/tools/cmd/staticcheck@latest
```

Execução:

```bash
make staticcheck
```

O projeto deve instalar ferramentas externas em `.tools/bin`, e não depender obrigatoriamente de instalação global.

## Race detector

Verificador dinâmico de data races do Go:

```bash
go test -race ./...
```

Executado por:

```bash
make race
```

Esse alvo passa a ser obrigatório quando o runtime concorrente for introduzido.

## Cobertura

Cobertura unitária:

```bash
go test -coverprofile=coverage.out ./...
```

Resumo:

```bash
go tool cover -func=coverage.out
```

Relatório HTML:

```bash
go tool cover -html=coverage.out -o coverage.html
```

Targets:

```bash
make coverage
make coverage-html
```

Meta inicial recomendada:

```text
>= 80% de cobertura de statements
```

A cobertura é uma métrica auxiliar; não substitui testes de qualidade.

---

# Makefile

O projeto deve fornecer pelo menos:

```text
make help
make tools
make build
make run
make test
make race
make fmt
make fmt-check
make vet
make staticcheck
make lint
make coverage
make coverage-html
make check
make clean
```

Semântica esperada:

```text
lint  = fmt-check + vet + staticcheck
check = lint + test + race
```

`make tools` deve instalar ferramentas externas necessárias em:

```text
.tools/bin/
```

O `Makefile` deve adicionar esse diretório ao `PATH` apenas dentro das próprias regras.

---

# Política de dependências

A biblioteca padrão do Go deve ser preferida.

Nova dependência externa somente deve ser adicionada quando:

1. resolver um problema real;
2. evitar complexidade substancial;
3. possuir manutenção razoável;
4. não substituir trivialmente uma funcionalidade da stdlib.

O lexer, parser e evaluator deverão ser implementados no próprio projeto; não usar geradores de parser na primeira versão.

---

# Git

Branches:

```text
main
feature/<nome>
fix/<nome>
refactor/<nome>
```

Commits devem ser pequenos e semanticamente coerentes.

Formato recomendado:

```text
feat: add token definitions
feat: implement numeric literals in lexer
test: add lexer table tests
fix: preserve source position on parser errors
refactor: isolate runtime value representation
```

Não versionar:

```text
.tools/
coverage.out
coverage.html
bin/
dist/
```

---

# Definition of Done

Uma alteração está pronta quando:

```bash
make check
```

termina com sucesso.

Antes de mergear alterações relevantes ao parser ou runtime, executar também:

```bash
make coverage
```

Nenhuma alteração deve reduzir deliberadamente a cobertura de componentes centrais sem justificativa.
