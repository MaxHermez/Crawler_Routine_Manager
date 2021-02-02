package main

import (
	"MongoHandles"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log"
	"net"
	"net/http"
	"strconv"
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
	ID      string `json:"ID,omitempty"`
}

type RoutineThread struct {
	Rtn    Routine
	Active chan bool
	ID     string
}

var ThreadObjs []RoutineThread = []RoutineThread{}

func CheckError(err error) {
	if err != nil {
		log.Printf("%v", err)
	}
}

func main() {
	http.HandleFunc("/routines", routineHandler)
	log.Fatal(http.ListenAndServe("localhost:8080", nil))
}

func schedule(rtns []Routine) {
	for _, rtn := range rtns {
		found := false
		for i, each := range ThreadObjs {
			if each.ID == rtn.ID && rtn.Active == "0" {
				found = true
				if rtn.Active == "0" {
					each.Active <- false
					ThreadObjs[i] = ThreadObjs[len(ThreadObjs)-1]
					ThreadObjs[len(ThreadObjs)-1] = RoutineThread{}
					ThreadObjs = ThreadObjs[:len(ThreadObjs)-1]
				}
			}
		}
		if !found {
			c := make(chan bool)
			x := RoutineThread{rtn, c, rtn.ID}
			go runRoutine(rtn, c)
			ThreadObjs = append(ThreadObjs, x)
			log.Println("added routine")
		}
	}
	// remove deleted routines
	for i, each := range ThreadObjs {
		found := false
		for _, rtn := range rtns {
			if rtn.ID == each.ID {
				found = true
			}
		}
		if !found {
			each.Active <- false
			ThreadObjs[i] = ThreadObjs[len(ThreadObjs)-1]
			ThreadObjs[len(ThreadObjs)-1] = RoutineThread{}
			ThreadObjs = ThreadObjs[:len(ThreadObjs)-1]
		}
	}
}

func untilNextTrigger(duration time.Duration, c chan bool) {
	log.Println("started waiting")
	time.Sleep(duration)
	c <- true
}

func runRoutine(r Routine, activeC chan bool) {
	startInt, err := strconv.ParseInt(r.Start, 10, 64)
	CheckError(err)
	start := time.Unix(startInt, 0)
	freqInt, err := strconv.ParseInt(r.Freq, 10, 64)
	CheckError(err)
	sDelta := start.Sub(time.Now())
	delta := sDelta + (time.Duration(freqInt) * time.Second)
	for delta > time.Duration(time.Second) {
		run := make(chan bool)
		go untilNextTrigger(delta, run)
		select {
		case <-activeC:
			log.Println("thread inactive")
			delta = time.Duration(time.Second * 0)
		case <-run:
			ServerAddr, err := net.ResolveUDPAddr("udp", "127.0.0.11:32323")
			CheckError(err)
			Conn, err := net.DialUDP("udp", nil, ServerAddr)
			CheckError(err)
			defer Conn.Close()
			ts := strconv.Itoa(int(time.Now().Unix()))
			s := r.Hub + "<>" + ts + "<>" + r.Script + "<>" + r.Command
			buf := []byte(s)
			_, err = Conn.Write(buf)
			CheckError(err)
			log.Println("routine checkpoint")
			delta = time.Duration(freqInt) * time.Second
		}
	}
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
	CheckError(err)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	defer conn.Client.Disconnect(ctx)
	CheckError(err)
	res, err := conn.GetCollection("Routines", "master", ctx)
	CheckError(err)
	rtns, err := getRoutines(res)
	CheckError(err)
	schedule(rtns)
	switch req.Method {
	case "GET":
		res, err := conn.GetCollection("Routines", "master", ctx)
		CheckError(err)
		rtns, err = getRoutines(res)
		CheckError(err)
	case "POST":
		buf := new(bytes.Buffer)
		buf.ReadFrom(req.Body)
		r := &Routine{}
		err := json.Unmarshal([]byte(buf.String()), r)
		CheckError(err)
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
