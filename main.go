package main

import (
	"HoneySmoke/config"
	"HoneySmoke/console"
	"HoneySmoke/proxy"
	"os"
	"runtime"

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
	conf := config.ConfigStart()
	if config.GConfig.ProxyServer.DEBUG {
		B1LF.SetLevel(logging.DEBUG, "")
	} else {
		B1LF.SetLevel(logging.INFO, "")
	}
	logging.SetBackend(B1LF)
	//Logger Creation END
	Log.Info("HoneySmoke" /*utils.GetVersionString()*/, "0.0.0.1", "starting...")
	Log.Warning("HoneySmoke is in alpha! It is not complete and has many left overs and debugging statements left!")
	Log.Warning("Please report any bugs, unexpected behaviour and potential features you would like")
	go console.Console()
	proxy.Keys()
	if config.GConfig.Performance.CPU <= 0 {
		Log.Info("Setting GOMAXPROCS to all available logical CPU's")
		runtime.GOMAXPROCS(runtime.NumCPU()) //Set it to the value of how many cores
	} else {
		Log.Info("Setting GOMAXPROCS to config: ", config.GConfig.Performance.CPU)
		runtime.GOMAXPROCS(config.GConfig.Performance.CPU)
	}
	go proxy.CheckForLimbo()
	proxy.CreateProxyListener(conf.ProxyServer.IP, conf.ProxyServer.Port)
	for i := 0; i < conf.Performance.Listeners-1; i++ {
		go proxy.ProxyListener()
	}
	proxy.ProxyListener()
}
