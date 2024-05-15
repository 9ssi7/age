package age

import (
	"fmt"
	"testing"

	_ "github.com/lib/pq"
)

var dsn string = "host=127.0.0.1 port=5455 dbname=postgres user=postgres password=postgres sslmode=disable"
var graphName string = "testGraph"

func Test(t *testing.T) {
	ag := New(Config{
		GraphName: graphName,
		Dsn:       dsn,
	})
	_, err := ag.Prepare()

	if err != nil {
		t.Fatal(err)
	}

	tx, err := ag.Begin()
	if err != nil {
		t.Fatal(err)
	}

	_, err = tx.Exec(0, "CREATE (n:Person {name: '%s'})", "Joe")
	if err != nil {
		t.Fatal(err)
	}

	_, err = tx.Exec(0, "CREATE (n:Person {name: '%s', age: %d})", "Smith", 10)
	if err != nil {
		t.Fatal(err)
	}

	_, err = tx.Exec(0, "CREATE (n:Person {name: '%s', weight:%f})", "Jack", 70.3)
	if err != nil {
		t.Fatal(err)
	}

	tx.Commit()

	tx, err = ag.Begin()
	if err != nil {
		t.Fatal(err)
	}

	cursor, err := tx.Exec(1, "MATCH (n:Person) RETURN n")
	if err != nil {
		t.Fatal(err)
	}

	count := 0
	for cursor.Next() {
		entities, err := cursor.GetRow()
		if err != nil {
			t.Fatal(err)
		}
		count++
		vertex := entities[0].(*Vertex)
		fmt.Println(count, "]", vertex.Id(), vertex.Label(), vertex.Props())
	}

	fmt.Println("Vertex Count:", count)

	_, err = tx.Exec(0, "MATCH (a:Person), (b:Person) WHERE a.name='%s' AND b.name='%s' CREATE (a)-[r:workWith {weight: %d}]->(b)",
		"Jack", "Joe", 3)
	if err != nil {
		t.Fatal(err)
	}

	_, err = tx.Exec(0, "MATCH (a:Person {name: '%s'}), (b:Person {name: '%s'}) CREATE (a)-[r:workWith {weight: %d}]->(b)",
		"Joe", "Smith", 7)
	if err != nil {
		t.Fatal(err)
	}

	tx.Commit()

	tx, err = ag.Begin()
	if err != nil {
		t.Fatal(err)
	}

	cursor, err = tx.Exec(1, "MATCH p=()-[:workWith]-() RETURN p")
	if err != nil {
		t.Fatal(err)
	}

	count = 0
	for cursor.Next() {
		entities, err := cursor.GetRow()
		if err != nil {
			t.Fatal(err)
		}
		count++
		path := entities[0].(*Path)

		fmt.Println(count, "]", path.GetAsVertex(0), path.GetAsEdge(1).props, path.GetAsVertex(2))
	}

	_, err = tx.Exec(0, "MATCH (n:Person) DETACH DELETE n RETURN *")
	if err != nil {
		t.Fatal(err)
	}
	tx.Commit()
}

func TestQueryManyReturn(t *testing.T) {
	ag := New(Config{
		GraphName: graphName,
		Dsn:       dsn,
	})

	_, err := ag.Prepare()

	if err != nil {
		t.Fatal(err)
	}

	tx, err := ag.Begin()
	if err != nil {
		t.Fatal(err)
	}

	// Create Vertex
	tx.Exec(0, "CREATE (n:Person {name: '%s'})", "Joe")
	tx.Exec(0, "CREATE (n:Person {name: '%s', age: %d})", "Smith", 10)
	tx.Exec(0, "CREATE (n:Person {name: '%s', weight:%f})", "Jack", 70.3)

	tx.Commit()

	tx, err = ag.Begin()
	if err != nil {
		t.Fatal(err)
	}

	// Create Path
	tx.Exec(0, "MATCH (a:Person), (b:Person) WHERE a.name='%s' AND b.name='%s' CREATE (a)-[r:workWith {weight: %d}]->(b)",
		"Jack", "Joe", 3)

	tx.Exec(0, "MATCH (a:Person {name: '%s'}), (b:Person {name: '%s'}) CREATE (a)-[r:workWith {weight: %d}]->(b)",
		"Joe", "Smith", 7)

	tx.Commit()

	tx, err = ag.Begin()
	if err != nil {
		t.Fatal(err)
	}

	// Query Path1
	cursor, err := tx.Exec(3, "MATCH (a:Person)-[l:workWith]-(b:Person) RETURN a, l, b")
	if err != nil {
		t.Fatal(err)
	}

	count := 0
	for cursor.Next() {
		entities, err := cursor.GetRow()
		if err != nil {
			t.Fatal(err)
		}
		count++
		v1 := entities[0].(*Vertex)
		edge := entities[1].(*Edge)
		v2 := entities[2].(*Vertex)
		fmt.Println("ROW ", count, ">>", "\n\t", v1, "\n\t", edge, "\n\t", v2)
	}

	// Query Path2
	cursor, err = tx.Exec(1, "MATCH p=(a:Person)-[l:workWith]-(b:Person) WHERE a.name = '%s' RETURN p", "Joe")
	if err != nil {
		t.Fatal(err)
	}

	count = 0
	for cursor.Next() {
		entities, err := cursor.GetRow()
		if err != nil {
			t.Fatal(err)
		}
		count++
		path := entities[0].(*Path)
		fmt.Println("ROW ", count, ">>", "\n\t", path.GetAsVertex(0),
			"\n\t", path.GetAsEdge(1),
			"\n\t", path.GetAsVertex(2))
	}

	// Clear Data
	_, err = tx.Exec(0, "MATCH (n:Person) DETACH DELETE n RETURN *")
	if err != nil {
		t.Fatal(err)
	}
	tx.Commit()
}
