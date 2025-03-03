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
	addr := GetenvOrDefault("C2_DB_ADDR", "0.0.0.0:3306")
	DBName := GetenvOrDefault("C2_DB_NAME", "c2")
	pwFile := GetenvOrDefault("C2_DB_PW_FILE", "secrets/db_password.txt")
	pwB, err := os.ReadFile(pwFile)
	if err != nil {
		log.Fatalf("Failed to read %s: %s", pwFile, err)
	}
	passwd := strings.TrimSpace(string(pwB))

	cfg := mysql.NewConfig()
	cfg.Addr = addr
	cfg.User = user
	cfg.Passwd = passwd
	cfg.DBName = DBName
	// if this isn't included, addr isn't used
	cfg.Net = "tcp"

	// log.Printf("Opening DB Connection: %s", cfg.FormatDSN())
	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		log.Fatalf("Failed to open DB Connection: %s", err)
	}
	// log.Printf("Opened DB Connection: %s", cfg.FormatDSN())

	// log.Print("Pinging DB")
	pingErr := db.Ping()
	pingErrCount := 0
	for pingErr != nil {
		pingErrCount++
		time.Sleep(time.Second * 1)

		pingErr = db.Ping()

		if pingErrCount >= 10 {
			log.Fatalf("Failed to ping DB: %s", pingErr)
		}
	}
	// log.Print("Pinged DB")

	log.Printf("DB connection ready: %s", addr)
	return db
}
