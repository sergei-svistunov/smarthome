package webserver

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/golang/glog"
	"github.com/gorilla/websocket"

	"github.com/sergei-svistunov/smarthome/x10"
)

type Webserver struct {
	httpServer    *http.Server
	serveMux      *http.ServeMux
	DoneChan      chan bool
	x10Controller *x10.Controller
	wsHubDevices  *WSHub
}

func NewWebserver(addr string, x10Controller *x10.Controller) (*Webserver, error) {
	webserver := new(Webserver)
	webserver.DoneChan = make(chan bool)
	webserver.serveMux = http.NewServeMux()
	webserver.httpServer = &http.Server{
		Addr:    addr,
		Handler: webserver.serveMux,
	}
	webserver.x10Controller = x10Controller

	webserver.serveMux.Handle("/", http.FileServer(http.Dir("html")))

	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	webserver.wsHubDevices = NewWSHub()

	webserver.x10Controller.RegisterOnUpdateCallback(func(device x10.IDevice) {
		data, _ := json.Marshal(map[string]interface{}{
			"type":   "updateDevice",
			"device": webserver.x10Controller.GetDeviceInfo(device),
		})
		glog.Infof("%s", string(data))
		webserver.wsHubDevices.broadcast <- data
	})

	webserver.serveMux.HandleFunc("/devices/", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			glog.Error(err)
			return
		}

		glog.Infof("Accept connection to devices WS")

		message := webserver.x10Controller.GetInfo()
		message["type"] = "devicesList"
		jsonB, _ := json.Marshal(message)

		processWSConnection(webserver.wsHubDevices, conn, jsonB, func(message []byte) {
			glog.Infof("Devices WS received message: %s", message)

			data := make(map[string]interface{})
			if err := json.Unmarshal(message, &data); err != nil {
				glog.Error(err)
				return
			}

			if data["command"] == nil {
				glog.Error("No command")
				return
			}

			switch data["command"].(string) {
			case "ON":
				if data["device"] != nil {
					webserver.x10Controller.SendOn(data["device"].(string))
				}
			case "OFF":
				if data["device"] != nil {
					webserver.x10Controller.SendOff(data["device"].(string))
				}
			case "PRESET_DIM":
				if data["device"] != nil && data["volume"] != nil {
					webserver.x10Controller.SendPresetDim(data["device"].(string), uint8(data["volume"].(float64)))
				}
			default:
				glog.Errorf("Invalid command %s", data["command"])
			}
		})
	})

	go func() {
		err := webserver.httpServer.ListenAndServe()
		if err != nil {
			fmt.Println(err)
		}
		webserver.DoneChan <- true
	}()

	return webserver, nil
}
