package main

import (
	"fmt"
	"github.com/chassepatate/go-sse"
	"github.com/sirupsen/logrus"
	"log"
	"net/http"
	"strconv"
	"time"
)

type API struct {
	server *sse.Server
}

func main() {
	logrus.SetLevel(logrus.DebugLevel)
	sseServer := sse.NewServer()
	sseServer.SetCustomHeaders(map[string]string{
		"Access-Control-Allow-Origin": "*",
	})
	sseServer.SetHeartBeatInterval(5 * time.Second)

	api := &API{server: sseServer}

	http.HandleFunc("/sse", api.sseHandler)

	go func() {
		count := 0
		tick := time.Tick(2 * time.Second)
		for {
			select {
			case <-tick:
				count++
				api.server.Broadcast(sse.Event{
					Id:    fmt.Sprintf("event-id-%v", count),
					Event: "message",
					Data:  "test " + strconv.Itoa(count),
				})
			}
		}
	}()

	log.Fatal(http.ListenAndServe(":8080", http.DefaultServeMux))
}

func (api *API) sseHandler(writer http.ResponseWriter, request *http.Request) {
	connection, err := api.server.NewConnection(writer, request)
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
