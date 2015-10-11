package webserver

import (
	"github.com/golang/glog"
	"github.com/gorilla/websocket"
)

type WSHub struct {
	connections map[*WSConnection]bool
	register    chan *WSConnection
	unregister  chan *WSConnection
	broadcast   chan []byte
}

type WSConnection struct {
	conn *websocket.Conn
	send chan []byte
}

func processWSConnection(hub *WSHub, conn *websocket.Conn, helloData []byte, callback func([]byte)) {
	connection := &WSConnection{
		conn: conn,
		send: make(chan []byte, 256),
	}

	hub.register <- connection

	connection.send <- helloData

	// Write messages
	go func() {
		defer func() {
			conn.Close()
		}()

		for {
			message, ok := <-connection.send
			if !ok {
				conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		}
	}()

	// Read messages
	defer func() {
		hub.unregister <- connection
		conn.Close()
	}()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			break
		}
		glog.Infof("WS workers message: %s\n", message)
		
		callback(message)
	}
}

func NewWSHub() *WSHub {
	hub := &WSHub{
		connections: make(map[*WSConnection]bool),
		register:    make(chan *WSConnection),
		unregister:  make(chan *WSConnection),
		broadcast:   make(chan []byte),
	}

	go func() {
		for {
			select {
			case c := <-hub.register:
				hub.connections[c] = true
			case c := <-hub.unregister:
				delete(hub.connections, c)
			case m := <-hub.broadcast:
				for c := range hub.connections {
					select {
					case c.send <- m:
					default:
						close(c.send)
						delete(hub.connections, c)
					}
				}
			}
		}
	}()

	return hub
}
