package main

import (
	"fmt"
	"log"
	"net/http"
	// "net/netip"
)

type APIServer struct {
	addr   string
	port   uint16
	dbAddr string
	dbPort uint16
}

func NewAPIServer(addr string, port uint16, dbAddr string, dbPort uint16) *APIServer {
	return &APIServer{
		addr:   addr,
		port:   port,
		dbAddr: dbAddr,
		dbPort: dbPort,
	}
}

func (s *APIServer) String() string {
	return fmt.Sprintf("%s:%d", s.addr, s.port)
}

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
		Addr:    s.String(),
		Handler: router,
	}

	log.Printf("Server has started %s", s.String())

	return server.ListenAndServe()
}
