package main

import (
	"flag"
	"fmt"

	"github.com/sergei-svistunov/smarthome/x10/controller"
	"github.com/sergei-svistunov/smarthome/webserver"
)

const APP_VERSION = "0.1"

// The flag package provides a default help printer via -h switch
var versionFlag *bool = flag.Bool("v", false, "Print the version number.")

func main() {
	flag.Parse() // Scan the arguments list

	if *versionFlag {
		fmt.Println("Version:", APP_VERSION)
	}

	x10_controller, x10ControllerErr := controller.NewController("/dev/tty1")
	webServer, webServerErr := webserver.NewWebserver(":38080")
	
	fmt.Println("!!!")
	
	if x10ControllerErr == nil {
		<-x10_controller.DoneChan
	} else {
		fmt.Println(x10ControllerErr)
	}
	
	if webServerErr == nil {
		<-webServer.DoneChan
	} else {
		fmt.Println(webServerErr)
	}

}
