package proxy

import (
	"HoneySmoke/config"
	"net"
	"runtime"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/op/go-logging"
)

type ProxyObject struct {
	ClientConn        net.Conn
	ServerConn        net.Conn
	CloseMutex        sync.RWMutex
	LimboMutex        sync.Mutex
	Closed            bool
	Limbo             bool
	State             uint32
	ProtocolVersion   int32
	ServerAddress     string
	Compression       int32
	Player            *PlayerObject
	Reconnection      bool
	ReconnectionMutex sync.Mutex
}

type PlayerObject struct {
	UUID                 uuid.UUID
	Name                 string
	Locale               string
	ViewDistance         int8
	ChatMode             int32 //Could've been a byte
	ChatColours          bool
	DisplayedSkinParts   byte
	MainHand             int32 //Why is this varint
	DisableTextFiltering bool
	Authenticated        bool
}

var (
	Log               = logging.MustGetLogger("HoneySmoke")
	Listener          net.Listener
	ListenerMutex     sync.Mutex
	Limbo             = false
	LimboMutex        sync.RWMutex
	ProxyObjects      map[string]*ProxyObject
	ProxyObjectsMutex sync.RWMutex
)

const (
	HANDSHAKE = 0
	STATUS    = 1
	LOGIN     = 2
	PLAY      = 3
)

func CreateProxyListener(IP string, Port string) {
	if GetListener() == nil {
		L, err := net.Listen("tcp", IP+Port)
		if err != nil {
			panic(err)
		}
		SetListener(L)
	}
}

func ProxyListener() {
	runtime.LockOSThread()
	for {
		ClientConn, err := Listener.Accept()
		if err != nil {
			Log.Critical("Could not accept connection!")
			Log.Critical("Reason: ", err)
			ClientConn.Close()
		} else {
			//Create Proxy object with all required info and spin off the handlers
			P := new(ProxyObject)
			P.Player = new(PlayerObject)
			P.ClientConn = ClientConn
			P.Closed = false
			go P.ProxyBackEnd()
		}
	}
}

func CheckForLimbo() {
	if config.GConfig.Performance.LimboMode {
		var err error
		var S net.Conn
		Log.Debug("LIMBO CHECK ACTIVE!")
		ticks := time.Duration(config.GConfig.Performance.CheckServerOnlineTick * 50)
		ticker := time.NewTicker(ticks * time.Millisecond) //time.Tick(ticks * time.Millisecond)
		defer ticker.Stop()
		for {
			<-ticker.C
			S, err = net.Dial("tcp", config.GConfig.Backends.Servers[0])
			if err != nil {
				SetLimbo(true)
				<-ticker.C
			} else {
				SetLimbo(false)
			}
			S.Close()
		}
	}
}
