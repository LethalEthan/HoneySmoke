package proxy

import (
	"HoneySmoke/config"
	"crypto/rand"
	"encoding/binary"
	"net"
	"strings"
	"time"
)

func (P *ProxyObject) ProxyBackEnd() {
	var err error
	for {
		P.ServerConn, err = net.Dial("tcp", config.GConfig.Backends.Servers[0])
		if err == nil {
			Log.Debug("Connected handling...")
			break
		} else {
			Log.Critical("Error dialing, waiting", config.GConfig.Performance.CheckServerSeconds, "seconds until retry")
			err = nil //reset err otherwise it could just loop
		}
		time.Sleep(time.Duration(config.GConfig.Performance.CheckServerSeconds) * time.Second)
	}
	go P.HandleFrontEnd()
	go P.HandleBackEnd()
}

func (P *ProxyObject) HandleBackEnd() {
	var err error
	var PacketID int32
	PR := CreatePacketReader([]byte{0})
	P.SetState(HANDSHAKE)
	P.SetCompression(-1)
	Log.Debug("Started Server Handle")
	for P.GetClosed() {
	start:
		err = nil
		if P.GetClosed() && P.ServerConn != nil {
			//BytesRead, err = P.ServerConn.Read(data)
			_, _, PacketID, err = P.HandlePacketHeader(P.ServerConn, &PR)
		} else {
			Log.Warning("Closing backend handler") //Probably do a goto reconnect
			return
		}
		if err != nil && P.ClientConn != nil && P.State == PLAY && P.Player.Name != "" && config.GConfig.Performance.LimboMode {
			Log.Warning("Server connection lost unexpectedly! ", err)
			Log.Debug("Limbo active")
			MainProxy.SetLimbo(true)
			//Create keepalive packet
			KP := CreatePacketWriterWithCapacity(0x21, 18)
			buf := make([]byte, 8)
			rand.Read(buf)
			r := binary.LittleEndian.Uint64(buf)
			KP.WriteLong(int64(r))
			for P.ClientConn != nil && P.GetClosed() && config.GConfig.Performance.LimboMode {
			Reconnect:
				err = nil //reset err
				P.ServerConn, err = net.Dial("tcp", config.GConfig.Backends.Servers[0])
				if err == nil && P.Player.Name != "" { //Can connect
					Log.Info("Reconnecting...")
					MainProxy.SetLimbo(false)
					Log.Debug("Initiating reconnection!")
					PW := CreatePacketWriter(0x00) //Handshake packet
					PW.ResetData(0x00)
					PW.WriteVarInt(P.ProtocolVersion)
					PW.WriteString(strings.Split(config.GConfig.Proxy.Host, ":")[0])
					Log.Debug(strings.Split(config.GConfig.Proxy.Host, ":")[0])
					PW.WriteShort(25567)
					PW.WriteVarInt(2)
					if P.ServerConn != nil {
						_, err := P.ServerConn.Write(PW.GetPacket())
						if err != nil {
							P.ServerConn.Close()
							goto Reconnect
						}
					} else {
						goto Reconnect
					}
					//Login Start
					P.SetState(PLAY) //Client is in play state so just make sure it's still set
					PW.ResetData(0x00)
					PW.WriteString(P.Player.Name)
					P.ServerConn.Write(PW.GetPacket())
					_, err = P.ServerConn.Read(PR.data) //data)
					if err != nil {
						P.ServerConn.Close()
						goto Reconnect
					}
					P.SetReconnection(true)
					MainProxy.SetLimbo(false)
					goto start //Continue from here
				} else {
					Log.Debug("Error dialing, waiting", config.GConfig.Performance.CheckServerSeconds, "seconds until retry...")
					if MainProxy.GetLimbo() {
						if P.GetCompression() > 0 {
							_, err := P.ClientConn.Write(KP.GetCompressedPacket())
							if err != nil {
								Log.Error(err)
							}
						} else {
							_, err := P.ClientConn.Write(KP.GetPacket())
							if err != nil {
								Log.Error(err)
							}
							Log.Debug(("Sent limbo non compressed"))
						}
						Log.Debug("Sent server limbo keepalive")
					}
					time.Sleep(time.Duration(config.GConfig.Performance.CheckServerSeconds) * time.Second)
				}
			}
		} else {
			if err != nil && P.ClientConn == nil && !P.GetClosed() {
				Log.Debug("123: ", err)
				P.Close()
				return
			}
		}
		err = nil
		// if len(PR.data) > 2097152 {
		// 	Log.Critical("Packet size is greater than 2097152!")
		// 	P.Close()
		// 	return
		// }
		switch P.GetState() {
		case STATUS:
			switch PacketID {
			case 0x01:
				P.ClientConn.Write(PR.data)
				P.Close()
				return
			}
		case LOGIN:
			switch PacketID {
			case 0x02:
				Log.Debug("Login success, setting play state")
				if P.ProtocolVersion > 760 {
					P.Player.UUID, err = PR.ReadUUID()
					if err != nil {
						Log.Error("Error reading UUID: ", err)
					}
					P.Player.Name, err = PR.ReadString()
					if err != nil {
						Log.Error("Error reading player name: ", err)
					}
					Log.Debug("UUID: ", P.Player.UUID, "Player: ", P.Player.Name)
				}
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
			case 0x1A:
				Log.Critical("Disconnect Play receieved")
				if config.GConfig.Performance.LimboMode {
					MainProxy.SetLimbo(true)
				}
			case 0x1B:
				if P.ProtocolVersion == 578 {
					Log.Critical("Disconnect Play receieved")
					//SetLimbo(true)
				}
			case 0x21:
				Log.Debug("Sending Keepalive CB 0x21")
			case 0x26:
				Log.Debug("Sending JoinGame 0x26")
			case 0x32:
				if P.GetReconnection() {
					PW := CreatePacketWriter(0x05)
					PW.WriteString(P.Player.Locale)
					PW.WriteByte(P.Player.ViewDistance)
					PW.WriteVarInt(P.Player.ChatMode)
					PW.WriteBoolean(P.Player.ChatColours)
					PW.WriteUnsignedByte(P.Player.DisplayedSkinParts)
					PW.WriteBoolean(P.Player.DisableTextFiltering)
					if P.GetCompression() > 0 {
						Log.Debug("Sending COP")
						P.ServerConn.Write(PW.GetCompressedPacket())
					} else {
						Log.Debug("Sending UNP")
						P.ServerConn.Write(PW.GetPacket())
					}
				}
			case 0x36:
				if P.GetReconnection() {
					Log.Debug("Sending player pos and look")
					P.ClientConn.Write(PR.data)
					Log.Debug("Sent pos and look")
					P.SetReconnection(false)
					MainProxy.SetLimbo(false)
				}
			}
		}
		err = nil
		if !MainProxy.GetLimbo() && !P.GetReconnection() && P.GetClosed() {
			P.ClientConn.Write(PR.data)
		}
	}
	Log.Critical("Left loop")
}
