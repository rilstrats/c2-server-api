package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
)

type APIServer struct {
	addr string
	db   *sql.DB
}

func GetNewAPIServer() *APIServer {
	db := GetNewDBServer()
	addr, present := os.LookupEnv("C2_API_ADDR")
	if !present {
		addr = "0.0.0.0:8080"
	}
	return &APIServer{
		addr: addr,
		db:   db,
	}
}

// func (s *APIServer) String() string {
// 	return fmt.Sprintf("%s:%d", s.addr, s.port)
// }

func (s *APIServer) Run() error {
	router := http.NewServeMux()
	router.HandleFunc("POST /register", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Registered"))
	})
	router.HandleFunc("GET /beacon/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		w.Write([]byte("Got ID: " + id))
	})
	router.HandleFunc("DELETE /beacon/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		w.Write([]byte("Deleted ID: " + id))
	})
	router.HandleFunc("GET /beacon/{id}/commands", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		w.Write([]byte("Got ID Commands: " + id))
	})
	router.HandleFunc("POST /beacon/{id}/commands", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		w.Write([]byte("Post ID Commands: " + id))
	})
	server := http.Server{
		Addr:    s.addr,
		Handler: router,
	}

	log.Printf("Server has started %s", s.addr)

	return server.ListenAndServe()
}
