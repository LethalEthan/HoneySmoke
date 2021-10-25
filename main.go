package main

import (
	"HoneySmoke/config"
	"HoneySmoke/proxy"
	"os"

	"github.com/op/go-logging"
)

var (
	format = logging.MustStringFormatter("%{color}[%{time:01-02-2006 15:04:05.000}] [%{level}] [%{shortfunc}]%{color:reset} %{message}")
	Log    = logging.MustGetLogger("HoneySmoke")
)

func main() {
	//Hello from HoneySmoke
	//Logger Creation Start
	B1 := logging.NewLogBackend(os.Stderr, "", 0)       //New Backend
	B1Format := logging.NewBackendFormatter(B1, format) //Set Format
	B1LF := logging.AddModuleLevel(B1Format)            //Add formatting Levels
	B1LF.SetLevel(logging.DEBUG, "")
	logging.SetBackend(B1LF)
	//Logger Creation END
	Log.Info("HoneySmoke" /*utils.GetVersionString()*/, "0.0.0.1", "starting...")
	Log.Warning("HoneySmoke is in alpha! It is not complete and has many left overs and debugging statements left!")
	conf := config.ConfigStart()
	proxy.ProxyListener(conf.ProxyServer.IP, conf.ProxyServer.Port)
}
