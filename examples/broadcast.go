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
	broker *sse.Broker
}

func main() {
	logrus.SetLevel(logrus.DebugLevel)
	sseClientBroker := sse.NewBroker()
	sseClientBroker.SetCustomHeaders(map[string]string{
		"Access-Control-Allow-Origin": "*",
	})

	api := &API{broker: sseClientBroker}

	http.HandleFunc("/sse", api.sseHandler)

	// Broadcast message to all clients every 5 seconds
	go func() {
		count := 0
		tick := time.Tick(5 * time.Second)
		for {
			select {
			case <-tick:
				count++
				api.broker.Broadcast(sse.Event{
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
	connection, err := api.broker.NewConnection(writer, request)
	if err != nil {
		log.Println(err)
		return
	}

	err = connection.Open()
	if err != nil {
		panic(err)
	}

	log.Printf("connection with client closed")
}
