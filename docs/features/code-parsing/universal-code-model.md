# Universal Code Model (UCM)

## Philosophy

Different programming languages have vastly different paradigms (OOP, functional, procedural, declarative, logic-based), but they share common conceptual building blocks. The Universal Code Model abstracts language-specific constructs into universal concepts that enable cross-language analysis.

## Core Abstraction Principles

1. **Function Points** - Executable logic units (functions, methods, procedures, predicates)
2. **Groupings** - Organizational containers (classes, modules, files, packages, namespaces)
3. **Data Definitions** - Data structures (variables, fields, properties, constants)
4. **Type Definitions** - Type systems (classes, interfaces, structs, types, prototypes)
5. **Dependencies** - Relationships between code units (imports, includes, uses, requires)
6. **Documentation** - Code documentation (comments, docstrings, annotations)

## Universal Element Types

### 1. Function Points (Executable Logic)

Maps to: functions, methods, procedures, subroutines, predicates, rules, lambdas, closures

**Element Type:** `code_function`

**Unified Attributes:**
- `function_name`: Name of the executable unit
- `signature`: Complete signature (parameters + return type)
- `namespace`: Fully qualified path (module.class.function)
- `function_kind`: function | method | procedure | predicate | rule | lambda
- `scope`: global | instance | static | class | local
- `visibility`: public | private | protected | internal | package | exported

**Language Mappings:**
```yaml
Go:
  function: "func Foo()"
  method: "func (r *Receiver) Method()"

Java:
  method: "public void method()"
  static_method: "public static void method()"
  constructor: "public ClassName()"

JavaScript/TypeScript:
  function: "function foo() {}"
  method: "class.method() {}"
  arrow_function: "const foo = () => {}"

Python:
  function: "def foo():"
  method: "def method(self):"
  static_method: "@staticmethod def method():"
  lambda: "lambda x: x + 1"

Perl:
  subroutine: "sub foo { }"

COBOL:
  paragraph: "PARAGRAPH-NAME."
  section: "SECTION-NAME SECTION."

Prolog:
  predicate: "predicate(Args) :- body."
  rule: "rule(X) :- condition(X)."

SQL:
  stored_procedure: "CREATE PROCEDURE proc_name"
  function: "CREATE FUNCTION func_name"
```

### 2. Groupings (Organizational Containers)

Maps to: classes, modules, packages, namespaces, files, divisions, records

**Element Type:** `code_grouping`

**Unified Attributes:**
- `grouping_name`: Name of the container
- `grouping_kind`: class | module | package | namespace | file | division | record | schema
- `namespace`: Parent path
- `visibility`: public | private | internal

**Language Mappings:**
```yaml
Go:
  package: "package mypackage"
  struct: "type MyStruct struct { }"
  interface: "type MyInterface interface { }"
  file: "filename.go"

Java:
  package: "package com.example;"
  class: "public class MyClass { }"
  interface: "public interface MyInterface { }"
  enum: "public enum MyEnum { }"

JavaScript/TypeScript:
  module: "export module MyModule { }"
  class: "class MyClass { }"
  namespace: "namespace MyNamespace { }"

Python:
  module: "# file.py"
  package: "# __init__.py"
  class: "class MyClass:"

Perl:
  package: "package MyPackage;"
  module: "# Module.pm"

COBOL:
  program: "PROGRAM-ID. PROGRAM-NAME."
  division: "DATA DIVISION."
  section: "WORKING-STORAGE SECTION."

Prolog:
  module: ":- module(module_name, [exports])."
  file: "filename.pl"

SQL:
  schema: "CREATE SCHEMA schema_name"
  database: "CREATE DATABASE db_name"
```

### 3. Data Definitions (Variables & Fields)

Maps to: variables, fields, properties, attributes, constants, parameters, arguments

**Element Type:** `code_data`

**Unified Attributes:**
- `data_name`: Name of the data element
- `data_kind`: variable | field | property | constant | parameter | attribute
- `data_type`: Type information (string, varies by language)
- `scope`: global | instance | local | static | class
- `mutability`: mutable | immutable | const

**Language Mappings:**
```yaml
Go:
  variable: "var x int"
  constant: "const Pi = 3.14"
  field: "struct { Name string }"
  parameter: "func(x int)"

Java:
  field: "private int count;"
  static_field: "public static final int MAX = 100;"
  local_variable: "int x = 0;"
  parameter: "void method(int x)"

JavaScript/TypeScript:
  variable: "let x = 0;"
  constant: "const PI = 3.14;"
  property: "class { x: number; }"

Python:
  variable: "x = 0"
  class_variable: "class Foo: x = 0"
  instance_variable: "self.x = 0"

Perl:
  scalar: "$variable"
  array: "@array"
  hash: "%hash"

COBOL:
  data_item: "01 RECORD-NAME PIC X(10)."
  field: "05 FIELD-NAME PIC 9(5)."

Prolog:
  variable: "Variable (starts with uppercase)"
  atom: "atom (lowercase)"
```

### 4. Type Definitions

Maps to: types, classes, interfaces, structs, records, enums, typedefs, prototypes

**Element Type:** `code_type`

**Unified Attributes:**
- `type_name`: Name of the type
- `type_kind`: class | interface | struct | enum | typedef | record | prototype
- `base_types`: Inheritance/implementation relationships
- `visibility`: public | private | internal

**Language Mappings:**
```yaml
Go:
  struct: "type Name struct { }"
  interface: "type Name interface { }"
  type_alias: "type Name = OtherType"

Java:
  class: "class Name extends Base implements IFace { }"
  interface: "interface Name { }"
  enum: "enum Name { VALUES }"
  record: "record Name(int x) { }"

JavaScript/TypeScript:
  class: "class Name { }"
  interface: "interface Name { }" (TS only)
  type: "type Name = { }" (TS only)

Python:
  class: "class Name(Base):"
  protocol: "class Name(Protocol):" (3.8+)

COBOL:
  record: "01 RECORD-LAYOUT."

SQL:
  table: "CREATE TABLE table_name"
  view: "CREATE VIEW view_name"
  type: "CREATE TYPE type_name"
```

### 5. Dependencies (Relationships)

Maps to: imports, includes, uses, requires, references, calls

**Element Type:** `code_dependency`

**Unified Attributes:**
- `dependency_kind`: import | include | use | require | reference
- `target_namespace`: What is being imported
- `dependency_alias`: Alias or nickname
- `is_conditional`: Whether dependency is conditional

**Language Mappings:**
```yaml
Go:
  import: "import \"package/path\""
  import_alias: "import alias \"package/path\""

Java:
  import: "import java.util.List;"
  static_import: "import static java.lang.Math.PI;"

JavaScript/TypeScript:
  import: "import { x } from 'module';"
  require: "const x = require('module');"
  dynamic_import: "await import('module');"

Python:
  import: "import module"
  from_import: "from module import func"
  import_alias: "import module as alias"

Perl:
  use: "use Module;"
  require: "require Module;"

COBOL:
  copy: "COPY COPYBOOK."

Prolog:
  use_module: ":- use_module(library(lists))."
  consult: ":- consult('file.pl')."

SQL:
  foreign_key: "FOREIGN KEY REFERENCES table(column)"
```

### 6. Documentation

Maps to: comments, docstrings, annotations, pragmas, decorators

**Element Type:** `code_documentation`

**Unified Attributes:**
- `doc_kind`: comment | docstring | annotation | pragma | decorator
- `doc_content`: The documentation text
- `doc_target`: What is being documented

**Language Mappings:**
```yaml
Go:
  comment: "// comment"
  doc_comment: "// Package-level doc"
  block_comment: "/* block */"

Java:
  javadoc: "/** @param x description */"
  comment: "// comment"
  annotation: "@Override"

JavaScript/TypeScript:
  jsdoc: "/** @param {number} x */"
  comment: "// comment"
  decorator: "@decorator" (TS only)

Python:
  docstring: "\"\"\"Doc string\"\"\""
  comment: "# comment"
  decorator: "@decorator"

Perl:
  pod: "=head1 HEADING"
  comment: "# comment"

COBOL:
  comment: "*> comment"

Prolog:
  comment: "% comment"
  block_comment: "/* block */"

SQL:
  comment: "-- comment"
  block_comment: "/* block */"
```

## Promoted Fields Strategy

The Element struct's promoted fields should use **universal abstractions**:

```go
// Universal code element promoted fields
FunctionName  *string  // Universal: function/method/procedure/predicate name
ClassName     *string  // Universal: class/struct/interface/module name
Namespace     *string  // Universal: fully qualified path
LineNumber    *int     // Universal: source line number
Signature     *string  // Universal: complete signature (varies by language)
```

Language-specific details go in `Metadata`:

```go
Metadata: map[string]interface{}{
    // Universal metadata
    "code_element_kind": "function",  // function | method | class | module | etc.
    "scope": "instance",               // global | instance | static | local
    "visibility": "public",            // public | private | protected | etc.

    // Language-specific metadata
    "language": "go",
    "receiver_type": "*MyStruct",      // Go-specific
    "is_pointer_receiver": true,       // Go-specific
    "is_exported": true,               // Go-specific

    // Or for Java
    "modifiers": ["public", "static"], // Java-specific
    "throws": ["IOException"],         // Java-specific

    // Or for Python
    "decorators": ["@property"],       // Python-specific
    "is_async": false,                 // Python-specific
}
```

## Cross-Language Query Examples

### Find all public functions across languages

```sql
SELECT
    function_name,
    namespace,
    line_number,
    metadata->>'language' as language
FROM elements
WHERE element_type = 'code_function'
  AND metadata->>'visibility' = 'public'
```

### Find all class definitions across languages

```sql
SELECT
    class_name,
    namespace,
    metadata->>'code_element_kind' as kind,
    metadata->>'language' as language
FROM elements
WHERE element_type IN ('code_grouping', 'code_type')
  AND metadata->>'code_element_kind' IN ('class', 'struct', 'interface')
```

### Find dependencies on specific packages

```sql
SELECT DISTINCT
    namespace,
    metadata->>'dependency_kind' as import_type,
    metadata->>'language' as language
FROM elements
WHERE element_type = 'code_dependency'
  AND content LIKE '%database%'
```

### Cross-language policy enforcement

```cypher
// Find all functions that don't have documentation
MATCH (func:code_function)
WHERE NOT EXISTS(
    (func)<-[:documents]-(doc:code_documentation)
)
RETURN func.function_name, func.namespace, func.metadata.language
```

## Implementation Strategy

1. **Each language parser** translates language-specific constructs to universal types
2. **Element taxonomy** defines universal code types
3. **Promoted fields** use language-agnostic names
4. **Metadata** stores language-specific nuances
5. **Ontologies** can query across languages using universal concepts

## Benefits

- **Cross-language analysis**: "Show me all exported functions" works for Go, Java, Python, etc.
- **Polyglot codebases**: Analyze microservices written in different languages uniformly
- **Universal policies**: "All functions must have docs" applies to all languages
- **Language migration**: Compare equivalent structures across language boundaries
- **Knowledge graphs**: Connect requirements to implementation regardless of language

## Next Steps

1. Implement parsers for high-priority languages (Go, Python, Java, JavaScript/TypeScript)
2. Validate universal abstractions with real codebases
3. Create cross-language ontologies for common patterns
4. Build policy enforcement rules using universal concepts
