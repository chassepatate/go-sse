package sse

import (
	"errors"
	"net/http"
	"sync"
	"time"
)

var ErrServerClosed = errors.New("server closed")
var ErrUnknownConnection = errors.New("connection not found")

type Server struct {
	mu                 sync.Mutex
	closed             bool
	connections        map[string]*Connection
	heartBeatInterval  time.Duration
	customHeaders      map[string]string
	disconnectCallback func(connectionId string)
}

func NewServer() *Server {
	return &Server{
		connections:   make(map[string]*Connection),
		customHeaders: make(map[string]string),
	}
}

func (s *Server) NewConnection(w http.ResponseWriter, r *http.Request) (*Connection, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil, ErrServerClosed
	}

	s.setHeaders(w)

	connection, err := newConnection(w, r, s.heartBeatInterval)
	if err != nil {
		return nil, err
	}

	connection.onClose = func() {
		s.deleteConnection(connection.id)
	}

	s.connections[connection.id] = connection

	return connection, nil
}

func (s *Server) SetCustomHeaders(headers map[string]string) {
	s.customHeaders = headers
}

func (s *Server) SetHeartBeatInterval(d time.Duration) {
	s.heartBeatInterval = d
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

func (s *Server) deleteConnection(connectionId string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.connections, connectionId)

	if s.disconnectCallback != nil {
		go s.disconnectCallback(connectionId)
	}
}
func (s *Server) getConnection(id string) (*Connection, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	connection, exists := s.connections[id]
	if !exists {
		return nil, ErrUnknownConnection
	}

	return connection, nil
}

func (s *Server) getConnections() []*Connection {
	s.mu.Lock()
	defer s.mu.Unlock()

	result := make([]*Connection, 0, len(s.connections))
	for _, c := range s.connections {
		result = append(result, c)
	}

	return result
}

func (s *Server) Write(connectionId string, event Event) error {
	connection, err := s.getConnection(connectionId)
	if err != nil {
		return err
	}

	connection.Write(event)

	return nil
}

func (s *Server) Broadcast(event Event) {
	for _, c := range s.getConnections() {
		c.Write(event)
	}
}

func (s *Server) SetDisconnectCallback(cb func(connectionId string)) {
	s.disconnectCallback = cb
}

func (s *Server) Close() error {
	s.mu.Lock()
	s.closed = true
	s.mu.Unlock()

	for _, connection := range s.connections {
		// The callback executed on close uses the server mutex, so it's important we don't lock the mutex here
		connection.Close()
	}

	return nil
}
