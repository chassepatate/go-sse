package sse

import "sync"

type connectionStore struct {
	mu          sync.Mutex
	connections map[string]*Connection
}

func newConnectionStore() *connectionStore {
	return &connectionStore{
		connections: make(map[string]*Connection),
	}
}

func (s *connectionStore) add(connection *Connection) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.connections[connection.id] = connection
}

func (s *connectionStore) getAll() []*Connection {
	s.mu.Lock()
	defer s.mu.Unlock()

	result := make([]*Connection, 0, len(s.connections))
	for _, c := range s.connections {
		result = append(result, c)
	}

	return result
}

// Returns the connection and a bool that indicates whether the connection exists
func (s *connectionStore) get(id string) (*Connection, bool) {
	c, exists := s.connections[id]
	return c, exists
}

func (s *connectionStore) delete(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.connections, id)
}
