# Age Go Driver

[![Go Reference](https://pkg.go.dev/badge/github.com/9ssi7/age.svg)](https://pkg.go.dev/github.com/9ssi7/age) [![Go Report Card](https://goreportcard.com/badge/github.com/9ssi7/age)](https://goreportcard.com/report/github.com/9ssi7/age) [![Apache License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

A high-performance and easy-to-use Go driver for Apache AGE. This driver is written entirely in Go without any Java dependencies and utilizes an ANTLR4-based query parser.

## Features

- **Java-Free:** Written purely in Go, eliminating external dependencies.
- **ANTLR4 Integration:** Accurately and efficiently parses Apache AGE's Cypher-like query language.
- **Type-Safe Mapping:** Easily map database objects (vertices, edges) to Go structs.
- **Transaction Support:** Provides transaction management with ACID guarantees.
- **Simple and Intuitive API:** Offers a user-friendly and straightforward API for rapid development.

## Installation

```bash
go get -u github.com/9ssi7/age
```

## Running Tests

To run the tests, you'll need a running Apache AGE instance. You can easily spin one up using Docker:

```bash
docker run \
--name age  \
-p 5455:5432 \
-e POSTGRES_USER=postgres \
-e POSTGRES_PASSWORD=postgres \
-e POSTGRES_DB=postgres \
-d \
apache/age
```

## Examples

If you want to see some examples of how to use the driver, check out the [samples](https://github.com/9ssi7/age-samples) repository.

## Usage

```go
package main

import (
    "fmt"

    "github.com/9ssi7/age"
)

// ... (Person and WorksWith structs) ...

func main() {
    // ... (connection settings) ...

    ag := age.New(age.Config{
        GraphName: graphName,
        Dsn:       dsn,
    })
    ok, err := ag.Prepare()

    // ... (query and data processing examples) ...
}
```

## Examples

- **Creating Vertices:**

```go
_, err = tx.Exec(0, "CREATE (n:Person {name: '%s', age: %d})", "Alice", 30)
```

- **Creating Edges:**

```go
_, err = tx.Exec(0, "MATCH (a:Person), (b:Person) WHERE a.name = 'Alice' AND b.name = 'Bob' CREATE (a)-[:KNOWS]->(b)")
```

- **Querying and Processing Data:**

```go
cursor, err := tx.Exec(1, "MATCH (n:Person) RETURN n")

for cursor.Next() {
    entities, _ := cursor.GetRow()
    vertex := entities[0].(*age.Vertex)

    // Mapping age.Vertex to Person struct
    person := &Person{}
    mapper.Unmarshal(vertex, person) // Or use ParseStruct for more control

    // age.ParseStruct(entities[0], person) // Alternative method

    fmt.Println(person)
}
```

### Using `ParseStruct`

```go
// Assuming entities[0] is a Vertex
person := &Person{}
err := age.ParseStruct(entities[0], person)
if err != nil {
    // Handle error
}
```

## Contributing

Contributions are welcome via pull requests. Bug fixes, new features, and improvements are always appreciated. Please read CONTRIBUTING.md before contributing.

## License

Apache License 2.0