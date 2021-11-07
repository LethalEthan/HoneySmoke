package proxy

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"

	"github.com/google/uuid"
)

type PacketReader struct {
	data   []byte
	seeker int
	end    int
}

//CreatePacketReader - Creates Packet Reader
func CreatePacketReader(data []byte) PacketReader {
	pr := *new(PacketReader)
	pr.data = data
	pr.seeker = 0
	pr.end = len(data)
	return pr
}

//Seek - Seek through the data array
func (pr *PacketReader) Seek(offset int) (int, error) {
	pr.seeker += offset
	return pr.seeker, nil
}

func (pr *PacketReader) SetData(data []byte) {
	pr.seeker = 0
	pr.data = data
	pr.end = len(data)
}

func (pr *PacketReader) GetData() []byte {
	return pr.data
}

func (pr *PacketReader) SeekTo(pos int) bool {
	if pos > pr.end {
		return false
	}
	pr.seeker = pos
	return true
}

func (pr *PacketReader) GetSeek() int {
	return pr.seeker
}

//CheckForEOF
func (pr *PacketReader) CheckForEOF() bool {
	return pr.seeker >= pr.end
}

func (pr *PacketReader) CheckIfEnd() bool {
	return pr.seeker == pr.end
}

func (pr *PacketReader) CheckForEOFWithSeek(SeekTo int) bool {
	if pr.seeker > pr.end {
		return true
	}
	if pr.seeker+SeekTo > pr.end {
		return true
	}
	return false
}

//whence is where the current Seek is and offset is how far the Seek should offset to
func (pr *PacketReader) SeekWithEOF(offset int) (int, error) {
	if offset+pr.seeker > pr.end {
		return offset, errors.New("Seek reached end")
	}
	//Seek after EOF check
	offset, err := pr.Seek(offset)
	if err != nil {
		return offset, err
	}
	return offset, nil
}

//ReadBoolean - reads a single byte from the packet, and interprets it as a boolean.
//It throws an error and returns false if it has a problem either reading from the packet or encounters a value outside of the boolean range.
func (pr *PacketReader) ReadBoolean() (bool, error) {
	bool, err := pr.ReadByte()
	if err != nil {
		return false, err
	}
	switch bool {
	case 0x00:
		return false, nil
	case 0x01:
		return true, nil
	default:
		return false, errors.New("invalid value found in boolean, likely incorrect seek")
	}
}

//ReadByte - reads a single byte from the packet and returns it, it returns a zero and an io.EOF if the packet has been already read to the end.
func (pr *PacketReader) ReadByte() (int8, error) {
	Byte, err := pr.ReadUnsignedByte()
	return int8(Byte), err
}

func (pr *PacketReader) ReadUnsignedByte() (byte, error) {
	if pr.CheckForEOFWithSeek(1) {
		return pr.data[pr.end], nil //return the end byte
		//return 0, errors.New("EOF: UnsignedByte")
	}
	if pr.CheckIfEnd() {
		return pr.data[pr.end], nil
	}
	//Get byte from slice
	Byte := pr.data[pr.seeker]
	//Move the Seek
	_, err := pr.SeekWithEOF(1)
	if err != nil {
		return Byte, err
	}
	return Byte, nil
}

func (pr *PacketReader) ReadShort() (int16, error) {
	short, err := pr.ReadUnsignedShort()
	return int16(short), err
}

func (pr *PacketReader) ReadUnsignedShort() (uint16, error) {
	if pr.CheckForEOFWithSeek(2) {
		return 0, io.EOF
	}
	short := binary.BigEndian.Uint16(pr.data[pr.seeker : pr.seeker+2])
	//Get the 2 bytes that make up the short
	_, err := pr.SeekWithEOF(2)
	if err != nil {
		return 0, err
	}
	return short, nil
}

func (pr *PacketReader) ReadInt() (int32, error) {
	if pr.CheckForEOFWithSeek(4) {
		return 0, io.EOF
	}
	//Get the 4 bytes that make up the int
	Integer := int32(binary.BigEndian.Uint32(pr.data[pr.seeker : pr.seeker+4]))
	//Move the Seek
	_, err := pr.SeekWithEOF(4)
	if err != nil {
		return Integer, err
	}
	return Integer, nil
}

func (pr *PacketReader) ReadLong() (int64, error) {
	if pr.CheckForEOFWithSeek(8) {
		return 0, io.EOF
	}
	//Get the 8 bytes that make up the long
	long := int64(binary.BigEndian.Uint64(pr.data[pr.seeker : pr.seeker+8]))
	//Move the Seek
	_, err := pr.SeekWithEOF(8)
	if err != nil {
		return long, err
	}
	return long, nil
}

func (pr *PacketReader) ReadFloat() (float32, error) {
	if pr.CheckForEOFWithSeek(4) {
		return 0, io.EOF
	}
	//Read the Int
	floatBits, err := pr.ReadInt()
	if err != nil {
		return 0, err
	}
	//Turn the int into float32
	return math.Float32frombits(uint32(floatBits)), nil
}

func (pr *PacketReader) ReadDouble() (float64, error) {
	if pr.CheckForEOFWithSeek(8) {
		return 0, io.EOF
	}
	//Read the long
	doubleBits, err := pr.ReadLong()
	if err != nil {
		return 0, err
	}
	//Turn the long into float64
	return math.Float64frombits(uint64(doubleBits)), nil
}

func (pr *PacketReader) ReadString() (string, error) {
	if pr.CheckForEOF() {
		return "", errors.New("error on begin start string")
	}
	//Read string size
	StringSize, err := pr.ReadVarInt()
	if err != nil {
		return "", err
	}
	if pr.CheckForEOF() {
		return "", io.EOF
	}
	//StringSize check
	if StringSize <= 0 {
		return "", errors.New("string size of %d invalid" + strconv.Itoa(int(StringSize)))
	}
	if int(StringSize) > pr.end {
		return "", errors.New("StringSize exceeds EOF, size: " + strconv.Itoa(int(StringSize)) + "Packet size: " + strconv.Itoa(len(pr.data)))
	}
	if int(StringSize)+pr.seeker > pr.end {
		return "", errors.New("string size + seeker = EOF")
	}
	//Read the string
	StringVal := string(pr.data[pr.seeker : pr.seeker+int(StringSize)])
	//move the Seek
	_, err = pr.SeekWithEOF(int(StringSize))
	if err != nil {
		return StringVal, err
	}
	return StringVal, nil
}

func (pr *PacketReader) ReadVarInt() (int32, error) {
	var Result uint32
	var val uint32
	var NumRead byte
	var Byte byte
	var err error
	Byte, err = pr.ReadUnsignedByte()
	if err != nil {
		return 0, err
	}
	for {
		val = uint32((Byte & 0x7F))
		Result |= (val << (7 * NumRead))
		//Increment
		NumRead++
		//Size check
		if NumRead > 5 {
			return 0, fmt.Errorf("varint was over five bytes without termination")
		}
		//Termination
		if Byte&0x80 == 0 {
			break
		}
		Byte, err = pr.ReadUnsignedByte()
		if err != nil {
			return 0, err //int32(Result), NumRead, err
		}
	}
	return int32(Result), nil
}

func (pr *PacketReader) ReadVarLong() (int64, error) {
	var Result int64
	var val int64
	var Byte byte
	var NumRead byte
	var err error
	Byte, err = pr.ReadUnsignedByte()
	if err != nil {
		return 0, err
	}
	for {
		if err != nil {
			return Result, err
		}
		val = int64((Byte & 0x7F))
		Result |= (val << (7 * NumRead))
		//Increment
		NumRead++
		//Size check
		if NumRead > 10 {
			return 0, fmt.Errorf("varlong was over 10 bytes without termination")
		}
		//Termination
		if Byte&0x80 == 0 {
			break
		}
		Byte, err = pr.ReadUnsignedByte()
	}
	_, err = pr.SeekWithEOF(int(NumRead))
	if err != nil {
		return Result, err
	}
	return Result, nil
}

func (pr *PacketReader) ReadUUID() (uuid.UUID, error) {
	if pr.CheckForEOFWithSeek(16) {
		return uuid.Nil, io.EOF
	}
	UUIDBytes := pr.data[pr.seeker : pr.seeker+16]
	_, err := pr.SeekWithEOF(16)
	if err != nil {
		return uuid.Nil, err
	}
	UUID, err := uuid.FromBytes(UUIDBytes)
	if err != nil {
		return uuid.Nil, err
	}
	return UUID, err
}

//ReadArray - Returns the array (slice) of the packet data
func (pr *PacketReader) ReadByteArray(length int) ([]byte, error) {
	fmt.Print("Current: ", pr.seeker, "len: ", length)
	data := pr.data[pr.seeker : pr.seeker+length]
	fmt.Print("Datalen: ", len(data))
	pr.SeekWithEOF(length)
	fmt.Println("seeker: ", pr.seeker)
	return data, nil
}

func (pr *PacketReader) ReadRestOfByteArray() []byte {
	data := pr.data[pr.seeker:]
	pr.seeker = pr.end
	return data
}
