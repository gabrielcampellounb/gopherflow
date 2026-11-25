# GopherFlow — Especificação da Linguagem

## Visão geral

**GopherFlow** é uma linguagem funcional, dinamicamente tipada e orientada a
fluxos de processamento de dados.

Esta especificação descreve a direção planejada da linguagem. Consulte o
[`README.md`](./README.md) para saber quais recursos já estão implementados.

Características principais:

- funções de primeira classe;
- fechamentos léxicos;
- vínculos imutáveis;
- recursão;
- listas e registros;
- condicionais como expressões;
- tipagem dinâmica;
- composição de fluxos com `|>`;
- operações de fluxo de dados fornecidas pela biblioteca padrão.

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

Dentro de strings, aspas e barras invertidas devem ser escapadas. Os escapes
reconhecidos são `\"`, `\\`, `\n`, `\r` e `\t`. LF, CRLF e CR isolado contam como quebra de linha. Uma quebra de linha
literal dentro de uma string encerra a leitura com erro. As colunas dos
diagnósticos contam pontos de código Unicode, e os offsets contam bytes.

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

Quando o operando direito é uma chamada, o valor à esquerda entra como
primeiro argumento. `x |> f(a, b)` equivale a `f(x, a, b)`.

---

## Limites do analisador

O analisador aceita até 4.096 níveis de aninhamento da sintaxe e árvores de
expressão com até 1.024 nós por caminho da raiz a uma folha. Cadeias longas
de operadores ou sufixos também contam para a profundidade da árvore. O
limite de profundidade permite listas e records largos.

## Semântica básica

### Tipagem dinâmica

Não existem anotações estáticas de tipo na sintaxe.

```gopherflow
fn(x) => x + 1
```

Os valores possuem tipo em tempo de execução. Operações inválidas produzem erro
de tipo.

Exemplo:

```gopherflow
1 + "abc"
```

deve produzir um erro de tipo.

### Igualdade de coleções

Listas comparam seus elementos na ordem. Records comparam as chaves e os
respectivos valores, independentemente da ordem dos campos. Streams comparam
a ordem das colunas, quando presente, e os elementos na ordem. As comparações
aplicam as mesmas regras aos valores aninhados. Por exemplo, `[0] == [-0]`
retorna `true`.

### Imutabilidade

Vínculos introduzidos por `let`, parâmetros de função e definições de topo são
imutáveis.

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

Definições de topo podem ser recursivas. O interpretador aceita até 1024
chamadas de função ativas, incluindo callbacks de dataflow. A próxima
chamada produz um erro de execução.

A igualdade entre funções compara a identidade do valor. Uma função
comparada com uma referência à mesma função é igual a si própria. Duas
closures criadas em avaliações distintas têm identidades distintas.

### Registros

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

Chaves também podem ser acessadas por uma expressão `String`. Isso permite
consultar nomes de colunas CSV e chaves de `keyBy` que não são identificadores.

```gopherflow
record["air temperature"]
```

Na saída do CLI, os nomes de campos que não são identificadores válidos
aparecem entre aspas, com escapes quando necessário.

Os campos são avaliados na ordem em que aparecem. Nomes repetidos são erros
de sintaxe, e o acesso a um campo ausente produz um erro em tempo de execução.

### Listas

```gopherflow
[1, 2, 3, 4]
```

Indexação:

```gopherflow
xs[0]
```

---

## Biblioteca padrão para fluxo de dados

Os nomes das operações de fluxo são vínculos comuns do ambiente padrão. As
operações aceitam `List` ou `Stream` e percorrem os elementos na ordem de
entrada. Nesta implementação, os streams são materializados em memória e as
operações executam de forma síncrona.

| Operação | Resultado |
|---|---|
| `source(path)` | Stream de records lidos de um CSV local. |
| `map(data, f)` | Stream com o resultado de `f(item)` para cada item. |
| `filter(data, f)` | Stream com os itens para os quais `f(item)` retorna `true`. O retorno deve ser `Bool`. |
| `flatMap(data, f)` | Stream com os elementos das listas ou streams retornados por `f(item)`, concatenados em ordem. |
| `take(data, n)` | Stream com até os primeiros `n` itens. `n` deve ser um inteiro não negativo. |
| `keyBy(data, f)` | Record que associa cada chave `String` retornada por `f(item)` ao stream de itens dessa chave. |
| `reduce(data, initial, f)` | Resultado de aplicar `f(acumulador, item)` sucessivamente. Em uma entrada vazia, retorna `initial`. |
| `forEach(data, f)` | Executa `f(item)` pela ordem de entrada e retorna `null`. |
| `sink(data, path)` | Grava os records em um CSV local e retorna `null`. |

`source` aceita arquivos UTF-8 com ou sem BOM. Usa a primeira linha como
cabeçalho e exige nomes de coluna únicos e não vazios. Uma primeira linha
vazia é rejeitada. Um arquivo sem conteúdo produz um stream vazio. Valores
decimais finitos são lidos como `Number`. As demais células são lidas como
`String`, inclusive células vazias. Quebras de linha
dentro de campos entre aspas são preservadas.

`sink` exige que todas as linhas tenham os mesmos campos e que seus valores
sejam `String`, `Number`, `Bool` ou `Null`. Valores `Null` viram células vazias.
A ordem de colunas do CSV de origem é preservada por `filter`, `take` e
`keyBy`. Para records sem cabeçalho de origem, `sink` ordena as colunas pelo
nome. Um stream vazio com cabeçalho grava somente o cabeçalho. A gravação
substitui o destino após a geração bem-sucedida do CSV. Ao atualizar um
arquivo regular existente, preserva os bits de permissão. No Linux, também
preserva o proprietário, o grupo e a ACL POSIX de acesso, quando presente.
Se a preservação falhar, o arquivo original permanece intacto. Links
simbólicos e outros destinos existentes que não sejam arquivos regulares
são rejeitados.

Erros de callbacks interrompem a operação e indicam a posição do item na
entrada, começando em 1.

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

A implementação poderá transformar internamente o fluxo em estágios concorrentes
conectados por canais do Go, sem expor detalhes de concorrência na sintaxe da
linguagem.
