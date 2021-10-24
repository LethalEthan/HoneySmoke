package proxy

import (
	"fmt"
	"net"
)

func ReadPacket(Conn net.Conn) (int32, int32, []byte, error) {
	PacketSize, PSS, err := ParseVarIntFromConnection(Conn)
	if err != nil {
		return 0, 0, nil, err
	}
	PacketID, PIS, err := ParseVarIntFromConnection(Conn)
	if err != nil {
		return 0, 0, nil, err
	}
	packet := make([]byte, PacketSize-1)
	Conn.Read(packet)
	//Reassemble the full packet - janky but works
	FullPacket := make([]byte, 0, len(packet)+len(PSS)+len(PIS))
	FullPacket = append(FullPacket, PSS...)
	FullPacket = append(FullPacket, PIS...)
	FullPacket = append(FullPacket, packet...)
	Log.Debug("PacketSize: ", PacketSize, "PacketID: ", PacketID)
	return PacketSize - 1, PacketID, FullPacket, nil
}

func ParseVarIntFromConnection(conn net.Conn) (int32, []byte, error) {
	var result int32
	var numRead uint32
	buff := make([]byte, 1)
	BA := make([]byte, 0, 5)
	n, err := conn.Read(buff)
	if err != nil || n != 1 {
		return 0, []byte{0}, err
	}
	for {
		BA = append(BA, buff[0])
		val := int32((buff[0] & 0x7F))
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
