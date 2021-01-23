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
	connections        *connectionStore
	heartBeatInterval  time.Duration
	customHeaders      map[string]string
	disconnectCallback func(connectionId string)
}

func NewServer() *Server {
	return &Server{
		connections:   newConnectionStore(),
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

	s.connections.add(connection)

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

func (s *Server) deleteConnection(id string) {
	s.connections.delete(id)

	if s.disconnectCallback != nil {
		go s.disconnectCallback(id)
	}
}
func (s *Server) Write(connectionId string, event Event) error {
	connection, exists := s.connections.get(connectionId)
	if !exists {
		return ErrUnknownConnection
	}

	connection.Write(event)

	return nil
}

func (s *Server) Broadcast(event Event) {
	for _, c := range s.connections.getAll() {
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

	for _, c := range s.connections.getAll() {
		c.Close()
	}

	return nil
}
