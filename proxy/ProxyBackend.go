package proxy

import (
	"HoneySmoke/config"
	"bufio"
	"net"
	"time"

	"github.com/google/uuid"
)

func (P *ProxyObject) HandleBackEnd() {
	PR := CreatePacketReader([]byte{0})
	BR := bufio.NewReader(P.ServerConn)
	for P.GetClosed() {
	start:
		OriginalData, err := DecodePacket(BR, P)
		if err != nil {
			Log.Debug(err)

		}
		if err != nil && !P.GetClosed() {
			Log.Debug(err)
			P.Close()
			return
		}
		if err != nil && P.Player.Name != "" {
			if config.GConfig.Performance.LimboMode && P.GetClosed() {
				Log.Debug("Limbo seg")
				P.BEReconnect(&PR)
				goto start
			}
		}
		PR.SetData(OriginalData)
		_, PacketID, err := P.HandlePacket(&PR)
		if err != nil {
			P.Close()
			return
		}
		Log.Debugf("BE: %x, S: %d", PacketID, P.GetState())
		switch P.State {
		case STATUS:
			switch PacketID {
			case 0x01:
				P.ClientConn.Write(OriginalData)
				P.Close()
				return
			}
		case LOGIN:
			switch PacketID {
			case 0x02:
				Log.Debug("Recieved login success!")
				if P.ProtocolVersion > 700 {
					P.Player.UUID, _ = PR.ReadUUID()
					P.Player.Name, _ = PR.ReadString()
				} else {
					tmp, _ := PR.ReadString()
					P.Player.UUID, _ = uuid.Parse(tmp)
					P.Player.Name, _ = PR.ReadString()
				}
				Log.Debugf("Player: %s logged in with UUID: %s", P.Player.Name, P.Player.UUID.String())
				UB, _ := P.Player.UUID.MarshalBinary()
				Log.Debug("UUID in bytes: ", UB)
				P.SetState(PLAY)
			case 0x03:
				if P.GetCompression() <= 0 {
					Log.Debug("Set Compression recieved")
					Compression, err := PR.ReadVarInt()
					P.SetCompression(Compression)
					if err != nil {
						Log.Critical("Could not read compression threshold!")
					}
					Log.Debug("Compression threshold: ", Compression)
				}
			}
		case PLAY:
			switch PacketID {
			case 0x02:
				Log.Debug("Spawn Living Entity: ", OriginalData)
				EID, _ := PR.ReadVarInt()
				UUID, _ := PR.ReadUUID()
				Type, _ := PR.ReadVarInt()
				UB, _ := UUID.MarshalBinary()
				Log.Debugf("EID: %d UUID: %x Type %d", EID, UB, Type)
			case 0x14:
				Log.Debug("Window Items: ", OriginalData)
			case 0x18:
				Log.Debug("Channel message: ", OriginalData)
			case 0x1A:
				if P.ProtocolVersion >= 756 {
					Log.Debug("Disconnect Play recieved")
					MainProxy.SetLimbo(true)
					P.BEReconnect(&PR)
				}
			case 0x1B:
				if P.ProtocolVersion == 578 {
					Log.Debug("Disconnect Play receieved")
					MainProxy.SetLimbo(true)
				} else {
					Log.Debug("Entity Status 0x1B")
					EID, err := PR.ReadInt()
					if err != nil {
						Log.Error(err)
					}
					ES, err := PR.ReadByte()
					if err != nil {
						Log.Error(err)
					}
					Log.Debugf("EID: %d, ES: %08b - %d", EID, ES, ES)
				}
			case 0x20:
				Log.Debug("Initialise World Border", OriginalData)
			case 0x21:
				Log.Debug("Sending Keepalive CB 0x21")
				Long, _ := PR.ReadLong()
				Log.Debug(time.UnixMilli(Long))
			case 0x26:
				Log.Debug("Sending JoinGame 0x26")
				// Log.Debug("JoinGame: ", OriginalData)
				// P.Close()
			case 0x32:
				if P.GetReconnection() {
					PW := CreatePacketWriter(0x05)
					PW.WriteString(P.Player.Locale)
					PW.WriteByte(P.Player.ViewDistance)
					PW.WriteVarInt(P.Player.ChatMode)
					PW.WriteBoolean(P.Player.ChatColours)
					PW.WriteUByte(P.Player.DisplayedSkinParts)
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
					P.ClientConn.Write(OriginalData)
					Log.Debug("Sent pos and look")
					P.SetReconnection(false)
					MainProxy.SetLimbo(false)
				}
				Log.Debug("Tab: ", OriginalData)
			case 0x39:
				Log.Debug("Unlock Recipes: ", OriginalData)
			case 0x22:
				//Log.Debug("ChunkData: ", OriginalData)
			case 0x55:
				Log.Debug("Teams: ", OriginalData)
			case 0x58:
				WorldAge, _ := PR.ReadLong()
				TimeOfDay, _ := PR.ReadLong()
				Log.Debug("Time Update: ", WorldAge, "TOD:", TimeOfDay)
			case 0x4D:
				Log.Debug("Entity metadata: ", OriginalData)

			case 0x66:
				Log.Debug("Tags: ", OriginalData)
			}
		}
		if !MainProxy.GetLimbo() && !P.GetReconnection() && P.GetClosed() {
			P.ClientConn.Write(OriginalData)
		}
	}
}

func (P *ProxyObject) BEReconnect(PR *PacketReader) {
	var err error
	MainProxy.SetLimbo(true)
	Log.Warning("Server connection lost unexpectedly!")
	//Build handshake packet
	PW := CreatePacketWriter(0x00)
	PW.WriteVarInt(P.ProtocolVersion)
	PW.WriteString(MainProxy.IP)
	PW.WriteUShort(MainProxy.Port)
	PW.WriteVarInt(2)
	HP := PW.GetPacket() //Store the packet
	for P.GetClosed() {
	start:
		err = nil //reset err
		P.ServerConn, err = net.Dial("tcp", config.GConfig.Backends.Servers[0])
		if err == nil { //Can connect
			Log.Info("Reconnecting...")
			MainProxy.SetLimbo(false)
			Log.Debug("Initiating reconnection!")
			if P.ServerConn != nil {
				_, err := P.ServerConn.Write(HP) //Send Handshake packet
				if err != nil {
					P.ServerConn.Close()
					goto start
				}
				//Login Start
				PW.ResetData(0x00)
				PW.WriteString(P.Player.Name)
				P.ServerConn.Write(PW.GetPacket())
				_, err = P.ServerConn.Read(PR.data)
				if err != nil {
					P.ServerConn.Close()
					goto start
				}
				P.SetReconnection(true)
				MainProxy.SetLimbo(false)
				break
			}
		} else {
			Log.Debug("Error dialing, waiting", config.GConfig.Performance.CheckServerSeconds, "seconds until retry...")
			if MainProxy.GetLimbo() {
				if P.GetCompression() > 0 {
					if _, err := P.ClientConn.Write([]byte{9, 0, 33, 20, 55, 211, 32, 245, 176, 77, 4}); err != nil {
						Log.Error(err)
						P.Close()
						return
					}
				} else {
					if _, err := P.ClientConn.Write([]byte{9, 33, 20, 55, 211, 32, 245, 176, 77, 4}); err != nil {
						Log.Error(err)
						P.Close()
						return
					}
				}
				Log.Debug("Sent server limbo keepalive")
			}
			time.Sleep(time.Duration(config.GConfig.Performance.CheckServerSeconds) * time.Second)
		}
	}
}
