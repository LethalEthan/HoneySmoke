package proxy

import (
	"bufio"

	"github.com/google/uuid"
)

func (P *ProxyObject) HandleBackEnd() {
	PR := CreatePacketReader([]byte{0})
	BR := bufio.NewReader(P.ServerConn)
	for P.GetClosed() {
		//start:
		OriginalData, err := DecodePacket(BR, P)
		if err != nil { //&& P.ClientConn != nil && !P.GetClosed() {
			Log.Error(err)
			P.Close()
			return
		}
		/*
			if err != nil {
				if config.GConfig.Performance.LimboMode {
					MainProxy.SetLimbo(true)
					Log.Warning("Server connection lost unexpectedly!")
					Log.Debug("Limbo active")
					//Create keepalive packet
					for P.ClientConn != nil && P.GetClosed() && P.Player.Name != "" {
						//Build handshake packet
						PW := CreatePacketWriter(0x00)
						PW.WriteVarInt(P.ProtocolVersion)
						PW.WriteString(strings.Split(config.GConfig.Proxy.Host, ":")[0])
						Port, _ := strconv.ParseUint(strings.Split(config.GConfig.Proxy.Host, ":")[1], 10, 16)
						PW.WriteUnsignedShort(uint16(Port))
						PW.WriteVarInt(2)
						HP := PW.GetPacket()
					Reconnect:
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
									goto Reconnect
								}
							} else {
								goto Reconnect
							}
							//Login Start
							PW.ResetData(0x00)
							PW.WriteString(P.Player.Name)
							P.ServerConn.Write(PW.GetPacket())
							_, err = P.ServerConn.Read(PR.data)
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
									if _, err := P.ClientConn.Write([]byte{9, 0, 33, 20, 55, 211, 32, 245, 176, 77, 4}); err != nil {
										Log.Error(err)
									}
								} else {
									if _, err := P.ClientConn.Write([]byte{9, 33, 20, 55, 211, 32, 245, 176, 77, 4}); err != nil {
										Log.Error(err)
									}
								}
								Log.Debug("Sent server limbo keepalive")
							}
							time.Sleep(time.Duration(config.GConfig.Performance.CheckServerSeconds) * time.Second)
						}
					}
					P.Close()
					return
				} else {
					P.Close()
					return
				}
			}*/
		PR.SetData(OriginalData)
		_, PacketID, err := P.HandlePacket(&PR)
		if err != nil {
			P.Close()
			return
		}
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
				Log.Infof("Player: %s logged in with UUID: %s", P.Player.Name, P.Player.UUID.String())
			}
		case PLAY:
			switch PacketID {
			case 0x1A:
				if P.ProtocolVersion > 756 {
					Log.Debug("Disconnect Play recieved")
					MainProxy.SetLimbo(true)
				}
			case 0x1B:
				if P.ProtocolVersion == 578 {
					Log.Debug("Disconnect Play receieved")
					MainProxy.SetLimbo(true)
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
					P.ClientConn.Write(OriginalData)
					Log.Debug("Sent pos and look")
					P.SetReconnection(false)
					MainProxy.SetLimbo(false)
				}
			}
		}
		if !MainProxy.GetLimbo() && !P.GetReconnection() && P.GetClosed() {
			P.ClientConn.Write(OriginalData)
		}
	}
}
