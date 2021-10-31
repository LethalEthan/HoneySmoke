package proxy

func (P *ProxyObject) HandleFrontEnd() {
	var data = make([]byte, 2097152) //Create buffer to store packet contents in
	var err error
	var BytesRead int
	var PacketID int32
	var RSD []byte
	PR := CreatePacketReader([]byte{0x00})
	Log.Debug("Started Client Handle")
	P.SetState(HANDSHAKE)
	for P.GetClosed() {
		//start:
		err = nil
		if P.GetClosed() && P.ClientConn != nil {
			BytesRead, err = P.ClientConn.Read(data)
			if err != nil {
				Log.Warning("Closing Client")
				Log.Debug("Closing Client reason: ", err)
				P.Close()
				return
			}
		} else {
			Log.Critical("Closing")
			P.Close()
			return
		}
		if BytesRead <= 0 {
			Log.Critical("Closing because bytes read is 0!")
			P.Close()
			return
		}
		PR.SetData(data[:BytesRead])
		_, _, PacketID, RSD, err = P.HandlePacketHeader(PR)
		if err != nil {
			panic(err)
		}
		PR.SetData(RSD)
		switch P.GetState() {
		case HANDSHAKE:
			switch PacketID {
			case 0x00:
				if P.ProtocolVersion, _, err = PR.ReadVarInt(); err != nil {
					Log.Error(err)
					P.Close()
					return
				}
				Log.Debug("PV: ", P.ProtocolVersion)
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
				// if P.ProtocolVersion != 756 {
				// 	Log.Debug("NEXTSTATE: ", P.GetState())
				// 	PW := CreatePacketWriter(0x00)
				// 	PW.WriteVarInt(756)
				// 	PW.WriteString(P.ServerAddress)
				// 	PW.WriteUnsignedShort(ServerPort)
				// 	PW.WriteVarInt(NextState)
				// 	P.ServerConn.Write(PW.GetPacket())
				// 	goto start
				// }
			}
		case LOGIN:
			switch PacketID {
			case 0x00:
				P.Player.Name, err = PR.ReadString()
				if err != nil {
					panic(err)
				}
			case 0x01:
				Log.Debug("Recieved Encrypion Response")
			case 0x02:

			}
		case STATUS:
			switch PacketID {
			case 0x00:
				Log.Debug("Recieved Status 0x00 SB")
				P.ClientConn.Write(data[:BytesRead]) //PR.Data)
				P.Close()
				return
			case 0x01:
				Log.Debug("Recieved Status 0x01 SB")
			}
		case PLAY:
			switch PacketID {
			case 0x1A:
				Log.Critical("Disconnect Play receieved, ignoring")
				SetLimbo(true)
			case 0x05:
				P.Player.Locale, _ = PR.ReadString()
				P.Player.ViewDistance, _ = PR.ReadByte()
				P.Player.ChatMode, _, _ = PR.ReadVarInt()
				P.Player.ChatColours, _ = PR.ReadBoolean()
				P.Player.DisplayedSkinParts, _ = PR.ReadUnsignedByte()
				P.Player.MainHand, _, _ = PR.ReadVarInt()
				P.Player.DisableTextFiltering, _ = PR.ReadBoolean()
			case 0x0F:
				Log.Debug("Recieved Play Keepalive SB 0x0F")
				if GetLimbo() {
					Log.Debug("Recieved limbo keepalive")
				}
			}
		}
		if !GetLimbo() && !P.GetReconnection() && P.GetClosed() {
			P.ServerConn.Write(data[:BytesRead])
		}
	}
	Log.Critical("Left loop")
}
