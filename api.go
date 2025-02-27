package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
)

type Beacon struct {
	IP       string `json:"ip" sql:"ip"`
	Hostname string `json:"hostname" sql:"hostname"`
	Commands []Command
}

type Command struct {
	Type         string `json:"type" sql:"c_type"`
	Arg          string `json:"arg" sql:"c_arg"`
	SeenByBeacon bool   `json:"seen_by_beacon" sql:"seen_by_beacon"`
}

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

func (s *APIServer) CheckBeaconIDExistence(id int32) (bool, error) {
	dbID := id

	row := s.db.QueryRow("SELECT id FROM beacons WHERE id = ?;", id)
	err := row.Scan(&dbID)

	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if id == dbID {
		return true, nil
	}
	return false, nil
}

func (s *APIServer) GenerateUniqueBeaconID() (int32, error) {
	var id int32
	for {
		id = rand.Int31()
		inDB, err := s.CheckBeaconIDExistence(id)
		if err != nil {
			return id, err
		}
		if !inDB {
			break
		}
	}
	return id, nil
}

func (s *APIServer) Register(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")

	defer r.Body.Close()
	rBody, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, `{"message": "no request body"}`, http.StatusInternalServerError)
		return
	}

	beaconData := Beacon{}
	err = json.Unmarshal(rBody, &beaconData)
	if err != nil {
		http.Error(w, fmt.Sprintf(
			`{"message": "%s"}`, err.Error()),
			http.StatusInternalServerError)
		return
	}

	id, err := s.GenerateUniqueBeaconID()
	if err != nil {
		http.Error(w, fmt.Sprintf(
			`{"message": "%s"}`, err.Error()),
			http.StatusInternalServerError)
		return
	}

	_, err = s.db.Exec(
		"INSERT INTO beacons VALUES (?, ?, ?)",
		id,
		beaconData.IP,
		beaconData.Hostname,
	)
	if err != nil {
		http.Error(w, fmt.Sprintf(
			`{"message": "%s"}`, err.Error()),
			http.StatusInternalServerError)
		return
	}

	w.Write(fmt.Appendf(nil,
		`{"message": "successfully registered",
		"id": %d}`,
		id))
}

func (s *APIServer) GetBeaconByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	w.Header().Add("Content-Type", "application/json")

	row := s.db.QueryRow(
		`SELECT ip, hostname 
		FROM beacons
		WHERE id = ?;`,
		id,
	)
	beacon := Beacon{}
	err := row.Scan(&beacon.IP, &beacon.Hostname)
	if err != nil {
		http.Error(w, fmt.Sprintf(
			`{"message": "beacon failure: %s"}`, err.Error()),
			http.StatusInternalServerError)
		return
	}

	rows, err := s.db.Query(
		`SELECT c_type, c_arg, seen_by_beacon
		FROM commands
		WHERE beacon_id = ?
		ORDER BY create_time;`,
		id,
	)
	if err != nil {
		http.Error(w, fmt.Sprintf(
			`{"message": "%s"}`, err.Error()),
			http.StatusInternalServerError)
		return
	}
	for {
		anotherRow := rows.Next()
		if !anotherRow && rows.Err() != nil {
			http.Error(w, fmt.Sprintf(
				`{"message": "command failure: %s"}`, rows.Err().Error()),
				http.StatusInternalServerError)
			return

		} else if !anotherRow {
			break
		}
		c := Command{}
		rows.Scan(&c.Type, &c.Arg, &c.SeenByBeacon)
		beacon.Commands = append(beacon.Commands, c)
	}

	body, err := json.Marshal(beacon)
	if err != nil {
		http.Error(w, fmt.Sprintf(
			`{"message": "%s"}`, err.Error()),
			http.StatusInternalServerError)
		return
	}
	w.Write(body)
}

func (s *APIServer) DeleteBeaconByID(id int32) error {
	ctx := context.TODO()

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return err
	}

	_, execErr := tx.Exec(`DELETE FROM beacons WHERE id = ?`, id)
	if execErr != nil {
		tx.Rollback()
	}

	_, execErr = tx.Exec(`DELETE FROM commands WHERE beacon_id = ?`, id)
	if execErr != nil {
		tx.Rollback()
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (s *APIServer) Run() error {
	router := http.NewServeMux()
	router.HandleFunc("POST /register", s.Register)
	router.HandleFunc("GET /beacon/{id}", s.GetBeaconByID)
	router.HandleFunc("DELETE /beacon/{id}", func(w http.ResponseWriter, r *http.Request) {
		id64, err := strconv.ParseInt(r.PathValue("id"), 10, 32)
		if err != nil {
			http.Error(w, fmt.Sprintf(
				`{"message": "%s"}`, err.Error()),
				http.StatusInternalServerError)
			return
		}

		id := int32(id64)
		err = s.DeleteBeaconByID(id)
		if err != nil {
			http.Error(w, fmt.Sprintf(
				`{"message": "%s"}`, err.Error()),
				http.StatusInternalServerError)
			return
		}

		w.Write(fmt.Appendf(nil,
			`{"message": "successfully deleted %d}`,
			id))
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

	log.Printf("API Server ready: %s", s.addr)

	return server.ListenAndServe()
}
