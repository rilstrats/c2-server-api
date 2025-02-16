package main

import (
	"database/sql"
	"log"
	"os"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
)

func GetNewDBServer() *sql.DB {
	user := GetenvOrDefault("C2_DB_USER", "c2")
	addr := GetenvOrDefault("C2_DB_ADDR", "db:3306")
	DBName := GetenvOrDefault("C2_DB_NAME", "c2")
	pwFile := GetenvOrDefault("C2_DB_PW_FILE", "secrets/db_password.txt")
	pwB, err := os.ReadFile(pwFile)
	if err != nil {
		log.Fatal(err)
	}
	passwd := strings.TrimSpace(string(pwB))

	cfg := mysql.Config{
		Addr:                 addr,
		User:                 user,
		Passwd:               passwd,
		DBName:               DBName,
		AllowNativePasswords: true,
	}

	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		log.Fatal(err)
	}

	pingErr := db.Ping()
	pingErrCount := 0
	for pingErr != nil {
		pingErrCount++
		time.Sleep(time.Second * 1)
		if pingErrCount >= 10 {
			log.Fatal(pingErr)
		}
	}

	return db
}
