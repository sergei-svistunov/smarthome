package main

import (
	"encoding/json"
	"flag"
	"os"

	"github.com/golang/glog"

	"github.com/sergei-svistunov/smarthome/webserver"
	"github.com/sergei-svistunov/smarthome/x10"
)

func main() {
	flag.Parse()

	f, err := os.Open("config.json")
	if err != nil {
		glog.Fatal(err)
	}

	jdec := json.NewDecoder(f)
	var config struct {
		X10 struct {
			Controller struct {
				TTY     string
				Repeats byte
			}
			Devices []struct {
				Caption, Type, Address string
			}
		}

		WebServer struct {
			Listen string
		}

		Torrent struct {
			SavePath string
		}
	}
	err = jdec.Decode(&config)
	if err != nil {
		glog.Fatal(err)
	}

	x10_controller, x10ControllerErr := x10.NewController(config.X10.Controller.TTY, config.X10.Controller.Repeats)
	if x10ControllerErr == nil {
		for _, devConf := range config.X10.Devices {
			addr, err := x10.StringToAddress(devConf.Address)
			if err != nil {
				glog.Error(err)
				continue
			}
			switch devConf.Type {
			case "MDTx07":
				x10_controller.AddDevice(x10.NewDeviceMDTx07(), devConf.Caption, addr)
			default:
				glog.Errorf("Unknown X10 device type \"%s\"", devConf.Type)
			}
		}
	} else {
		glog.Error(x10ControllerErr)
	}

	webServer, webServerErr := webserver.NewWebserver(config.WebServer.Listen, x10_controller)
	if webServerErr != nil {
		glog.Error(x10ControllerErr)
	}

	if x10ControllerErr == nil {
		<-x10_controller.DoneChan
	}

	if webServerErr == nil {
		<-webServer.DoneChan
	}
}
