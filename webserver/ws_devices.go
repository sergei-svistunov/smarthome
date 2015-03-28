package webserver

import (
	"github.com/golang/glog"
	"github.com/sergei-svistunov/smarthome/x10"
	"golang.org/x/net/websocket"
)

type WsDevicesConnection struct {
	conn      *websocket.Conn
	doneChan  chan bool
	writeChan chan interface{}
	server    *WsDevices
}

type WsDevices struct {
	x10Controller *x10.Controller
	clients       map[*websocket.Conn]*WsDevicesConnection
}

func NewWsDevices(x10Controller *x10.Controller) *WsDevices {
	return &WsDevices{
		x10Controller: x10Controller,
		clients:       make(map[*websocket.Conn]*WsDevicesConnection),
	}
}

func (ws *WsDevices) AssignConnection(conn *websocket.Conn) {
	wsConn := &WsDevicesConnection{
		conn:      conn,
		doneChan:  make(chan bool),
		writeChan: make(chan interface{}, 100),
		server:    ws,
	}
	defer ws.doneConnection(conn)

	ws.clients[conn] = wsConn

	wsConn.onConnect()

	go wsConn.listenWrite()
	wsConn.listenRead()

	<-wsConn.doneChan
}

func (ws *WsDevices) doneConnection(conn *websocket.Conn) {
	conn.Close()
	delete(ws.clients, conn)
}

func (wsConn *WsDevicesConnection) onConnect() {
	message := wsConn.server.x10Controller.GetInfo()
	message["type"] = "devicesList"
	wsConn.write(message)
}

func (wsConn *WsDevicesConnection) write(data interface{}) {
	wsConn.writeChan <- data
}

func (wsConn *WsDevicesConnection) listenWrite() {
	for {
		select {
		case <-wsConn.doneChan:
			wsConn.doneChan <- true
			return

		case data := <-wsConn.writeChan:
			err := websocket.JSON.Send(wsConn.conn, data)
			if err != nil {
				wsConn.doneChan <- true
				return
			}
		}
	}
}

func (wsConn *WsDevicesConnection) listenRead() {
	for {
		select {
		case <-wsConn.doneChan:
			wsConn.doneChan <- true
			return

		default:
			var data interface{}
			err := websocket.JSON.Receive(wsConn.conn, &data)

			if err == nil {
				glog.Infof("%+v", data)

				switch data.(type) {
				case map[string]interface{}:
					wsConn.doCmd(data.(map[string]interface{}))
				default:
					glog.Error("Invalid JSON")
				}
			} else {
				wsConn.doneChan <- true
				return
			}
		}
	}
}

func (wsConn *WsDevicesConnection) doCmd(data map[string]interface{}) {
	if data["command"] == nil {
		glog.Error("No command")
		return
	}

	switch data["command"] {
	case "ON":
		if data["device"] != nil {
			wsConn.server.x10Controller.SendOn(data["device"].(string), 1)
		}
	case "OFF":
		if data["device"] != nil {
			wsConn.server.x10Controller.SendOff(data["device"].(string), 1)
		}
	default:
		glog.Errorf("Invalid command %s", data["command"])
	}
}
