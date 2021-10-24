package proxy

import (
	"HoneySmoke/config"
	"bytes"
	"compress/zlib"
	"crypto/rand"
	"encoding/binary"
	"io"
	"io/ioutil"
	"net"
	"runtime"
	"sync"
	"time"

	"github.com/op/go-logging"
)

type ProxyObject struct {
	ClientConn      net.Conn
	ServerConn      net.Conn
	CloseMutex      sync.RWMutex
	Closed          bool
	Limbo           bool
	State           uint32
	ProtocolVersion int32
	ServerAddress   string
	Compression     int32
}

var (
	Log      = logging.MustGetLogger("HoneySmoke")
	Listener net.Listener
)

const (
	HANDSHAKE = 0
	STATUS    = 1
	LOGIN     = 2
	PLAY      = 3
)

func ProxyFrontEnd(IP string, Port string) {
	runtime.LockOSThread()
	var err error
	Listener, err = net.Listen("tcp", IP+Port)
	if err != nil {
		panic(err)
	}
	for {
		ClientConn, err := Listener.Accept()
		if err != nil {
			Log.Critical("Could not accept connection!")
			Log.Critical("Reason: ", err)
			ClientConn.Close()
		} else {
			//Create Proxy object with all required info and spin off the handlers
			P := new(ProxyObject)
			P.ClientConn = ClientConn
			//P.FAddr = ClientConn.RemoteAddr().String()
			P.Closed = true
			go P.ProxyBackEnd()
		}
	}
}

func (P *ProxyObject) ProxyBackEnd() {
	var err error
	for {
		P.ServerConn, err = net.Dial("tcp", config.GConfig.Server.IP+config.GConfig.Server.Port)
		if err == nil {
			Log.Debug("Connected handling...")
			break
		} else {
			Log.Critical("Error dialing, waiting 3 seconds until retry")
			err = nil //reset err otherwise it could just loop
		}
		time.Sleep(3 * time.Second)
	}
	go P.HandleFrontEnd()
	go P.HandleBackEnd()
	//go P.CheckForLimbo()
}

func (P *ProxyObject) HandleBackEnd() {
	var data = make([]byte, 2097152) //Create buffer to store packet contents in
	var err error
	var BytesRead int
	var PacketID int32
	var PacketSize int32
	var DataLength int32
	var ByteReader *bytes.Reader
	PR := CreatePacketReader([]byte{0x00})
	P.SetState(HANDSHAKE)
	P.SetCompression(-1)
	Log.Debug("Started Server Handle")
	for P.CheckClosed() {
		err = nil
		BytesRead, err = P.ServerConn.Read(data)
		if err != nil {
			if err == io.EOF {
				Log.Critical("Closing Backend Handle")
				P.Close()
				return
			} else {
				Log.Critical("Closing because: ", err)
				P.Close()
				return
			}
		}
		if BytesRead < 0 {
			Log.Critical("Closing because bytes read is 0!")
			P.Close()
			return
		}
		PR.SetData(data[:BytesRead])
		PacketSize, _, err = PR.ReadVarInt()
		if err != nil {
			Log.Critical("Could not read packet size!")
		}
		//If compression is set
		if P.GetCompression() > 0 /*&& PacketSize > P.Compression*/ {
			//Log.Debug("Compression branch server -> client", "State: ", P.State, "PacketSize: ", PacketSize)
			//Data length is uncompressed length of packetID and data
			//Log.Debug("PacketSize: ", PacketSize, "STATE: ", P.GetState())
			if PacketSize > P.GetCompression() {
				DataLength, _, err = PR.ReadVarInt()
				if err != nil {
					Log.Debug("Could not read Data Length!", err)
					P.Close()
					return
				}
				if DataLength == 0 {
					Log.Debug("Data length is 0")
				}
				//Log.Debug("Data length: ", DataLength)
				//Read the compressed packet into byte reader
				ByteReader = bytes.NewReader(PR.ReadRestOfByteArray())
				//New zlib reader
				ZlibReader, err := zlib.NewReader(ByteReader)
				if err != nil {
					panic(err)
				}
				//Read decompressed packet
				DecompressedPacket, err := ioutil.ReadAll(ZlibReader)
				if err != nil {
					panic(err)
				}
				//Close ZlibReader
				ZlibReader.Close()
				//Check if size is correct
				if len(DecompressedPacket) != int(DataLength) {
					Log.Debug("Data length != bytes read")
				}
				//Set PacketReader to the decompressed packet
				PR.SetData(DecompressedPacket)
				//Read Packet ID
				PacketID, _, err = PR.ReadVarInt()
				if err != nil {
					Log.Debug("Could not read PacketID!", err)
					P.Close()
					return
				}
				//Log.Debug("PID: ", PacketID, "DataLength: ", DataLength)
			} else { //Not-compressed but compression is used and is below the threshold
				/*DataLength*/ _, _, err = PR.ReadVarInt()
				if err != nil {
					Log.Debug("Could not read Data Length!", err)
					P.Close()
					return
				}
				// if DataLength == 0 {
				// 	Log.Debug("Data length is 0")
				// }
				PacketID, _, err = PR.ReadVarInt()
				if err != nil {
					Log.Debug("Could not read PacketID!", err)
					P.Close()
					return
				}
				//Log.Debug("PID: ", PacketID, "DataLength: ", DataLength)
			}
			//Log.Debug("PacketID C: ", PacketID)
		} else { //No compression
			//Log.Debug("No compression branch server -> client, state: ", P.State, "PacketSize: ", PacketSize)
			PacketID, _, err = PR.ReadVarInt()
			if err != nil {
				Log.Debug("Could not read PacketID!", err)
				P.Close()
				return
			}
			// Log.Debug("packetID NC: ", PacketID, "PacketSize: ", PacketSize, "State: ", P.GetState())
			// if len(PR.Data) > 4 {
			// 	Log.Debug("Packet: ", PR.Data[0:3], "COMP: ", P.GetCompression())
			// 	if P.GetCompression() > 0 {
			// 		T, _, _ := PR.ReadVarInt()
			// 		Log.Debug("TEST: ", T, "DB: ", DB)
			// 	}
			// }
		}
		switch P.GetState() {
		case LOGIN:
			switch PacketID {
			case 0x02:
				Log.Debug("Login success, setting play state")
				P.SetState(PLAY)
			case 0x03:
				if P.GetCompression() < 0 {
					Log.Debug("Set Compression recieved")
					Compression, _, err := PR.ReadVarInt()
					P.SetCompression(Compression)
					if err != nil {
						Log.Critical("Could not read compression threshold!")
					}
					Log.Debug("Compression threshold: ", Compression)
				}
			}
		case PLAY:
			switch PacketID {
			case 0x21:
				Log.Debug("Recieved Keepalive CB 0x21")
			}
		}
		err = nil
		P.ClientConn.Write(data[:BytesRead])
	}
	Log.Critical("Left loop lmao")
}

func (P *ProxyObject) HandleFrontEnd() {
	var data = make([]byte, 2097152) //Create buffer to store packet contents in
	var err error
	var BytesRead int
	var PacketID int32
	var PacketSize int32
	var DataLength int32
	var ByteReader *bytes.Reader
	PR := CreatePacketReader([]byte{0x00})
	Log.Debug("Started Client Handle")
	P.SetState(HANDSHAKE)
	for P.CheckClosed() {
		err = nil
		BytesRead, err = P.ClientConn.Read(data)
		if err != nil {
			if err == io.EOF {
				Log.Critical("Closing Frontend Handle")
				P.Close()
				return
			} else {
				Log.Critical("Closing because Frontend: ", err)
				P.Close()
				return
			}
		}
		if BytesRead <= 0 {
			Log.Critical("Closing because bytes read is 0!")
			P.Close()
			return
		}
		if BytesRead > 2 {
			PR.SetData(data[:BytesRead])
			PacketSize, _, err = PR.ReadVarInt()
			if err != nil {
				Log.Critical("Could not read packet size!")
			}
			//If compression is set
			if P.GetCompression() > 0 /*&& PacketSize > P.Compression*/ {
				//Data length is uncompressed length of packetID and data
				if PacketSize > P.GetCompression() {
					DataLength, _, err = PR.ReadVarInt()
					if err != nil {
						Log.Debug("Could not read Data Length!", err)
						P.Close()
						return
					}
					if DataLength == 0 {
						Log.Debug("Data length is 0")
					}
					//Log.Debug("Data length: ", DataLength)
					//Read the compressed packet into byte reader
					ByteReader = bytes.NewReader(PR.ReadRestOfByteArray())
					//New zlib reader
					ZlibReader, err := zlib.NewReader(ByteReader)
					if err != nil {
						panic(err)
					}
					//Read decompressed packet
					DecompressedPacket, err := ioutil.ReadAll(ZlibReader)
					if err != nil {
						panic(err)
					}
					//Close ZlibReader
					ZlibReader.Close()
					//Check if size is correct
					if len(DecompressedPacket) != int(DataLength) {
						Log.Debug("Data length != bytes read")
					}
					//Set PacketReader to the decompressed packet
					PR.SetData(DecompressedPacket)
					//Read Packet ID
					PacketID, _, err = PR.ReadVarInt()
					if err != nil {
						Log.Debug("Could not read PacketID!", err)
						P.Close()
						return
					}
				} else { //Not-compressed but compression is used and is below the threshold
					/*DataLength*/ _, _, err = PR.ReadVarInt()
					if err != nil {
						Log.Debug("Could not read Data Length!", err)
						P.Close()
						return
					}
					PacketID, _, err = PR.ReadVarInt()
					if err != nil {
						Log.Debug("Could not read PacketID!", err)
						P.Close()
						return
					}
				}
			} else { //No compression
				PacketID, _, err = PR.ReadVarInt()
				if err != nil {
					Log.Debug("Could not read PacketID!", err)
					P.Close()
					return
				}
			}
			switch P.GetState() {
			case HANDSHAKE:
				switch PacketID {
				case 0x00:
					if P.ProtocolVersion, _, err = PR.ReadVarInt(); err != nil {
						Log.Error(err)
						P.Close()
						return
					}
					Log.Info("PV: ", P.ProtocolVersion)
					if P.ServerAddress, err = PR.ReadString(); err != nil {
						Log.Error(err)
						P.Close()
						return
					}
					_, err := PR.ReadUnsignedShort()
					if err != nil {
						Log.Error(err)
						P.Close()
						return
					}
					NextState, _, err := PR.ReadVarInt()
					if err != nil || NextState > 2 || NextState < 1 {
						Log.Error(err)
						P.Close()
						return
					}
					P.SetState(uint32(NextState))
					Log.Debug("NEXTSTATE: ", P.State)
				}
			case LOGIN:
				switch PacketID {

				}
			case STATUS:
				switch PacketID {
				case 0x00:
					Log.Debug("Recieved Status 0x00 SB")
				case 0x01:
					Log.Debug("Recieved Status 0x01 SB")
				}
			case PLAY:
				switch PacketID {
				case 0x21:
					Log.Critical("Recieved Play Keepalive CB")
				case 0x0F:
					Log.Critical("Recieved Play Keepalive SB 0x0F")
				}
			}
		}
		P.ServerConn.Write(data[:BytesRead])
	}
	Log.Critical("Left loop lmao")
}

func (P *ProxyObject) CheckForLimbo() {
	if config.GConfig.Performance.LimboMode {
		Log.Debug("LIMBO CHECK ACTIVE!")
		ticks := time.Duration(config.GConfig.Performance.CheckServerOnlineTick * 50)
		ticker := time.NewTicker(ticks * time.Millisecond) //time.Tick(ticks * time.Millisecond)
		defer ticker.Stop()
		for P.CheckClosed() {
			<-ticker.C
			if P.GetState() == PLAY {
				PW := CreatePacketWriterWithCapacity(0x0F, 18)
				buf := make([]byte, 8)
				rand.Read(buf) // Always succeeds, no need to check error
				r := binary.LittleEndian.Uint64(buf)
				PW.WriteLong(int64(r))
				_, err := P.ServerConn.Write(PW.GetPacket())
				if err != nil {
					P.Limbo = true
				}
			}
		}
	}
}
