package sse

import (
	"errors"
	"net/http"
	"sync"
)

var ErrUnknownConnection = errors.New("connection not found")

type Broker struct {
	connectionsLock sync.Mutex
	connections     map[string]*Connection

	customHeaders map[string]string

	disconnectCallback func(connectionId string)
}

func NewBroker() *Broker {
	return &Broker{
		connections:   make(map[string]*Connection),
		customHeaders: make(map[string]string),
	}
}

func (b *Broker) SetCustomHeaders(headers map[string]string) {
	b.customHeaders = headers
}

func (b *Broker) NewConnection(w http.ResponseWriter, r *http.Request) (*Connection, error) {
	return b.ConnectWithHeartBeatInterval(w, r)
}

func (b *Broker) ConnectWithHeartBeatInterval(w http.ResponseWriter, r *http.Request) (*Connection, error) {
	connection, err := newConnection(w, r)
	if err != nil {
		return nil, err
	}

	connection.onClose = func() {
		b.removeClient(connection.id)
	}

	b.setHeaders(w)

	b.addConnection(connection)

	return connection, nil
}

func (b *Broker) setHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Transfer-Encoding", "chunked")

	for k, v := range b.customHeaders {
		w.Header().Set(k, v)
	}
}

func (b *Broker) addConnection(connection *Connection) {
	b.connectionsLock.Lock()
	defer b.connectionsLock.Unlock()

	b.connections[connection.id] = connection
}

func (b *Broker) removeClient(connectionId string) {
	b.connectionsLock.Lock()
	defer b.connectionsLock.Unlock()

	delete(b.connections, connectionId)

	if b.disconnectCallback != nil {
		go b.disconnectCallback(connectionId)
	}
}

func (b *Broker) Write(connectionId string, event Event) error {
	b.connectionsLock.Lock()
	defer b.connectionsLock.Unlock()

	if _, exists := b.connections[connectionId]; !exists {
		return ErrUnknownConnection
	}

	b.connections[connectionId].Write(event)

	return nil
}

func (b *Broker) Broadcast(event Event) {
	b.connectionsLock.Lock()
	defer b.connectionsLock.Unlock()

	for _, connection := range b.connections {
		connection.Write(event)
	}
}

func (b *Broker) SetDisconnectCallback(cb func(connectionId string)) {
	b.disconnectCallback = cb
}

func (b *Broker) Close() error {
	b.connectionsLock.Lock()
	defer b.connectionsLock.Unlock()

	for _, connection := range b.connections {
		connection.Close()
	}

	return nil
}
