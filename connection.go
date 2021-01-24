package sse

import (
	"errors"
	"github.com/google/uuid"
	"net/http"
	"time"
)

var heartbeatMessage = []byte(": heartbeat\n\n")

type Connection struct {
	id                string
	responseWriter    http.ResponseWriter
	request           *http.Request
	flusher           http.Flusher
	heartbeatInterval time.Duration
	msg               chan []byte
	onClose           func()
	closed            bool
	done              chan struct{}
}

func newConnection(w http.ResponseWriter, r *http.Request, heartbeatInterval time.Duration) (*Connection, error) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, errors.New("streaming not supported")
	}

	connection := &Connection{
		id:                uuid.New().String(),
		responseWriter:    w,
		request:           r,
		flusher:           flusher,
		msg:               make(chan []byte),
		heartbeatInterval: heartbeatInterval,
		done:              make(chan struct{}),
	}

	go connection.closeOnClientDisconnect()

	return connection, nil
}

func (c *Connection) ID() string {
	return c.id
}

// Serve starts writing the messages to the client
// This cannot be used to reopen a closed connection
func (c *Connection) Serve() error {
	if c.closed {
		return errors.New("can't serve closed connection")
	}

	go c.handleHeartbeat()

	for {
		select {
		case <-c.done:
			return nil
		case msg := <-c.msg:
			_, err := c.responseWriter.Write(msg)
			if err != nil {
				return errors.New("write failed")
			}
			c.flusher.Flush()
		}
	}
}

func (c *Connection) handleHeartbeat() {
	if c.heartbeatInterval <= 0 {
		return
	}

	heartbeats := time.NewTicker(c.heartbeatInterval)
	defer heartbeats.Stop()

	for {
		select {
		case <-c.done:
			return
		case <-heartbeats.C:
			c.msg <- heartbeatMessage
		}
	}
}

// Write writes an event to the client
func (c *Connection) Write(event Event) {
	c.msg <- event.format()
}

func (c *Connection) closeOnClientDisconnect() {
	<-c.request.Context().Done()
	c.close()
}

func (c *Connection) close() {
	close(c.done)
	c.closed = true

	if c.onClose != nil {
		c.onClose()
	}

	close(c.msg)
}

// Closed shows whether the connection was closed
func (c *Connection) Closed() bool {
	return c.closed
}
