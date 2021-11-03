package proxy

import (
	"bytes"
	"compress/zlib"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
)

var errPacketSizeZero = errors.New("Packet size is 0")

func (P *ProxyObject) HandlePacketHeader(conn net.Conn, PR *PacketReader) (int32, int32, int32, error) {
	var PacketSize int32
	var DataLength int32
	var PacketID int32
	//Read packet size
	PacketSize, PSS, err := ParseVarIntFromConnection(conn) //PR.ReadVarInt()
	if err != nil {
		//Log.Error("Could not read packet size!: ", err, "bound: ", conn.RemoteAddr().String())
		P.Close()
		return 0, 0, 0, err
	}
	if PacketSize < 0 {
		return 0, 0, 0, errPacketSizeZero
	}
	Data := make([]byte, PacketSize)
	_, err = conn.Read(Data)
	if err != nil {
		//Log.Error(err)
		return 0, 0, 0, err
	}
	Data = append(PSS, Data...)
	PR.SetData(Data)
	PR.SeekTo(len(PSS) - 1)
	//If compression is set
	if P.GetCompression() > 0 {
		//Data length is uncompressed length of packetID and data
		DataLength, _, err := PR.ReadVarInt()
		if err != nil {
			Log.Debug("Could not read Data Length!", err)
			P.Close()
			return 0, 0, 0, err
		}
		if DataLength > 0 {
			//Read the compressed packet into byte reader
			ByteReader := bytes.NewReader(PR.ReadRestOfByteArray())
			//New zlib reader
			ZlibReader, err := zlib.NewReader(ByteReader)
			if err != nil {
				Log.Debug("ZLR ", err)
				return 0, 0, 0, err
			}
			//Read decompressed packet
			DecompressedPacket, err := ioutil.ReadAll(ZlibReader)
			if err != nil {
				Log.Debug("DCP ", err)
				return 0, 0, 0, err
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
				return 0, 0, 0, err
			}
			return PacketSize, DataLength, PacketID, nil
		}
		if DataLength == 0 {
			//Not-compressed but compression is used and is below the threshold
			PacketID, _, err = PR.ReadVarInt()
			if err != nil {
				Log.Debug("Could not read PacketID!", err)
				P.Close()
				return 0, 0, 0, err
			}
			return PacketSize, DataLength, PacketID, nil
		}
	} else { //No compression
		PacketID, _, err = PR.ReadVarInt()
		if err != nil {
			Log.Debug("Could not read PacketID!", err)
			P.Close()
			return 0, 0, 0, err
		}
		return PacketSize, 0, PacketID, nil
	}
	return PacketSize, DataLength, PacketID, nil
}

func ParseVarIntFromConnection(conn net.Conn) (int32, []byte, error) {
	var result uint32
	var numRead byte
	buff := make([]byte, 1)
	BA := make([]byte, 0, 5)
	n, err := conn.Read(buff)
	if err != nil || n != 1 {
		return 0, []byte{0}, err
	}
	for {
		BA = append(BA, buff[0])
		val := uint32(buff[0] & 0x7F)
		result |= (val << (7 * numRead))
		numRead++
		//Check length
		if numRead > 5 {
			return 0, []byte{0}, fmt.Errorf("varint was over five bytes without termination")
		}
		//Termination byte
		if buff[0]&0x80 == 0 {
			break
		}
		//Read byte
		n, err := conn.Read(buff)
		if err != nil || n != 1 {
			return int32(result), BA, err
		}
	}
	return int32(result), BA, nil
}

/*
func (P *ProxyObject) HandlePacketHeader(PR *PacketReader) (int32, int32, int32, []byte, error) {
	var PacketSize int32
	var DataLength int32
	var PacketID int32
	//Read packet size
	PacketSize, _, err := PR.ReadVarInt()
	if err != nil {
		Log.Critical("Could not read packet size!")
	}
	//If compression is set
	if P.GetCompression() > 0 {
		//Data length is uncompressed length of packetID and data
		DataLength, _, err = PR.ReadVarInt()
		if err != nil {
			Log.Debug("Could not read Data Length!", err)
			P.Close()
			return 0, 0, 0, []byte{0}, err
		}
		if DataLength > 0 {
			//Read the compressed packet into byte reader
			ByteReader := bytes.NewReader(PR.ReadRestOfByteArray())
			//New zlib reader
			ZlibReader, err := zlib.NewReader(ByteReader)
			if err != nil {
				return 0, 0, 0, []byte{0}, err
			}
			//Read decompressed packet
			DecompressedPacket, err := ioutil.ReadAll(ZlibReader)
			if err != nil {
				return 0, 0, 0, []byte{0}, err
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
				return 0, 0, 0, []byte{0}, err
			}
			return PacketSize, DataLength, PacketID, PR.ReadRestOfByteArray(), err
		}
		if DataLength == 0 {
			//Not-compressed but compression is used and is below the threshold
			PacketID, _, err = PR.ReadVarInt()
			if err != nil {
				Log.Debug("Could not read PacketID!", err)
				P.Close()
				return 0, 0, 0, []byte{0}, err
			}
			return PacketSize, DataLength, PacketID, PR.ReadRestOfByteArray(), err
		}
	} else { //No compression
		PacketID, _, err = PR.ReadVarInt()
		if err != nil {
			Log.Debug("Could not read PacketID!", err)
			P.Close()
			return 0, 0, 0, []byte{0}, err
		}
		return PacketSize, 0, PacketID, PR.ReadRestOfByteArray(), nil
	}
	return PacketSize, DataLength, PacketID, PR.ReadRestOfByteArray(), nil
}*/
