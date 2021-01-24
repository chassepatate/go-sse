package sse

import (
	"errors"
	"net/http"
	"time"
)

const heartbeatInterval = 15 * time.Second

var ErrServerClosed = errors.New("server closed")
var ErrUnknownConnection = errors.New("connection not found")

type Server struct {
	connections        *connectionStore
	heartbeatInterval  time.Duration
	customHeaders      map[string]string
	disconnectCallback func(connectionId string)
}

func NewServer() *Server {
	return &Server{
		connections:       newConnectionStore(),
		heartbeatInterval: heartbeatInterval,
		customHeaders:     make(map[string]string),
	}
}

func (s *Server) NewConnection(w http.ResponseWriter, r *http.Request) (*Connection, error) {
	s.setHeaders(w)

	connection, err := newConnection(w, r, s.heartbeatInterval)
	if err != nil {
		return nil, err
	}

	connection.onClose = func() {
		s.deleteConnection(connection.id)
	}

	s.connections.add(connection)

	return connection, nil
}

// SetCustomHeaders adds custom response headers
func (s *Server) SetCustomHeaders(headers map[string]string) {
	s.customHeaders = headers
}

// SetHeartbeatInterval sets the interval of heartbeats which are used to keep connections open.
// Setting the interval to 0 disables the heartbeat
func (s *Server) SetHeartbeatInterval(d time.Duration) {
	s.heartbeatInterval = d
}

func (s *Server) setHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Transfer-Encoding", "chunked")

	for k, v := range s.customHeaders {
		w.Header().Set(k, v)
	}
}

func (s *Server) deleteConnection(id string) {
	s.connections.delete(id)

	if s.disconnectCallback != nil {
		go s.disconnectCallback(id)
	}
}

// Write writes an event to a connection based on the ID
// If the connection ID is not found, it will return an error
func (s *Server) Write(connectionId string, event Event) error {
	connection, exists := s.connections.get(connectionId)
	if !exists {
		return ErrUnknownConnection
	}
	connection.Write(event)
	return nil
}

// Broadcast sends an event to all connected clients
func (s *Server) Broadcast(event Event) {
	for _, c := range s.connections.getAll() {
		c.Write(event)
	}
}

// SetDisconnectCallback sets a function which will be called when a connection is closed
func (s *Server) SetDisconnectCallback(cb func(connectionId string)) {
	s.disconnectCallback = cb
}
