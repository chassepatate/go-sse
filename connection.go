package sse

import (
	"errors"
	"github.com/google/uuid"
	"net/http"
	"time"
)

type Connection struct {
	id string

	responseWriter http.ResponseWriter
	request        *http.Request
	flusher        http.Flusher

	heartBeatInterval time.Duration

	msg     chan []byte
	onClose func()
	closed  bool
}

// Users should not create instances of client. This should be handled by the SSE server.
func newConnection(w http.ResponseWriter, r *http.Request, interval time.Duration) (*Connection, error) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, errors.New("streaming not supported")
	}

	return &Connection{
		id:             uuid.New().String(),
		responseWriter: w,
		request:        r,
		flusher:        flusher,
		msg:            make(chan []byte),
		onClose:        func() {},
	}, nil
}

func (c *Connection) ID() string {
	return c.id
}

func (c *Connection) Open() error {
	return c.serve()
}

func (c *Connection) Closed() bool {
	return c.closed
}

func (c *Connection) Write(event Event) {
	bytes := event.format()
	c.msg <- bytes
}

func (c *Connection) serve() error {
	defer func() {
		c.Close()
	}()

	var heartBeat <-chan time.Time

	if c.heartBeatInterval > 0 {
		ticker := time.NewTicker(c.heartBeatInterval)
		heartBeat = ticker.C
		defer func() {
			ticker.Stop()
		}()
	}

writeLoop:
	for {
		select {
		case <-c.request.Context().Done():
			break writeLoop
		case <-heartBeat:
			c.Write(Event{
				Event: "heartbeat",
			})
		case msg, open := <-c.msg:
			if !open {
				return errors.New("msg chan closed")
			}
			_, err := c.responseWriter.Write(msg)
			if err != nil {
				return errors.New("write failed")
			}
			c.flusher.Flush()
		}
	}
	return nil
}

func (c *Connection) Close() {
	c.onClose()
	c.closed = true
}
