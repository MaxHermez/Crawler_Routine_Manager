package main

import (
	"MongoHandles"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Routine struct {
	Hub     string `json:"Hub"`
	Script  string `json:"Script"`
	Command string `json:"Command"`
	Start   string `json:"Start"`
	Freq    string `json:"Freq"`
	Active  string `json:"Active,omitempty"`
	ID      string `json:"ID, omitempty"`
}

func main() {
	http.HandleFunc("/routines", routineHandler)
	log.Fatal(http.ListenAndServe("localhost:8080", nil))
}

func getRoutine(row interface{}) (Routine, error) {
	s, ok := row.(bson.D)
	if !ok {
		return Routine{}, errors.New("Type error: failed to parse row to bson.D")
	}
	m := s.Map()
	log.Printf("%v", m)
	hub, ok := m["Hub"].(string)
	if !ok {
		return Routine{}, errors.New("Type error: while analyzing row 'Hub'")
	}
	script, ok := m["Script"].(string)
	if !ok {
		return Routine{}, errors.New("Type error: while analyzing row 'Script'")
	}
	command, ok := m["Command"].(string)
	if !ok {
		return Routine{}, errors.New("Type error: while analyzing row 'Command'")
	}
	start, ok := m["Start"].(string)
	if !ok {
		return Routine{}, errors.New("Type error: while analyzing row 'Start'")
	}
	freq, ok := m["Freq"].(string)
	if !ok {
		return Routine{}, errors.New("Type error: while analyzing row 'Freq'")
	}
	active, ok := m["Freq"].(string)
	if !ok {
		active = ""
	}
	ID, ok := m["_id"].(primitive.ObjectID)
	if !ok {
		ID = primitive.ObjectID{}
	}
	return Routine{hub, script, command, start, freq, active, ID.Hex()}, nil
}

func getRoutines(response []interface{}) ([]Routine, error) {
	var out []Routine
	for _, each := range response {
		rtn, err := getRoutine(each)
		if err != nil {
			return []Routine{}, err
		}
		out = append(out, rtn)
	}
	return out, nil
}

func routineHandler(rw http.ResponseWriter, req *http.Request) {
	conn, err := MongoHandles.NewConn("mongodb+srv://pappa:ohh5UMa3caBAdozq@cluster0.q8o2d.mongodb.net/Routines/?retryWrites=true&w=majority")
	if err != nil {
		log.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	defer conn.Client.Disconnect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	switch req.Method {
	case "GET":
		res, err := conn.GetCollection("Routines", "master", ctx)
		if err != nil {
			println(err)
		}
		rtns, err := getRoutines(res)
		if err != nil {
			println(err)
		}
		println(len(rtns))
	case "POST":
		buf := new(bytes.Buffer)
		buf.ReadFrom(req.Body)
		r := &Routine{}
		err := json.Unmarshal([]byte(buf.String()), r)
		if err != nil {
			println(err)
		}
		conn.InsertPost("Routines", "master", *r, ctx)
	case "PUT":
		buf := new(bytes.Buffer)
		buf.ReadFrom(req.Body)
		r := &Routine{}
		err := json.Unmarshal([]byte(buf.String()), r)
		if err != nil {
			log.Fatalf("%v", err)
		}
		ID, _ := primitive.ObjectIDFromHex(r.ID)
		err = conn.ReplaceEntry("Routines", "master",
			bson.D{{Key: "_id", Value: ID}},
			bson.D{{Key: "Hub", Value: r.Hub}, {Key: "Script", Value: r.Script},
				{Key: "Command", Value: r.Command}, {Key: "Start", Value: r.Start},
				{Key: "Freq", Value: r.Freq}, {Key: "Active", Value: r.Active}}, ctx)
		if err != nil {
			log.Fatalf("%v", err)
		}
	case "DELETE":
		buf := new(bytes.Buffer)
		buf.ReadFrom(req.Body)
		r := &Routine{}
		err := json.Unmarshal([]byte(buf.String()), r)
		if err != nil {
			log.Fatalf("%v", err)
		}
		ID, _ := primitive.ObjectIDFromHex(r.ID)
		err = conn.DeleteOne("Routines", "master",
			bson.D{{Key: "_id", Value: ID}}, ctx)
		if err != nil {
			log.Fatalf("%v", err)
		}
	}
}
