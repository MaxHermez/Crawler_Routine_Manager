package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type Routine struct {
	Hub     string `json:"hub"`
	Script  string `json:"script"`
	Command string `json:"command"`
	Start   string `json:"start"`
	Freq    string `json:"freq"`
}

func main() {
	http.HandleFunc("/routines", routineHandler)
	log.Fatal(http.ListenAndServe("localhost:8080", nil))
}

func routineHandler(rw http.ResponseWriter, req *http.Request) {
	if req.Method == "GET" {
		fmt.Println("get req")
	}
	if req.Method == "POST" {
		buf := new(bytes.Buffer)
		buf.ReadFrom(req.Body)
		// println(buf.String())
		r := &Routine{}
		// out, err := json.Marshal([]byte(buf.String()))
		// if err != nil {
		// 	println(err)
		// }
		// println(string(out))
		err := json.Unmarshal([]byte(buf.String()), r)
		if err != nil {
			println(err)
		}
		println(r.Hub)
		// decoder := json.NewDecoder(req.Body)
		// var t Routine
		// err := decoder.Decode(&t)
		// if err != nil {
		// 	fmt.Println(err)
		// } else {
		// 	println(t.doc)
		// }
	}
}
