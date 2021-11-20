package main

import (
	"HoneySmoke/config"
	"HoneySmoke/console"
	"HoneySmoke/proxy"
	"log"
	"os"
	"runtime"
	"runtime/debug"
	"runtime/pprof"

	logging "github.com/op/go-logging"
)

var (
	format = logging.MustStringFormatter("%{color}[%{time:01-02-2006 15:04:05.000}] [%{level}] [%{shortfunc}]%{color:reset} %{message}")
	Log    = logging.MustGetLogger("HoneySmoke")
	mprof  *os.File
	cprof  *os.File
)

func main() {
	//Hello from HoneySmoke
	//Logger Creation Start
	B1 := logging.NewLogBackend(os.Stderr, "", 0)       //New Backend
	B1Format := logging.NewBackendFormatter(B1, format) //Set Format
	B1LF := logging.AddModuleLevel(B1Format)            //Add formatting Levels
	_ = config.ConfigStart()
	if config.GConfig.Proxy.DEBUG {
		B1LF.SetLevel(logging.DEBUG, "")
	} else {
		B1LF.SetLevel(logging.INFO, "")
	}
	logging.SetBackend(B1LF)
	//Logger Creation END
	Log.Info("HoneySmoke", "0.0.0.5", "starting...")
	Log.Critical("Limbo mode is not available for HoneySmoke 0.0.0.5! due to instability and not enough testing (network re-written)")
	Log.Warning("HoneySmoke is in alpha! It is not complete and has many left overs and debugging statements left!")
	Log.Warning("Please report any bugs, unexpected behaviour and potential features you would like")
	if config.GConfig.Performance.CPU <= 0 {
		Log.Info("Setting GOMAXPROCS to all available logical CPU's")
		runtime.GOMAXPROCS(runtime.NumCPU()) //Set it to the value of how many cores
	} else {
		Log.Info("Setting GOMAXPROCS to config: ", config.GConfig.Performance.CPU)
		runtime.GOMAXPROCS(config.GConfig.Performance.CPU)
	}
	if config.GConfig.Performance.GCPercent < 0 {
		debug.SetGCPercent(100)
	} else {
		Log.Debug("Setting GCPercent to: ", config.GConfig.Performance.GCPercent)
		debug.SetGCPercent(config.GConfig.Performance.GCPercent)
	}
	//MemProf
	if config.MemProfile != "" {
		var err error
		mprof, err = os.Create(config.MemProfile)
		if err != nil {
			Log.Fatal(err)
		}
		pprof.StartCPUProfile(mprof)
	}
	//CPUProf
	if config.CpuProfile != "" {
		var err error
		cprof, err = os.Create(config.CpuProfile)
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}
		pprof.StartCPUProfile(cprof)
	}
	go console.Console()
	proxy.Keys()
	//go proxy.CheckForLimbo()
	if config.GConfig.Proxy.Host == "" {
		panic("Host is undefined in config!")
	}
	P := proxy.CreateProxyListener(config.GConfig.Proxy.Host)
	for i := 0; i < config.GConfig.Performance.Listeners-1; i++ {
		go P.ProxyListener()
	}
	P.ProxyListener()
}
