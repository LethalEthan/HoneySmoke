package proxy

import (
	"HoneySmoke/config"
	"crypto/rand"
	"encoding/binary"
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
}

var (
	Log        = logging.MustGetLogger("HoneySmoke")
	Listener   net.Listener
	Limbo      = false
	LimboMutex sync.RWMutex
)

const (
	HANDSHAKE = 0
	STATUS    = 1
	LOGIN     = 2
	PLAY      = 3
)

func ProxyListener(IP string, Port string) {
	runtime.LockOSThread()
	var err error
	if Listener == nil {
		Listener, err = net.Listen("tcp", IP+Port)
		if err != nil {
			panic(err)
		}
	}
	go CheckForLimbo()
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
			//P.FAddr = ClientConn.RemoteAddr().String()
			P.Closed = true
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
			S, err = net.Dial("tcp", config.GConfig.Server.IP+config.GConfig.ProxyServer.Port)
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

func (P *ProxyObject) StartLimbo() {
	Log.Debug("LIMBOACTIVE!")
	ticks := time.Duration(20 * 50)
	ticker := time.NewTicker(ticks * time.Millisecond) //time.Tick(ticks * time.Millisecond)
	defer ticker.Stop()
	for GetLimbo() {
		<-ticker.C
		if P.GetState() == PLAY {
			//Keepalive
			PW := CreatePacketWriterWithCapacity(0x21, 18)
			buf := make([]byte, 8)
			rand.Read(buf)
			r := binary.LittleEndian.Uint64(buf)
			PW.WriteLong(int64(r))
			_, err := P.ClientConn.Write(PW.GetPacket())
			if err != nil {
				Log.Critical("Cannot send to client closing limbo")
				P.Close()
				return
			}
		}
	}
}
