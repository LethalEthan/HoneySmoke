package proxy

import (
	"HoneySmoke/config"
	"bufio"
	"strings"
)

func (P *ProxyObject) HandleFrontEnd() {
	PR := CreatePacketReader([]byte{0})
	BR := bufio.NewReader(P.ClientConn)
	for P.GetClosed() {
		OriginalData, err := DecodePacket(BR, P)
		if err != nil {
			P.Close()
			return
		}
		PR.SetData(OriginalData)
		_, PacketID, err := P.HandlePacket(&PR)
		if err != nil {
			P.Close()
			return
		}
		Log.Debugf("FE: %x, S: %d", PacketID, P.GetState())
		switch P.State {
		case HANDSHAKE:
			switch PacketID {
			case 0x00:
				if P.ProtocolVersion, err = PR.ReadVarInt(); err != nil {
					Log.Errorf("PVERR: %d", err)
					P.Close()
					return
				}
				Log.Debug("PV: ", P.ProtocolVersion)
				if P.ServerAddress, err = PR.ReadString(); err != nil {
					Log.Errorf("SAERR: %d", err)
					P.Close()
					return
				}
				Port, err := PR.ReadUShort()
				if err != nil {
					Log.Errorf("SPERR: %d", err)
					P.Close()
					return
				}
				NextState, err := PR.ReadVarInt()
				if err != nil || NextState > 2 || NextState < 1 {
					Log.Errorf("NSERR: %d", err)
					P.Close()
					return
				}
				PW := CreatePacketWriter(0x00)
				PW.WriteVarInt(P.ProtocolVersion)
				PW.WriteString(strings.Split(config.GConfig.Backends.Servers[0], ":")[0])
				PW.WriteUShort(Port)
				PW.WriteVarInt(NextState)
				OriginalData = PW.GetPacket() //Overwrite original handshake
				P.SetState(uint32(NextState))
			}
		case STATUS:
			switch PacketID {

			}
		case LOGIN:
			switch PacketID {

			}
		case PLAY:
			switch PacketID {
			case 0x05:
				if P.ProtocolVersion > 750 {
					Log.Debug("Set player object values")
					P.Player.Locale, _ = PR.ReadString()
					P.Player.ViewDistance, _ = PR.ReadByte()
					P.Player.ChatMode, _ = PR.ReadVarInt()
					P.Player.ChatColours, _ = PR.ReadBoolean()
					P.Player.DisplayedSkinParts, _ = PR.ReadUByte()
					P.Player.MainHand, _ = PR.ReadVarInt()
					P.Player.DisableTextFiltering, _ = PR.ReadBoolean()
				}
			case 0x0F:
				Log.Debug("Recieved Play Keepalive SB 0x0F")
				if MainProxy.GetLimbo() {
					Log.Debug("Recieved limbo keepalive")
				}

			}
		}
		if !MainProxy.GetLimbo() && !P.GetReconnection() && P.GetClosed() {
			P.ServerConn.Write(OriginalData)
		}
	}
	Log.Debug("Frontend closed")
}
