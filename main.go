package main

import (
	"os"
	"strconv"
)

func main() {
	api_addr, present := os.LookupEnv("C2_SERVER_API_ADDR")
	if !present {
		api_addr = "0.0.0.0"
	}
	api_port_s, present := os.LookupEnv("C2_SERVER_API_PORT")
	if !present {
		api_port_s = "8080"
	}
	api_port_u64, err := strconv.ParseUint(api_port_s, 10, 16)
	if err != nil {
		panic(err)
	}
	api_port := uint16(api_port_u64)

	db_addr, present := os.LookupEnv("C2_SERVER_DB_ADDR")
	if !present {
		db_addr = "0.0.0.0"
	}
	db_port_s, present := os.LookupEnv("C2_SERVER_DB_PORT")
	if !present {
		db_port_s = "3306"
	}
	db_port_u64, err := strconv.ParseUint(db_port_s, 10, 16)
	if err != nil {
		panic(err)
	}
	db_port := uint16(db_port_u64)

	s := NewAPIServer(api_addr, api_port, db_addr, db_port)
	s.Run()
}
