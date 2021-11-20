package proxy

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
)

var BRZ = errors.New("Bytes read is 0")

func DecodePacket(BR *bufio.Reader, P *ProxyObject) ([]byte, error) {
	PacketSize, TMP, err := DecodeVarInt(BR)
	if err != nil {
		Log.Debug(err)
		return []byte{0}, err
	}
	if PacketSize > 2097152 {
		Log.Error("Packet size too large")
		return []byte{0}, errors.New("Packet too large")
	}
	//Log.Debug("PacketSize: ", PacketSize)
	PacketSize = PacketSize + int32(len(TMP)) //add the length of varint to packetsize
	Data := make([]byte, PacketSize)
	copy(Data[0:], TMP) // copy bytes of packet size into buffer
	// read entire packet into buffer
	n, err := io.ReadFull(BR, Data[len(TMP):]) // read the rest of the conn into buffer
	if err != nil {
		Log.Debug(err)
		return []byte{0}, err
	}
	if n <= 0 {
		return []byte{0}, BRZ
	}
	return Data, nil
}

func DecodeVarInt(BR *bufio.Reader) (int32, []byte, error) {
	var result uint32
	var numRead byte
	var tmpbyte byte
	var err error
	BA := make([]byte, 0, 5)
	tmpbyte, err = BR.ReadByte()
	if err != nil {
		Log.Debug(err)
		return 0, []byte{0}, err
	}
	for {
		BA = append(BA, tmpbyte)
		val := uint32(tmpbyte & 0x7F)
		result |= (val << (7 * numRead))
		numRead++
		//Check length
		if numRead > 5 {
			return 0, []byte{0}, fmt.Errorf("varint was over five bytes without termination")
		}
		//Termination byte
		if tmpbyte&0x80 == 0 {
			break
		}
		//Read byte
		tmpbyte, err = BR.ReadByte()
		if err != nil {
			Log.Debug(err)
			return 0, []byte{0}, err
		}
	}
	return int32(result), BA, nil
}

func (P *ProxyObject) HandlePacket(PR *PacketReader) (int32, int32, error) {
	if PR.GetIndex() != 0 {
		panic("seek is not 0 when preparing to handle packet!")
	}
	var PacketSize int32
	var PacketID int32
	var err error
	//Log.Debug("Data: ", PR.data, "Seeker: ", PR.seeker, "End: ", PR.end)
	if P.GetCompression() <= 0 {
		PacketSize, err = PR.ReadVarInt()
		if err != nil {
			return 0, 0, err
		}
		PacketID, err = PR.ReadVarInt()
		if err != nil {
			return 0, 0, err
		}
		return PacketSize, PacketID, nil
	} else {
		var DataSize int32
		PacketSize, err = PR.ReadVarInt()
		if err != nil {
			return 0, 0, err
		}
		DataSize, err = PR.ReadVarInt()
		if err != nil {
			return 0, 0, err
		}
		if DataSize == 0 {
			PacketID, err = PR.ReadVarInt()
			if err != nil {
				return 0, 0, err
			}
			return PacketSize, PacketID, nil
		} else {
			//Read the compressed packet into byte reader
			ByteReader := bytes.NewReader(PR.ReadRestOfByteArrayNoSeek())
			//New zlib reader
			ZlibReader, err := zlib.NewReader(ByteReader)
			if err != nil {
				Log.Debug("ZLR ", err)
				PR.SetData([]byte{0})
				return 0, 0, err
			}
			//Read decompressed packet
			DecompressedPacket, err := ioutil.ReadAll(ZlibReader)
			if err != nil {
				Log.Debug("DCP ", err)
				PR.SetData([]byte{0})
				return 0, 0, err
			}
			//Close ZlibReader
			ZlibReader.Close()
			//Check if size is correct
			if len(DecompressedPacket) != int(DataSize) {
				Log.Debug("Data length != bytes read")
			}
			//Set PacketReader to the decompressed packet
			PR.SetData(DecompressedPacket)
			//Read Packet ID
			PacketID, err = PR.ReadVarInt()
			if err != nil {
				Log.Debug("Could not read PacketID!", err)
				PR.SetData([]byte{0})
				return 0, 0, err
			}
			return PacketSize, PacketID, nil
		}
	}
}
