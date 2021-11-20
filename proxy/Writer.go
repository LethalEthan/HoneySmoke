package proxy

import (
	"encoding/binary"
	"math"

	"github.com/google/uuid"
)

type Identifier string

type PacketWriter struct {
	data     []byte
	packetID int32
}

func CreatePacketWriter(PacketID int32) PacketWriter {
	pw := *new(PacketWriter)       //new packet with data struct Above
	pw.packetID = PacketID         //PacketID passed via function arguments
	pw.data = make([]byte, 0, 128) //Data is created with a byte array
	pw.WriteVarInt(PacketID)       //write PacketID to packet
	return pw
}

//CreatePacketWriterWithCapacity - Create a packet writer with capacity on the data slice
func CreatePacketWriterWithCapacity(PacketID int32, Capacity int) PacketWriter {
	pw := *new(PacketWriter)
	pw.packetID = PacketID
	if Capacity >= 0 {
		pw.data = make([]byte, 0, Capacity)
		pw.WriteVarInt(PacketID)
		return pw
	}
	pw.data = make([]byte, 0, 128)
	pw.WriteVarInt(PacketID)
	return pw
}

func (pw *PacketWriter) GetCompressedPacket() []byte { //Finish this
	if len(pw.data) < 256 {
		p := append(CreateVarInt(uint32(len(pw.data)+1)), 0x00)
		p = append(p, pw.data...)
		return p
	} else {
		panic("PacketSize is bigger than 256")
	}
}

func CreateWriterWithCapacity(Capacity int) PacketWriter {
	pw := *new(PacketWriter)
	if Capacity >= 0 {
		pw.data = make([]byte, 0, Capacity)
		return pw
	} else {
		pw.data = make([]byte, 0, 2048)
		return pw
	}
}

func (pw *PacketWriter) GetData() []byte {
	return pw.data
}

func (pw *PacketWriter) ClearData() {
	pw.data = make([]byte, 0, 4096)
}

func (pw *PacketWriter) ResetData(packetID int32) {
	pw.data = pw.data[:0] //set length to 0, keep cap same
	pw.packetID = packetID
	pw.WriteVarInt(packetID)
}

func (pw *PacketWriter) GetPacket() []byte {
	PacketSize := uint32(len(pw.data))
	p := append(CreateVarInt(PacketSize), pw.data...)
	// Log.Debug("PacketSize: ", len(pw.data))
	// Log.Debug("Packet Contents: ", p)
	return p
}

func (pw *PacketWriter) GetPacketID() int32 {
	return pw.packetID
}

func (pw *PacketWriter) GetPacketSize() int {
	return len(pw.data)
}

func (pw *PacketWriter) AppendByteSlice(Data []byte) {
	pw.data = append(pw.data, Data...)
}

//WriteBoolean - Write Boolean to packet
func (pw *PacketWriter) WriteBoolean(val bool) {
	if val {
		pw.WriteUByte(0x01) //true
	} else {
		pw.WriteUByte(0x00) //false
	}
}

//WriteByte - Write Byte to packet (int8)
func (pw *PacketWriter) WriteByte(val int8) {
	pw.WriteUByte(byte(val))
}

//WriteUnsignedByte - Write Unsigned Byte to packet (uint8)
func (pw *PacketWriter) WriteUByte(val byte) {
	pw.AppendByteSlice([]byte{val})
}

//WriteShort - Write Short to packet (int16)
func (pw *PacketWriter) WriteShort(val int16) {
	pw.WriteUShort(uint16(val))
}

//WriteUShort- Write Unsigned Short to packet (uint16)
func (pw *PacketWriter) WriteUShort(val uint16) {
	buff := make([]byte, 2)
	binary.BigEndian.PutUint16(buff, val)
	pw.AppendByteSlice(buff)
}

//WriteInt - Write Integer to packet (int32)
func (pw *PacketWriter) WriteInt(val int32) {
	pw.WriteUInt(uint32(val))
}

//writeUInt - Write Unsigned Integer to packet (uint32)
func (pw *PacketWriter) WriteUInt(val uint32) {
	buff := make([]byte, 4)
	binary.BigEndian.PutUint32(buff, val)
	pw.AppendByteSlice(buff)
}

//WriteLong - Write Long to packet (int64)
func (pw *PacketWriter) WriteLong(val int64) {
	pw.WriteULong(uint64(val))
}

//writeULong - Write Unsigned Long (unint64)
func (pw *PacketWriter) WriteULong(val uint64) {
	buff := make([]byte, 8)
	binary.BigEndian.PutUint64(buff, val)
	pw.AppendByteSlice(buff)
}

//WriteFloat - Write Float to packet (float32)
func (pw *PacketWriter) WriteFloat(val float32) {
	pw.WriteUInt(math.Float32bits(val))
}

//WriteDouble - Write Double to packet (float64)
func (pw *PacketWriter) WriteDouble(val float64) {
	pw.WriteULong(math.Float64bits(val))
}

//WriteArray - Write an array of bytes ([]byte)
func (pw *PacketWriter) WriteArray(val []byte) {
	pw.AppendByteSlice(val)
}

//WriteString - Write String to packet (string)
func (pw *PacketWriter) WriteString(val string) {
	pw.WriteVarInt(int32(len(val)))
	pw.AppendByteSlice([]byte(val))
}

//WriteString - Write String to packet (string)
func (pw *PacketWriter) WriteStringArray(val []string) {
	for i := range val {
		pw.WriteString(val[i])
	}
}

func (pw *PacketWriter) WriteIdentifier(val Identifier) {
	pw.WriteVarInt(int32(len(val)))
	pw.AppendByteSlice([]byte(val))
}

func (pw *PacketWriter) WriteArrayIdentifier(val []Identifier) {
	for _, v := range val {
		pw.WriteIdentifier(v)
	}
}

//WriteVarInt - Write VarInt to packet (int32)
func (pw *PacketWriter) WriteVarInt(val int32) {
	pw.AppendByteSlice(CreateVarInt(uint32(val)))
}

//WriteVarLong - Write VarLong (int64)
func (pw *PacketWriter) WriteVarLong(val int64) {
	pw.AppendByteSlice(CreateVarLong(uint64(val)))
}

//CreateVarInt - creates VarInt, requires uint to move the sign bit
func CreateVarInt(val uint32) []byte {
	if val > 0 && val < 128 { // values 0-127 are the same and are not encoded in any special way so we skip the logic
		return []byte{byte(val)}
	} else {
		var buff = make([]byte, 0, 5)
		var tmp byte
		for {
			tmp = byte(val & 0x7F)
			val = val >> 7
			if val != 0 {
				tmp |= 0x80
			}
			buff = append(buff, tmp)
			if val == 0 {
				break
			}
			if len(buff) >= 5 {
				Log.Critical("Buff over 5!")
				break
			}
		}
		return buff
	}
}

//CreateVarLong - Creates a VarLong, requires uint to move the sign bit
func CreateVarLong(val uint64) []byte {
	var buff = make([]byte, 0, 10)
	for {
		temp := byte(val & 0x7F)
		val = val >> 7
		if val != 0 {
			temp |= 0x80
		}
		buff = append(buff, temp)
		if val == 0 {
			break
		}
	}
	return buff
}

func (pw *PacketWriter) WriteLongArray(val []int64) {
	for _, v := range val {
		pw.WriteLong(v)
	}
}

func (pw *PacketWriter) WriteUUID(val uuid.UUID) {
	BU, err := val.MarshalBinary()
	if err != nil {
		Log.Debug("Could not marshal UUID!")
	}
	pw.AppendByteSlice(BU)
}

func (pw *PacketWriter) WritePosition(X, Y, Z int64) {
	// if X < 33554432 && Y < 2048 && -Y > 2048 && Z < 33554432 {
	var Location uint64 = ((uint64(X) & 0x3FFFFFF) << 38) | ((uint64(Z) & 0x3FFFFFF) << 12) | (uint64(Y) & 0xFFF)
	pw.WriteULong(Location)
	// } else {
	// 	return
	// }
}

func (pw *PacketWriter) WriteChunkSectionPosition(X, Y, Z int64) {
	var Location uint64 = ((uint64(X) & 0x3FFFFF) << 42) | (uint64(Y) & 0xFFFFF) | ((uint64(Z) & 0x3FFFFF) << 20)
	pw.WriteULong(Location)
}

func (pw *PacketWriter) WriteBlockPosition(BlockState, X, Y, Z uint) {
	Block := BlockState<<12 | (X<<8 | Z<<4 | Y)
	pw.WriteULong(uint64(Block))
}
