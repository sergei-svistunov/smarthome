package main

import (
	"flag"
	"fmt"

	"github.com/sergei-svistunov/smarthome/webserver"
	"github.com/sergei-svistunov/smarthome/x10"
)

const APP_VERSION = "0.1"

// The flag package provides a default help printer via -h switch
var versionFlag *bool = flag.Bool("v", false, "Print the version number.")

func main() {
	flag.Parse() // Scan the arguments list

	if *versionFlag {
		fmt.Println("Version:", APP_VERSION)
	}

	x10_controller, x10ControllerErr := x10.NewController("/dev/tty1")
	if x10ControllerErr == nil {
		addr, err := x10.StringToAddress("A1")
		if err == nil {
			x10_controller.AddDevice(x10.NewDeviceMDTx07(), "Bedroom", addr)
		}
		addr, err = x10.StringToAddress("A2")
		if err == nil {
			x10_controller.AddDevice(x10.NewDeviceMDTx07(), "Hall", addr)
		}
	} else {
		fmt.Println(x10ControllerErr)
	}

	webServer, webServerErr := webserver.NewWebserver(":38080", x10_controller)
	if webServerErr != nil {
		fmt.Println(webServerErr)
	}

	if x10ControllerErr == nil {
		<-x10_controller.DoneChan
	}

	if webServerErr == nil {
		<-webServer.DoneChan
	}
}
