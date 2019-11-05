package ws

import (
	"bootic_data_collector/data"
	"fmt"
	"golang.org/x/net/websocket"
	"net/http"
	"strings"
)

type Connection struct {
	// The websocket connection.
	ws  *websocket.Conn
	hub *Hub
	// Buffered channel of outbound messages.
	send data.EventsChannel

	// Filters
	tags []string
}

func (c *Connection) reader() {
	tagsQuery := c.ws.Request().URL.Query().Get("tags")
	var tags []string

	if tagsQuery != "" {
		tags = strings.Split(tagsQuery, ",")
		c.tags = append(c.tags, tags...)
	}

	fmt.Println("ws [conn] initialized with", c.tags)

	// We need to block here, otherwise the connection closes. Not sure what the best solution is.
	for {
		var message string
		err := websocket.Message.Receive(c.ws, &message)
		if err != nil {
			break
		}
		// c.hub.broadcast <- message
	}
	c.ws.Close()
}

func encodeEventIntoString(event *data.Event) (str string, err error) {
	bytes, err := data.EncodeJSON(event)
	if err != nil {
		return
	}
	return string(bytes), err
}

// An event must match all filters in a connection in order to be sent to connection
// If connection has no filters, then we assume connection wants ALL events
func (c *Connection) includedInFilters(event *data.Event) bool {
	// if len(c.tags) == 0 { // no filters set. Allow everything
	//     return true
	//   } else { // only for set filters
	//     matches := 0
	//     for _, myTag := range c.tags {
	//       for _, t := range event.Tags {
	//         fmt.Println("INCHECK", myTag, t)
	//         if t == myTag {
	//           matches = matches + 1
	//         }
	//       }
	//     }
	//     if matches == len(c.tags) {
	//       return true
	//     }
	//   }
	return true
}

func (c *Connection) writer() {
	for event := range c.send {
		if c.includedInFilters(event) {
			message, err := encodeEventIntoString(event)
			if err != nil {
				break
			}

			err2 := websocket.Message.Send(c.ws, message)

			if err2 != nil {
				break
			}
		}
	}
	fmt.Println("NEVER HERE")
	c.ws.Close()
}

func HandleWebsocketsHub(path string) *Hub {

	hub := NewHub()

	http.Handle(path, websocket.Handler(func(ws *websocket.Conn) {
		c := &Connection{send: make(data.EventsChannel, 512), ws: ws, hub: hub}
		hub.register <- c
		defer func() { hub.unregister <- c }()
		go c.writer()
		c.reader()
	}))

	return hub
}
