package pg

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq"
	goquery "github.com/skye-lopez/go-query"
)

type PG struct {
	Conn *sql.DB
	GQ   *goquery.GoQuery
}

func NewPg() (*PG, error) {
	db := &PG{}

	connString := fmt.Sprintf("user=%s password=%s dbname=%s port=%s sslmode=disable",
		os.Getenv("PG_USER"),
		os.Getenv("PG_PWD"),
		os.Getenv("PG_DBNAME"),
		os.Getenv("PG_PORT"),
	)

	conn, err := sql.Open("postgres", connString)
	if err != nil {
		return db, err
	}

	gq := goquery.NewGoQuery(conn)

	_, err = gq.Conn.Exec("SELECT 1 as test")
	if err != nil {
		return db, err
	}

	db.Conn = conn
	db.GQ = &gq
	return db, nil
}
