# GopherFlow — Especificação da Linguagem

## Visão geral

**GopherFlow** é uma linguagem funcional, dinamicamente tipada e orientada a pipelines de processamento de dados.

Características principais:

- funções de primeira classe;
- closures léxicas;
- bindings imutáveis;
- recursão;
- listas e records;
- condicionais como expressões;
- tipagem dinâmica;
- composição por pipeline com `|>`;
- operações de dataflow fornecidas pela biblioteca padrão.

Operações como `source`, `map`, `filter`, `reduce` e `sink` **não são palavras reservadas**. Elas são funções disponíveis no ambiente padrão da linguagem.

---

## Convenções EBNF

- `{ X }` — zero ou mais ocorrências de `X`;
- `[ X ]` — elemento opcional;
- `X | Y` — alternativa;
- terminais aparecem entre aspas;
- não-terminais aparecem sem aspas.

---

## Gramática sintática

```ebnf
program =
    { definition ";" },
    expression,
    [ ";" ] ;

definition =
    "def", identifier,
    "(", [ parameter_list ], ")",
    "=",
    expression ;

parameter_list =
    identifier,
    { ",", identifier } ;

expression =
      let_expression
    | if_expression
    | lambda_expression
    | pipe_expression ;

let_expression =
    "let", identifier, "=", expression,
    "in", expression ;

if_expression =
    "if", expression,
    "then", expression,
    "else", expression ;

lambda_expression =
    "fn",
    "(", [ parameter_list ], ")",
    "=>",
    expression ;

pipe_expression =
    or_expression,
    { "|>", or_expression } ;

or_expression =
    and_expression,
    { "or", and_expression } ;

and_expression =
    equality_expression,
    { "and", equality_expression } ;

equality_expression =
    comparison_expression,
    { ("==" | "!="), comparison_expression } ;

comparison_expression =
    additive_expression,
    [
        ("<" | "<=" | ">" | ">="),
        additive_expression
    ] ;

additive_expression =
    multiplicative_expression,
    {
        ("+" | "-"),
        multiplicative_expression
    } ;

multiplicative_expression =
    unary_expression,
    {
        ("*" | "/" | "%"),
        unary_expression
    } ;

unary_expression =
      ("not" | "-"), unary_expression
    | postfix_expression ;

postfix_expression =
    primary_expression,
    {
          call_suffix
        | field_suffix
        | index_suffix
    } ;

call_suffix =
    "(", [ argument_list ], ")" ;

argument_list =
    expression,
    { ",", expression } ;

field_suffix =
    ".", identifier ;

index_suffix =
    "[", expression, "]" ;

primary_expression =
      literal
    | identifier
    | "(", expression, ")"
    | list_expression
    | record_expression ;

list_expression =
    "[",
    [ expression, { ",", expression } ],
    "]" ;

record_expression =
    "{",
    [ field, { ",", field } ],
    "}" ;

field =
    identifier, ":", expression ;

literal =
      number
    | string
    | boolean
    | "null" ;

boolean =
    "true" | "false" ;
```

---

## Gramática léxica

```ebnf
identifier =
    letter,
    { letter | digit | "_" } ;

number =
    digit,
    { digit },
    [ ".", digit, { digit } ] ;

string =
    '"',
    { string_character },
    '"' ;

letter =
      "a" | "b" | "c" | "d" | "e" | "f"
    | "g" | "h" | "i" | "j" | "k" | "l"
    | "m" | "n" | "o" | "p" | "q" | "r"
    | "s" | "t" | "u" | "v" | "w" | "x"
    | "y" | "z"
    | "A" | "B" | "C" | "D" | "E" | "F"
    | "G" | "H" | "I" | "J" | "K" | "L"
    | "M" | "N" | "O" | "P" | "Q" | "R"
    | "S" | "T" | "U" | "V" | "W" | "X"
    | "Y" | "Z" ;

digit =
    "0" | "1" | "2" | "3" | "4"
  | "5" | "6" | "7" | "8" | "9" ;
```

### Palavras reservadas

```text
def
let
in
if
then
else
fn
true
false
null
and
or
not
```

---

## Precedência

Da maior para a menor:

| Nível | Operadores / formas |
|---|---|
| 1 | `()` `.` `[]` |
| 2 | `-` `not` |
| 3 | `*` `/` `%` |
| 4 | `+` `-` |
| 5 | `<` `<=` `>` `>=` |
| 6 | `==` `!=` |
| 7 | `and` |
| 8 | `or` |
| 9 | `|>` |

O operador `|>` é associativo à esquerda.

```text
x |> f |> g
```

equivale a:

```text
g(f(x))
```

---

## Semântica básica

### Tipagem dinâmica

Não existem anotações estáticas de tipo na sintaxe.

```gopherflow
fn(x) => x + 1
```

Os valores possuem tipo em tempo de execução. Operações inválidas produzem erro de runtime.

Exemplo:

```gopherflow
1 + "abc"
```

deve produzir `TypeError`.

### Imutabilidade

Bindings introduzidos por `let`, parâmetros de função e definições de topo são imutáveis.

```gopherflow
let x = 10 in x + 1
```

Não existe operador de atribuição destrutiva.

### Funções

Funções são valores de primeira classe.

```gopherflow
fn(x) => x * x
```

Definições nomeadas:

```gopherflow
def square(x) = x * x;
```

Definições de topo podem ser recursivas.

### Records

```gopherflow
{
    city: "Brasilia",
    temperature: 31.5
}
```

Acesso:

```gopherflow
record.temperature
```

### Listas

```gopherflow
[1, 2, 3, 4]
```

Indexação:

```gopherflow
xs[0]
```

---

## Biblioteca padrão de dataflow

A biblioteca padrão deverá fornecer, inicialmente:

```text
source
map
filter
flatMap
take
keyBy
reduce
forEach
sink
```

Esses nomes são bindings comuns do ambiente padrão, não construções especiais do parser.

---

## Exemplo

```gopherflow
def valid(x) =
    x.temperature > 0;

source("data.csv")
|> filter(valid)
|> map(fn(x) => {
    city: x.city,
    temperature: x.temperature * 1.8 + 32
})
|> sink("result.csv");
```

A implementação poderá transformar internamente o pipeline em estágios concorrentes conectados por channels do Go, sem expor detalhes de concorrência na sintaxe da linguagem.
