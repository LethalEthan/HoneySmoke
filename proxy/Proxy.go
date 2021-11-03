package proxy

import (
	"net"
	"runtime"
	"sync"

	"github.com/google/uuid"
	"github.com/op/go-logging"
)

//Multiple proxies can be setup on different ports and IP's -- not finished

type Proxy struct {
	IP                string
	Port              string
	Listener          net.Listener
	ListenerMutex     sync.Mutex
	Limbo             bool
	ProxyObjects      map[string]ProxyObject
	ProxyObjectsMutex sync.RWMutex
	LimboMutex        sync.RWMutex
}

type ProxyObject struct {
	ClientConn        net.Conn
	ServerConn        net.Conn
	Closed            bool
	Reconnection      bool
	CloseMutex        sync.RWMutex
	ReconnectionMutex sync.Mutex
	PPSCount          uint32
	State             uint32
	Compression       int32
	ProtocolVersion   int32
	Player            PlayerObject
	ServerAddress     string
	PacketPerSecondC  chan byte
}

type PlayerObject struct {
	UUID                 uuid.UUID
	ChatMode             int32 //Could've been a byte
	MainHand             int32 //Why is this varint
	Name                 string
	Locale               string
	DisplayedSkinParts   byte
	ViewDistance         int8
	ChatColours          bool
	DisableTextFiltering bool
	Authenticated        bool
}

var (
	Log       = logging.MustGetLogger("HoneySmoke")
	MainProxy Proxy
)

const (
	HANDSHAKE = 0
	STATUS    = 1
	LOGIN     = 2
	PLAY      = 3
)

func CreateProxyListener(Host string) *Proxy {
	MainProxy = *new(Proxy)
	MainProxy.ProxyObjects = make(map[string]ProxyObject)
	if MainProxy.GetListener() == nil {
		L, err := net.Listen("tcp", Host)
		if err != nil {
			panic(err)
		}
		MainProxy.SetListener(L)
	}
	Log.Info("Created Proxy listener")
	return &MainProxy
}

func (P *Proxy) ProxyListener() {
	runtime.LockOSThread()
	for {
		ClientConn, err := P.Listener.Accept()
		if err != nil {
			Log.Critical("Could not accept connection!")
			Log.Critical("Reason: ", err)
			ClientConn.Close()
		} else {
			//Create Proxy object with all required info and spin off the handlers
			PO := new(ProxyObject)
			PO.Player = *new(PlayerObject)
			PO.ClientConn = ClientConn
			PO.Closed = false
			PO.PacketPerSecondC = make(chan byte, 10)
			P.Set(ClientConn.RemoteAddr().String(), *PO)
			//P.ProxyObjects[ClientConn.RemoteAddr().String()] = *PO
			go PO.ProxyBackEnd()
		}
	}
}

// func CheckForLimbo() {
// 	if config.GConfig.Performance.LimboMode {
// 		var err error
// 		var S net.Conn
// 		Log.Debug("LIMBO CHECK ACTIVE!")
// 		ticks := time.Duration(config.GConfig.Performance.CheckServerOnlineTick * 50)
// 		ticker := time.NewTicker(ticks * time.Millisecond) //time.Tick(ticks * time.Millisecond)
// 		defer ticker.Stop()
// 		for {
// 			<-ticker.C
// 			S, err = net.Dial("tcp", config.GConfig.Backends.Servers[0])
// 			if err != nil {
// 				SetLimbo(true)
// 				<-ticker.C
// 			} else {
// 				SetLimbo(false)
// 			}
// 			S.Close()
// 		}
// 	}
// }
