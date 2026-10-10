//go:build ignore

// Helpers added to the runtime when MySQLSupportEnabled is on.

package main

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/go-sql-driver/mysql"
)

// Public MySQL Methods
func _mysqli_connect(host, user, password, database string) *sql.DB {
	connection, err := sql.Open("mysql", user+":"+password+"@tcp("+host+":3306)/"+database+"?timeout=2s")

	if err != nil {
		fmt.Println("DB CONNECTION ERROR")
		fmt.Println(err.Error())

		__housekeeping()
		os.Exit(0)
	}

	return connection
}

func _mysqli_query(connection *sql.DB, query string) *sql.Rows {
	result, err := connection.Query(query)

	//if ( err != nil && strings.HasPrefix(err.Error(), "dial") ) {	// Only crash if database is down, not if there is a query error

	if err != nil {
		fmt.Println("DB QUERY ERROR")
		fmt.Println(err)

		__housekeeping()
		os.Exit(0)
	}

	return result

}
