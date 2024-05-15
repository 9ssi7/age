package age

import (
	"bytes"
	"database/sql"
	"fmt"
	"reflect"

	_ "github.com/lib/pq"
)

type Tx interface {
	Commit() error
	Rollback() error
	Exec(columnCount int, cypher string, args ...interface{}) (*CypherCursor, error)
	ExecMap(columnCount int, cypher string, args ...interface{}) (*CypherMapCursor, error)
}

type Driver interface {
	Prepare() (bool, error)
	Close() error
	DB() *sql.DB
	Begin() (Tx, error)
}

type CursorProvider func(columnCount int, rows *sql.Rows) Cursor

type Cursor interface {
	Next() bool
	Close() error
}

func execCypher(cursorProvider CursorProvider, tx *sql.Tx, graphName string, columnCount int, cypher string, args ...interface{}) (Cursor, error) {
	var buf bytes.Buffer

	cypherStmt := fmt.Sprintf(cypher, args...)

	buf.WriteString("SELECT * from cypher(NULL,NULL) as (v0 agtype")

	for i := 1; i < columnCount; i++ {
		buf.WriteString(fmt.Sprintf(", v%d agtype", i))
	}
	buf.WriteString(");")

	stmt := buf.String()

	// Pass in the graph name and cypher statement via parameters to prepare
	// the cypher function call for session info.

	prepare_stmt := "SELECT * FROM age_prepare_cypher($1, $2);"
	_, perr := tx.Exec(prepare_stmt, graphName, cypherStmt)
	if perr != nil {
		fmt.Println(prepare_stmt + " " + graphName + " " + cypher)
		return nil, perr
	}

	if columnCount == 0 {
		_, err := tx.Exec(stmt)
		if err != nil {
			fmt.Println(stmt)
			return nil, err
		}
		return nil, nil
	} else {
		rows, err := tx.Query(stmt)
		if err != nil {
			fmt.Println(stmt)
			return nil, err
		}
		return cursorProvider(columnCount, rows), nil
	}
}

// Exec : execute cypher query
// CREATE , DROP ....
// MATCH .... RETURN ....
// CREATE , DROP .... RETURN ...
func exec(tx *sql.Tx, graphName string, columnCount int, cypher string, args ...interface{}) (*CypherCursor, error) {
	cursor, err := execCypher(NewCypherCursor, tx, graphName, columnCount, cypher, args...)
	var cypherCursor *CypherCursor
	if cursor != nil {
		cypherCursor = cursor.(*CypherCursor)
	}
	return cypherCursor, err
}

// ExecMap
// CREATE , DROP ....
// MATCH .... RETURN ....
// CREATE , DROP .... RETURN ...
func execMap(tx *sql.Tx, graphName string, columnCount int, cypher string, args ...interface{}) (*CypherMapCursor, error) {
	cursor, err := execCypher(NewCypherMapCursor, tx, graphName, columnCount, cypher, args...)
	var cypherMapCursor *CypherMapCursor
	if cursor != nil {
		cypherMapCursor = cursor.(*CypherMapCursor)
	}
	return cypherMapCursor, err
}

type age struct {
	db        *sql.DB
	graphName string
}

type ageTx struct {
	age *age
	tx  *sql.Tx
}

type Config struct {
	GraphName string
	Db        *sql.DB
	Dsn       string
}

func New(cnf Config) Driver {
	if cnf.Db == nil && cnf.Dsn != "" {
		db, err := sql.Open("postgres", cnf.Dsn)
		if err != nil {
			panic(err)
		}
		cnf.Db = db
	}
	if cnf.Db == nil {
		panic("age: Database connection is required")
	}
	if cnf.GraphName == "" {
		panic("age: Graph name is required")
	}
	return &age{db: cnf.Db, graphName: cnf.GraphName}
}

func (age *age) Prepare() (bool, error) {
	tx, err := age.db.Begin()
	if err != nil {
		return false, err
	}

	_, err = tx.Exec("LOAD 'age';")
	if err != nil {
		return false, err
	}

	_, err = tx.Exec("SET search_path = ag_catalog, '$user', public;")
	if err != nil {
		return false, err
	}

	var count int = 0

	err = tx.QueryRow("SELECT count(*) FROM ag_graph WHERE name=$1", age.graphName).Scan(&count)

	if err != nil {
		return false, err
	}

	if count == 0 {
		_, err = tx.Exec("SELECT create_graph($1);", age.graphName)
		if err != nil {
			return false, err
		}
	}

	tx.Commit()

	return true, nil
}

func (a *age) Close() error {
	return a.db.Close()
}

func (a *age) DB() *sql.DB {
	return a.db
}

func (a *age) Begin() (Tx, error) {
	ageTx := &ageTx{age: a}
	tx, err := a.db.Begin()
	if err != nil {
		return nil, err
	}
	ageTx.tx = tx
	return ageTx, err
}

func (t *ageTx) Commit() error {
	return t.tx.Commit()
}

func (t *ageTx) Rollback() error {
	return t.tx.Rollback()
}

/** CREATE , DROP .... */
func (a *ageTx) Exec(columnCount int, cypher string, args ...interface{}) (*CypherCursor, error) {
	return exec(a.tx, a.age.graphName, columnCount, cypher, args...)
}

func (a *ageTx) ExecMap(columnCount int, cypher string, args ...interface{}) (*CypherMapCursor, error) {
	return execMap(a.tx, a.age.graphName, columnCount, cypher, args...)
}

type CypherCursor struct {
	Cursor
	columnCount int
	rows        *sql.Rows
	unmarshaler Unmarshaller
}

func NewCypherCursor(columnCount int, rows *sql.Rows) Cursor {
	return &CypherCursor{columnCount: columnCount, rows: rows, unmarshaler: NewAGUnmarshaler()}
}

func (c *CypherCursor) Next() bool {
	return c.rows.Next()
}

func (c *CypherCursor) GetRow() ([]Entity, error) {
	var gstrs = make([]interface{}, c.columnCount)
	for i := 0; i < c.columnCount; i++ {
		gstrs[i] = new(string)
	}

	err := c.rows.Scan(gstrs...)
	if err != nil {
		return nil, fmt.Errorf("CypherCursor.GetRow:: %s", err)
	}

	entArr := make([]Entity, c.columnCount)
	for i := 0; i < c.columnCount; i++ {
		gstr := gstrs[i].(*string)
		e, err := c.unmarshaler.Unmarshal(*gstr)
		if err != nil {
			fmt.Println(i, ">>", gstr)
			return nil, err
		}
		entArr[i] = e
	}

	return entArr, nil
}

func (c *CypherCursor) Close() error {
	return c.rows.Close()
}

type CypherMapCursor struct {
	CypherCursor
	mapper *Mapper
}

func NewCypherMapCursor(columnCount int, rows *sql.Rows) Cursor {
	mapper := NewMapper(make(map[string]reflect.Type))
	pcursor := CypherCursor{columnCount: columnCount, rows: rows, unmarshaler: mapper}
	return &CypherMapCursor{CypherCursor: pcursor, mapper: mapper}
}

func (c *CypherMapCursor) PutType(label string, tp reflect.Type) {
	c.mapper.PutType(label, tp)
}

func (c *CypherMapCursor) GetRow() ([]interface{}, error) {
	entities, err := c.CypherCursor.GetRow()

	if err != nil {
		return nil, fmt.Errorf("CypherMapCursor.GetRow:: %s", err)
	}

	elArr := make([]interface{}, c.columnCount)

	for i := 0; i < c.columnCount; i++ {
		ent := entities[i]
		if ent.GType() == G_MAP_PATH {
			elArr[i] = ent
		} else {
			elArr[i] = ent.(*SimpleEntity).Value()
		}
	}

	return elArr, nil
}
