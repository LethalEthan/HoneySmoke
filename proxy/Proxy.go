package proxy

import (
	"HoneySmoke/config"
	"net"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/op/go-logging"
)

//Multiple proxies can be setup on different ports and IP's -- not finished

type Proxy struct {
	IP                string
	Port              uint16
	Limbo             bool
	Listener          net.Listener
	ListenerMutex     sync.Mutex
	ProxyObjects      map[string]*ProxyObject
	ProxyObjectsMutex sync.RWMutex
	LimboMutex        sync.RWMutex
}

type ProxyObject struct {
	ClientConn        net.Conn
	ServerConn        net.Conn
	CloseMutex        sync.RWMutex
	ReconnectionMutex sync.Mutex
	Player            PlayerObject
	PPSCount          uint32
	State             uint32
	Compression       int32
	ProtocolVersion   int32
	ServerAddress     string
	Closed            bool
	Reconnection      bool
	PacketPerSecondC  chan byte
}

type PlayerObject struct {
	UUID                 uuid.UUID
	Name                 string
	Locale               string
	ChatMode             int32 //Could've been a byte
	MainHand             int32 //Why is this varint
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
	MainProxy.ProxyObjects = make(map[string]*ProxyObject)
	MainProxy.IP = strings.Split(config.GConfig.Proxy.Host, ":")[0]
	Port, _ := strconv.ParseUint(strings.Split(config.GConfig.Proxy.Host, ":")[1], 10, 16)
	MainProxy.Port = uint16(Port)
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
			P.Set(ClientConn.RemoteAddr().String(), PO)
			go PO.StartHandles()
		}
	}
}

func (P *ProxyObject) StartHandles() {
	var err error
	for i := 0; i < config.GConfig.Performance.CheckServerChances; i++ {
		P.ServerConn, err = net.Dial("tcp", config.GConfig.Backends.Servers[0])
		if err == nil {
			Log.Debug("Connected handling...")
			MainProxy.SetLimbo(false)
			break
		} else {
			Log.Critical("Error dialing, waiting", config.GConfig.Performance.CheckServerSeconds, "seconds until retry")
			err = nil //reset err otherwise it could just loop
			MainProxy.SetLimbo(true)
		}
		time.Sleep(time.Duration(config.GConfig.Performance.CheckServerSeconds) * time.Second)
	}
	if P.ServerConn != nil {
		go P.HandleFrontEnd()
		go P.HandleBackEnd()
	} else {
		P.ClientConn.Close()
		return
	}
}
