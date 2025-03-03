package main

import (
	"testing"
)

func TestIntegration(t *testing.T) {
	s := GetNewAPIServer()

	beaconCreate := Beacon{
		IP:       "192.168.1.1",
		Hostname: "test1",
	}

	id, err := s.Register(beaconCreate)
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
		t.Errorf("CheckBeaconIDExistence(%d) founded an ID that doesn't exist", fakeID)
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

	// s.AddCommand()

	// s.GetCommands()

	// s.MarkCommandsExecuted()

	err = s.DeleteBeacon(id)
	if err != nil {
		t.Errorf("DeleteBeacon(%d) failed: %s", id, err.Error())
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
