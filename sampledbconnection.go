package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/denisenkom/go-mssqldb"
)

func main() {
	connString := "sqlserver://sa:Password@123@13.201.10.68?connection+timeout=30&dial+timeout=30"
	db, err := sql.Open("mssql", connString)
	if err != nil {
		log.Fatal("Open connection failed:", err.Error())
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		log.Fatal("Ping failed:", err.Error())
	}

	fmt.Println("Successfully connected to SQL Server")

	// Query to get data from the system table sys.databases
	query := "SELECT name, database_id FROM sys.databases"
	rows, err := db.Query(query)
	if err != nil {
		log.Fatal("Query failed:", err.Error())
	}
	defer rows.Close()

	// Iterate through the result set
	for rows.Next() {
		var name string
		var database_id int
		err = rows.Scan(&name, &database_id)
		if err != nil {
			log.Fatal("Scan failed:", err.Error())
		}
		log.Printf("Database: name=%s, database_id=%d", name, database_id)
	}

	// Check for errors from iterating over rows.
	err = rows.Err()
	if err != nil {
		log.Fatal("Rows iteration error:", err.Error())
	}
}
