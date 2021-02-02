package main

import (
	"MongoHandles"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

type Routine struct {
	Hub     string `json:"Hub"`
	Script  string `json:"Script"`
	Command string `json:"Command"`
	Start   string `json:"Start"`
	Freq    string `json:"Freq"`
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
	return Routine{hub, script, command, start, freq}, nil
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
	if req.Method == "GET" {
		res, err := conn.GetCollection("Routines", "master", ctx)
		if err != nil {
			println(err)
		}
		rtns, err := getRoutines(res)
		if err != nil {
			println(err)
		}
		fmt.Printf("%v", rtns)
	}
	if req.Method == "POST" {
		buf := new(bytes.Buffer)
		buf.ReadFrom(req.Body)
		r := &Routine{}
		err := json.Unmarshal([]byte(buf.String()), r)
		if err != nil {
			println(err)
		}
		conn.InsertPost("Routines", "master", *r, ctx)
	}
}
