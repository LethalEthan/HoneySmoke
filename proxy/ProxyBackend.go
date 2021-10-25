package proxy

import (
	"HoneySmoke/config"
	"bytes"
	"compress/zlib"
	"crypto/rand"
	"encoding/binary"
	"io/ioutil"
	"net"
	"time"
)

func (P *ProxyObject) ProxyBackEnd() {
	var err error
	for {
		P.ServerConn, err = net.Dial("tcp", config.GConfig.Server.IP+config.GConfig.Server.Port)
		if err == nil {
			Log.Debug("Connected handling...")
			break
		} else {
			Log.Critical("Error dialing, waiting ", config.GConfig.Performance.CheckServerSeconds, " seconds until retry")
			err = nil //reset err otherwise it could just loop
		}
		time.Sleep(time.Duration(config.GConfig.Performance.CheckServerSeconds) * time.Second)
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
	// var RSD []byte
	PR := CreatePacketReader([]byte{0x00})
	P.SetState(HANDSHAKE)
	P.SetCompression(-1)
	Log.Debug("Started Server Handle")
	for P.GetClosed() {
	start:
		err = nil
		BytesRead, err = P.ServerConn.Read(data)
		if err != nil && P.ClientConn != nil {
			Log.Warning("Server connection lost unexpectedly! ", err)
			Log.Warning("Limbo active")
			SetLimbo(true)
			//Create keepalive packer
			KP := CreatePacketWriterWithCapacity(0x21, 18)
			buf := make([]byte, 8)
			rand.Read(buf)
			r := binary.LittleEndian.Uint64(buf)
			KP.WriteLong(int64(r))
			//SetLimbo(true)
			for P.ClientConn != nil && P.GetClosed() {
			Reconnect:
				err = nil //reset err
				//time.Sleep(4 * time.Second)
				P.ServerConn, err = net.Dial("tcp", config.GConfig.Server.IP+config.GConfig.Server.Port)
				if err == nil { //Can connect
					Log.Info("Reconnecting...")
					SetLimbo(false)
					//Handshake
					//P.SetState(HANDSHAKE)
					Log.Debug("Initiating reconnection!")
					PW := CreatePacketWriter(0x00)
					PW.ResetData(0x00)
					PW.WriteVarInt(P.ProtocolVersion)
					PW.WriteString(config.GConfig.ProxyServer.IP)
					PW.WriteShort(25567)
					PW.WriteVarInt(2)
					_, err := P.ServerConn.Write(PW.GetPacket())
					if err != nil {
						P.ServerConn.Close()
						goto Reconnect
					}
					//Login Start
					P.SetState(PLAY)
					//time.Sleep(1 * time.Millisecond)
					PW.ResetData(0x00)
					PW.WriteString(P.Player.Name)
					P.ServerConn.Write(PW.GetPacket())
					_, err = P.ServerConn.Read(data)
					if err != nil {
						P.ServerConn.Close()
						goto Reconnect
					}
					P.SetReconnection(true)
					goto start //Continue from here
				} else {
					Log.Critical("Error dialing, waiting ", config.GConfig.Performance.CheckServerSeconds, " seconds until retry")
				}
				if GetLimbo() {
					if P.GetCompression() > 0 {
						_, err := P.ClientConn.Write(KP.GetCompressedPacket())
						if err != nil {
							Log.Error(err)
							//SendLimbo = false
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
		err = nil
		if BytesRead <= 0 {
			Log.Critical("Closing because bytes read is 0!")
			P.Close()
			return
		}
		PR.SetData(data[:BytesRead])
		//Read packet size
		PacketSize, _, err = PR.ReadVarInt()
		if err != nil {
			Log.Critical("Could not read packet size! ", err)
		}
		_ = PacketSize
		//If compression is set
		if P.GetCompression() > 0 {
			//Data length is uncompressed length of packetID and data
			DataLength, _, err = PR.ReadVarInt()
			if err != nil {
				Log.Debug("Could not read Data Length!", err)
				P.Close()
			}
			if DataLength > 0 {
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
			}
			if DataLength == 0 {
				//Not-compressed but compression is used and is below the threshold
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
		// PR.SetData(data[:BytesRead])
		// PacketSize, DataLength, PacketID, RSD, err = P.HandlePacketHeader(PR)
		// PR.SetData(RSD)
		// if err != nil {
		// 	panic(err)
		// }
		// _ = PacketSize
		_ = DataLength
		switch P.GetState() {
		case STATUS:
			switch PacketID {
			case 0x01:
				P.ClientConn.Write(data[:BytesRead])
				P.Close()
				return
			}
		case LOGIN:
			switch PacketID {
			case 0x02:
				Log.Debug("Login success, setting play state")
				P.Player.UUID, err = PR.ReadUUID()
				if err != nil {
					panic(err)
				}
				P.Player.Name, err = PR.ReadString()
				if err != nil {
					panic(err)
				}
				Log.Debug("UUID: ", P.Player.UUID, "Player: ", P.Player.Name)
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
			case 0x05:
				P.Player.Locale, err = PR.ReadString()
				P.Player.ViewDistance, err = PR.ReadByte()
				P.Player.ChatMode, _, err = PR.ReadVarInt()
				P.Player.ChatColours, err = PR.ReadBoolean()
				P.Player.DisplayedSkinParts, err = PR.ReadUnsignedByte()
				P.Player.MainHand, _, err = PR.ReadVarInt()
				P.Player.DisableTextFiltering, err = PR.ReadBoolean()
				if err != nil {
					Log.Error(err)
				}
			case 0x1A:
				Log.Critical("Disconnect Play receieved, ignoring")
				SetLimbo(true)
			case 0x21:
				Log.Debug("Sending Keepalive CB 0x21")
			case 0x26:
				if config.GConfig.ProxyServer.DEBUG {
					Log.Debug("Sending JoinGame 0x26")
				}
			case 0x32:
				if P.Reconnection {
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
			case 0x38:
				if P.GetReconnection() {
					Log.Debug("Sending player pos and look")
					P.ClientConn.Write(PR.Data)
					Log.Debug("Sent pos and look")
					P.SetReconnection(false)
				}
			}
		}
		err = nil
		if !GetLimbo() && !P.GetReconnection() {
			P.ClientConn.Write(data[:BytesRead])
		}
	}
	Log.Critical("Left loop")
}
