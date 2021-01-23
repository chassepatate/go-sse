package sse

import (
	"bytes"
	"fmt"
	"strings"
)

type Event struct {
	Id    string
	Event string
	Data  string
}

func (e Event) format() []byte {
	var data bytes.Buffer

	if len(e.Id) > 0 {
		data.WriteString(fmt.Sprintf("id: %s\n", strings.Replace(e.Id, "\n", "", -1)))
	}

	data.WriteString(fmt.Sprintf("event: %s\n", strings.Replace(e.Event, "\n", "", -1)))

	// data field should not be empty
	lines := strings.Split(e.Data, "\n")
	for _, line := range lines {
		data.WriteString(fmt.Sprintf("data: %s\n", line))
	}

	data.WriteString("\n")
	return data.Bytes()
}
