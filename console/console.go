package console

import (
	"HoneySmoke/config"
	"bufio"
	"crypto/md5"
	"encoding/hex"
	"io"
	"log"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"syscall"

	logging "github.com/op/go-logging"
)

var (
	conf     *config.Config
	shutdown = make(chan os.Signal, 1)
	Log      = logging.MustGetLogger("HoneyBEE")
	Panicked bool
	hprof    *os.File
	cprof    *os.File
)

func Console() {
	runtime.LockOSThread()
	go Shutdown()
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		switch scanner.Text() {
		case "help":
			Log.Warning("There is no help atm :(")
			Log.Warning("This is a simple, quick and dirty way of doing commands, a proper thing is being made later")
		case "shutdown":
			shutdown <- os.Interrupt
		case "stop":
			shutdown <- os.Interrupt
		case "exit":
			shutdown <- os.Interrupt
		case "reload":
			config.ConfigReload()
			Log.Info("Reloaded config!")
		case "GC":
			runtime.GC()
			Log.Info("GC invoked")
		case "mem":
			PrintDebugStats()
		case "panic":
			panic("panicked, you told me to :)")
		case "cpuprofile":
			if config.CpuProfile != "" {
				pprof.StopCPUProfile()
				cprof.Close()
				Log.Warning("Written CPU Profile")
			} else {
				Log.Critical("cpuprofile flag not specified! not writing a profile")
			}
		case "memprofile":
			if config.MemProfile != "" {
				if err := pprof.WriteHeapProfile(hprof); err != nil {
					log.Fatal("could not write memory profile: ", err)
				} else {
					Log.Warning("Written Memory Profile")
				}
			} else {
				Log.Critical("memprofile flag not specified! not writing a profile")
			}
		case "profile":
			if config.CpuProfile != "" {
				pprof.StopCPUProfile()
				cprof.Close()
				Log.Warning("Written CPU Profile")
			} else {
				Log.Critical("cpuprofile flag not specified! not writing a cpuprofile")
			}
			//
			if config.MemProfile != "" {
				runtime.GC() // get up-to-date statistics
				if err := pprof.WriteHeapProfile(hprof); err != nil {
					log.Fatal("could not write memory profile: ", err)
				} else {
					Log.Warning("Written Memory Profile")
				}
			} else {
				Log.Critical("memprofile flag not specified! not writing a cpuprofile")
			}
		default:
			Log.Warning("Unknown command")
		}
	}
}

var MD5 string

func Hash() string {
	file, err := os.Open(os.Args[0])
	if err != nil {
		MD5 = "00000000000000000000000000000000"
	}
	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		MD5 = "00000000000000000000000000000000"
	}
	//Get the 16 bytes hash
	hBytes := hash.Sum(nil)[:16]
	file.Close()
	MD5 = hex.EncodeToString(hBytes) //Convert bytes to string
	return MD5
}

//Shutdown - listens for sigterm and exits
func Shutdown() {
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)
	<-shutdown
	Log.Warning("Starting shutdown")
	if config.GConfig.ProxyServer.DEBUG {
		PrintDebugStats()
	}
	//fmt.Println(proxy.ProxyObjects)
	os.Exit(0)
}
