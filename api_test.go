package main

import (
	"testing"
	"time"
)

func TestIntegration(t *testing.T) {
	s := GetNewAPIServer()

	beaconCreate := Beacon{
		IP:       "10.10.10.10",
		Hostname: "test-random",
	}

	id, err := s.RegisterBeacon(beaconCreate)
	if err != nil {
		t.Errorf("Register(beacon) failed: %s", err.Error())
		return
	}

	exists, err := s.CheckBeaconIDExistence(id)
	if err != nil {
		t.Errorf("CheckBeaconIDExistence(%d) failed: %s", id, err.Error())
		return
	}
	if !exists {
		t.Errorf("CheckBeaconIDExistence(%d) failed to find existing beacon", id)
		return
	}

	var fakeID int32 = -1
	exists, err = s.CheckBeaconIDExistence(fakeID)
	if err != nil {
		t.Errorf("CheckBeaconIDExistence(%d) failed: %s", fakeID, err.Error())
		return
	}
	if exists {
		t.Errorf("CheckBeaconIDExistence(%d) found an ID that doesn't exist", fakeID)
		return
	}

	beaconDB, err := s.GetBeacon(id)
	if err != nil {
		t.Errorf("GetBeacon(%d) failed: %s", id, err.Error())
		return
	}

	if beaconCreate.IP != beaconDB.IP || beaconCreate.Hostname != beaconDB.Hostname {
		t.Errorf("registered beacon != returned beacon")
		return
	}

	commandWebshellCreate := Command{BeaconID: id, Type: "webshell"}
	err = s.RegisterCommand(commandWebshellCreate)
	if err != nil {
		t.Errorf("RegisterCommand(webshell) failed: %s", err.Error())
		return
	}
	time.Sleep(time.Millisecond * 100)
	commandRevshellCreate := Command{BeaconID: id, Type: "revshell", Arg: "10.0.0.1"}
	err = s.RegisterCommand(commandRevshellCreate)
	if err != nil {
		t.Errorf("RegisterCommand(revshell) failed: %s", err.Error())
		return
	}
	time.Sleep(time.Millisecond * 100)
	commandRunCreate := Command{BeaconID: id, Type: "run", Arg: "whoami"}
	err = s.RegisterCommand(commandRunCreate)
	if err != nil {
		t.Errorf("RegisterCommand(run) failed: %s", err.Error())
		return
	}
	coms, err := s.GetCommands(id, false)
	if err != nil {
		t.Errorf("GetCommands(%d) failed: %s", id, err.Error())
		return
	}
	if coms[0].Type != "webshell" || coms[1].Type != "revshell" || coms[2].Type != "run" {
		t.Errorf("Commands weren't returned in chronological order")
		return
	}

	// err = s.MarkCommandsExecuted(id)
	// if err != nil {
	// 	t.Errorf("MarkCommandsExecuted(%d) failed: %s", id, err.Error())
	// 	return
	// }
	err = s.MarkCommandExecuted(coms[0].ID, "revshell created")
	err = s.MarkCommandExecuted(coms[1].ID, "webshell created")
	err = s.MarkCommandExecuted(coms[2].ID, "whoami executed")

	coms, err = s.GetCommands(id, false)
	if err != nil {
		t.Errorf("GetCommands(%d) failed: %s", id, err.Error())
		return
	}
	if !coms[0].Executed || !coms[1].Executed || !coms[2].Executed {
		t.Errorf("Commands weren't marked as executed")
		return
	}

	beaconFinal, err := s.GetBeacon(id)
	if err != nil {
		t.Errorf("GetBeacon(%d) failed: %s", id, err.Error())
		return
	}
	if len(beaconFinal.Commands) != 3 {
		t.Errorf("Commands weren't added to beacon")
		return
	}

	err = s.DeleteBeacon(id)
	if err != nil {
		t.Errorf("DeleteBeacon(%d) failed: %s", id, err.Error())
		return
	}

	exists, err = s.CheckBeaconIDExistence(id)
	if err != nil {
		t.Errorf("CheckBeaconIDExistence(%d) failed: %s", id, err.Error())
		return
	}
	if exists {
		t.Errorf("CheckBeaconIDExistence(%d) found an ID that was deleted", id)
		return
	}
}

// func TestRegister(t *testing.T) {}
//
// func TestCheckBeaconIDExistence(t *testing.T) {}
//
// func TestGetBeacon(t *testing.T) {}
//
// func TestGetCommands(t *testing.T) {}
//
// func TestMarkCommandsExecuted(t *testing.T) {}
//
// func TestDeleteBeaconByID(t *testing.T) {}
