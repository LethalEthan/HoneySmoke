package proxy

import (
	"HoneySmoke/config"
	"sync/atomic"
	"time"
)

func (P *ProxyObject) HandleFrontEnd() {
	//var data = make([]byte, 2097152) //Create buffer to store packet contents in
	//var RSD []byte
	var err error
	var PacketID int32
	var PPS uint32
	PR := CreatePacketReader([]byte{0x00})
	Log.Debug("Started Client Handle")
	P.SetState(HANDSHAKE)
	go P.PPS()
	for P.GetClosed() {
		//start:
		err = nil
		if P.GetClosed() && P.ClientConn != nil {
			PPS = atomic.LoadUint32(&P.PPSCount)
			if PPS > config.GConfig.Performance.PacketsPerSecond {
				Log.Info("PPS is greater than: ", config.GConfig.Performance.PacketsPerSecond)
			}
			//BytesRead, err = P.ClientConn.Read(data)
			_, _, PacketID, err = P.HandlePacketHeader(P.ClientConn, &PR)
			if err != nil {
				Log.Info("Closing Client")
				Log.Debug("Closing Client reason: ", err)
				P.Close()
				return
			}
		} else {
			Log.Critical("Closing")
			P.Close()
			return
		}
		if len(PR.GetData()) <= 0 {
			Log.Critical("Closing because bytes read is 0!")
			P.Close()
			return
		}
		go func(P *ProxyObject) {
			P.PacketPerSecondC <- 0
		}(P)
		switch P.GetState() {
		case HANDSHAKE:
			switch PacketID {
			case 0x00:
				if P.ProtocolVersion, _, err = PR.ReadVarInt(); err != nil {
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
				_, err := PR.ReadUnsignedShort()
				if err != nil {
					Log.Errorf("SPERR: %d", err)
					P.Close()
					return
				}
				NextState, _, err := PR.ReadVarInt()
				if err != nil || NextState > 2 || NextState < 1 {
					Log.Errorf("NSERR: %d", err)
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
				Log.Debug("Login Disconnect")
			case 0x01:
				Log.Debug("Recieved Encrypion Response")
			case 0x02:

			}
		case STATUS:
			switch PacketID {
			case 0x00:
				Log.Debug("Recieved Status 0x00 SB")
				P.ClientConn.Write(PR.GetData()) //PR.Data)
				P.Close()
				return
			case 0x01:
				Log.Debug("Recieved Status 0x01 SB")
			}
		case PLAY:
			switch PacketID {
			case 0x05:
				if P.ProtocolVersion > 750 {
					P.Player.Locale, _ = PR.ReadString()
					P.Player.ViewDistance, _ = PR.ReadByte()
					P.Player.ChatMode, _, _ = PR.ReadVarInt()
					P.Player.ChatColours, _ = PR.ReadBoolean()
					P.Player.DisplayedSkinParts, _ = PR.ReadUnsignedByte()
					P.Player.MainHand, _, _ = PR.ReadVarInt()
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
			P.ServerConn.Write(PR.GetData())
		}
	}
	Log.Critical("Left loop")
}

func (P *ProxyObject) PPS() {
	ticker := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-ticker.C:
			atomic.StoreUint32(&P.PPSCount, 0)
		case PPS := <-P.PacketPerSecondC:
			switch PPS {
			case 0:
				atomic.AddUint32(&P.PPSCount, 1)
			case 1:
				ticker.Stop()
				return
			}
		}
	}
}
