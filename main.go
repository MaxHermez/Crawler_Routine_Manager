package main

import (
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
	Hub     string             `json:"Hub"`
	Script  string             `json:"Script"`
	Command string             `json:"Command"`
	Start   string             `json:"Start"`
	Freq    string             `json:"Freq"`
	Active  bool               `json:"Active"`
	ID      primitive.ObjectID `bson:"_id,omitempty"`
}

type RoutineThread struct {
	Rtn    Routine
	Active chan bool
	ID     primitive.ObjectID
}

type JsonResponse struct {
	Code int       `json:"Code"`
	Data []Routine `json:"Data"`
}

const shards = "fwmaster-shard-00-00.5cnit.mongodb.net:27017,fwmaster-shard-00-01.5cnit.mongodb.net:27017,fwmaster-shard-00-02.5cnit.mongodb.net:27017"

var rtns []Routine
var ThreadObjs []RoutineThread = []RoutineThread{}

func CheckError(err error) {
	if err != nil {
		log.Printf("%v", err)
	}
}

func main() {
	http.HandleFunc("/routines", routineHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func init() {
	conn, err := NewConn(shards)
	CheckError(err)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	defer conn.Client.Disconnect(ctx)
	CheckError(err)
	res, err := conn.GetCollection("Dashboard", "Routines", ctx)
	CheckError(err)
	rtns, err = getRoutines(res)
	CheckError(err)
	schedule()
	log.Println("finished init")
}

func schedule() {
	// delete old threads
	for _, each := range ThreadObjs {
		each.Active <- false
	}
	ThreadObjs = []RoutineThread{}
	for _, rtn := range rtns {
		// found := false
		// for i, each := range ThreadObjs {
			// check if the routine is already running
			// if each.ID == rtn.ID {
				// found = true
				// if !rtn.Active {
				// 	each.Active <- false
				// 	ThreadObjs[i] = ThreadObjs[len(ThreadObjs)-1]
				// 	ThreadObjs[len(ThreadObjs)-1] = RoutineThread{}
				// 	ThreadObjs = ThreadObjs[:len(ThreadObjs)-1]
				// }
			// }
		// }
		// add missing routines
		// if !found && rtn.Active {
		if rtn.Active {
			c := make(chan bool)
			x := RoutineThread{rtn, c, rtn.ID}
			go runRoutine(rtn, c)
			ThreadObjs = append(ThreadObjs, x)
			log.Printf("added routine, %s\n", rtn.Hub)
		}
	}
	// remove deleted routines
	// for i, each := range ThreadObjs {
	// 	found := false
	// 	for _, rtn := range rtns {
	// 		if rtn.ID == each.ID {
	// 			found = true
	// 		}
	// 	}
	// 	if !found {
	// 		each.Active <- false
	// 		ThreadObjs[i] = ThreadObjs[len(ThreadObjs)-1]
	// 		ThreadObjs[len(ThreadObjs)-1] = RoutineThread{}
	// 		ThreadObjs = ThreadObjs[:len(ThreadObjs)-1]
	// 	}
	// }
}

func untilNextTrigger(duration time.Duration, c chan bool) {
	log.Println("started waiting until " + time.Now().Add(duration).Format("Mon Jan 2 15:04:05 2006"))
	time.Sleep(duration)
	c <- true
}

func runRoutine(r Routine, activeC chan bool) {
	startInt, err := strconv.ParseInt(r.Start, 10, 64)
	CheckError(err)
	start := time.Unix(startInt, 0)
	freqInt, err := strconv.ParseInt(r.Freq, 10, 64)
	freqInt = freqInt * 3600 * 24 // the frequency is inputted in days, we need the int to represent seconds
	CheckError(err)
	delta := time.Until(start)
	for delta < time.Duration(time.Second) {
		start = start.Add(time.Duration(freqInt) * time.Second)
		delta = time.Until(start)
	}
	running := true
	for running {
		run := make(chan bool)
		go untilNextTrigger(delta, run)
		select {
		case <-activeC:
			log.Println("thread inactive")
			running = false
		case <-run:
			log.Println("Routine triggered")
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

// converts a database row into a Routine object
func getRoutine(row interface{}) (Routine, error) {
	s, ok := row.(bson.D)
	if !ok {
		return Routine{}, errors.New("type error: failed to parse row to bson.D")
	}
	m := s.Map()
	hub, ok := m["hub"].(string)
	if !ok {
		return Routine{}, errors.New("type error: while analyzing row 'Hub'")
	}
	script, ok := m["script"].(string)
	if !ok {
		return Routine{}, errors.New("type error: while analyzing row 'Script'")
	}
	command, ok := m["command"].(string)
	if !ok {
		return Routine{}, errors.New("type error: while analyzing row 'Command'")
	}
	start, ok := m["start"].(string)
	if !ok {
		return Routine{}, errors.New("type error: while analyzing row 'Start'")
	}
	freq, ok := m["freq"].(string)
	if !ok {
		return Routine{}, errors.New("type error: while analyzing row 'Freq'")
	}
	ID, ok := m["_id"].(primitive.ObjectID)
	if !ok {
		return Routine{}, errors.New("type error: while analyzing row 'Active'")
	}
	active, ok := m["active"].(bool)
	if !ok {
		active = false
	}
	return Routine{hub, script, command, start, freq, active, ID}, nil
}

// get Routine objects slice from database rows
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

func respond(rw http.ResponseWriter, rtns []Routine) {
	rw.Header().Set("Content-Type", "application/json")
	resp := JsonResponse{200, rtns}
	json.NewEncoder(rw).Encode(resp)
}

func enableCors(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
	(*w).Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	(*w).Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, token")
}

func routineHandler(rw http.ResponseWriter, req *http.Request) {
	enableCors(&rw)
	conn, err := NewConn(shards)
	CheckError(err)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	defer conn.Client.Disconnect(ctx)
	CheckError(err)
	res, err := conn.GetCollection("Dashboard", "Routines", ctx)
	CheckError(err)
	rtns, err = getRoutines(res)
	CheckError(err)
	schedule()
	switch req.Method {
	case "GET":
		respond(rw, rtns)
	case "POST":
		buf := new(bytes.Buffer)
		buf.ReadFrom(req.Body)
		r := &Routine{}
		err := json.Unmarshal(buf.Bytes(), r)
		CheckError(err)
		conn.InsertPost("Dashboard", "Routines", *r, ctx)
		rtns = append(rtns, *r)
		schedule()
	case "PUT":
		buf := new(bytes.Buffer)
		buf.ReadFrom(req.Body)
		r := &Routine{}
		err := json.Unmarshal(buf.Bytes(), r)
		if err != nil {
			log.Fatalf("%v", err)
		}
		// ID, _ := primitive.ObjectIDFromHex(r.ID)
		log.Println(r)
		err = conn.ReplaceEntry("Dashboard", "Routines",
			bson.D{{Key: "_id", Value: r.ID}},
			bson.D{{Key: "hub", Value: r.Hub}, {Key: "script", Value: r.Script},
				{Key: "command", Value: r.Command}, {Key: "start", Value: r.Start},
				{Key: "freq", Value: r.Freq}, {Key: "active", Value: r.Active}}, ctx)
		CheckError(err)
		res, err := conn.GetCollection("Dashboard", "Routines", ctx)
		CheckError(err)
		rtns, err = getRoutines(res)
		CheckError(err)
		schedule()
		if err != nil {
			log.Fatalf("%v", err)
		}
	case "DELETE":
		buf := new(bytes.Buffer)
		buf.ReadFrom(req.Body)
		r := &Routine{}
		err := json.Unmarshal(buf.Bytes(), r)
		if err != nil {
			log.Fatalf("%v", err)
		}
		// ID, _ := primitive.ObjectIDFromHex(r.ID)
		err = conn.DeleteOne("Dashboard", "Routines",
			bson.D{{Key: "_id", Value: r.ID}}, ctx)
		if err != nil {
			log.Fatalf("%v", err)
		}
		res, err := conn.GetCollection("Dashboard", "Routines", ctx)
		CheckError(err)
		rtns, err = getRoutines(res)
		CheckError(err)
		schedule()
	}
}
