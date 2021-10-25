package proxy

func (P *ProxyObject) HandleFrontEnd() {
	var data = make([]byte, 2097152) //Create buffer to store packet contents in
	var err error
	var BytesRead int
	var PacketID int32
	var PacketSize int32
	var DataLength int32
	var RSD []byte
	PR := CreatePacketReader([]byte{0x00})
	Log.Debug("Started Client Handle")
	P.SetState(HANDSHAKE)
	for P.GetClosed() {
		err = nil
		BytesRead, err = P.ClientConn.Read(data)
		if err != nil {
			Log.Critical("Closing Frontend: ", err, "DATA: ", PR.Data)
			P.Close()
			return
		}
		if BytesRead <= 0 {
			Log.Critical("Closing because bytes read is 0!")
			P.Close()
			return
		}
		PR.SetData(data[:BytesRead])
		PacketSize, DataLength, PacketID, RSD, err = P.HandlePacketHeader(PR)
		PR.SetData(RSD)
		if err != nil {
			panic(err)
		}
		_ = PacketSize
		_ = DataLength
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
				Log.Debug("NEXTSTATE: ", P.GetState())
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
			case 0x01:
				Log.Debug("Recieved Status 0x01 SB")
			}
		case PLAY:
			switch PacketID {
			case 0x1A:
				Log.Critical("Disconnect Play receieved, ignoring")
				SetLimbo(true)
			case 0x05:
				P.Player.Locale, err = PR.ReadString()
				P.Player.ViewDistance, err = PR.ReadByte()
				P.Player.ChatMode, _, err = PR.ReadVarInt()
				P.Player.ChatColours, err = PR.ReadBoolean()
				P.Player.DisplayedSkinParts, err = PR.ReadUnsignedByte()
				P.Player.MainHand, _, err = PR.ReadVarInt()
				P.Player.DisableTextFiltering, err = PR.ReadBoolean()
			case 0x0F:
				Log.Debug("Recieved Play Keepalive SB 0x0F")
				if GetLimbo() {
					Log.Debug("Recieved limbo keepalive")
				}
			}
		}
		if !GetLimbo() && !P.GetReconnection() {
			P.ServerConn.Write(data[:BytesRead])
		}
	}
	Log.Critical("Left loop")
}
