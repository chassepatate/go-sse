package main

import (
	"fmt"
	"github.com/chassepatate/go-sse"
	"log"
	"net/http"
	"strconv"
	"time"
)

var server *sse.Server

func main() {
	server = sse.NewServer()

	http.HandleFunc("/", handler)

	go startEventProducer()

	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handler(writer http.ResponseWriter, request *http.Request) {
	connection, err := server.NewConnection(writer, request)
	if err != nil {
		log.Println(err)
		return
	}
	log.Printf("connecting with with new client %v", connection.ID())

	err = connection.Serve()
	if err != nil {
		log.Println(err)
		return
	}

	log.Printf("connection with client %v closed", connection.ID())
}

func startEventProducer() {
	count := 0
	for range time.Tick(2 * time.Second) {
		count++
		server.Broadcast(sse.Event{
			Id:    fmt.Sprintf("event-id-%v", count),
			Event: "message",
			Data:  "test " + strconv.Itoa(count),
		})

	}
}
