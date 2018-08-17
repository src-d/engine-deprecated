package main

import (
	"database/sql"
	"fmt"
	"log"

	mysql "github.com/go-sql-driver/mysql"
)

func main() {
	cfg := mysql.Config{
		User:                 "root",
		Addr:                 "127.0.0.1",
		AllowNativePasswords: true,
		MaxAllowedPacket:     32 * (2 << 10),
	}
	fmt.Println(cfg.FormatDSN())
	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		log.Fatal(err)
	}
	rows, err := db.Query("select count(*) from repositories;")
	if err != nil {
		log.Fatal(err)
	}
	columns, err := rows.Columns()
	if err != nil {
		log.Fatal(err)
	}
	for _, row := range columns {
		fmt.Println(row)
	}
}
