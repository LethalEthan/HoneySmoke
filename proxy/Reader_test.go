package proxy

import (
	"fmt"
	"testing"
)

func TestReaderSeekTo(T *testing.T) {
	PR := CreatePacketReader([]byte{0x00, 0xF0, 0x03, 0x09})
	t := PR.SeekTo(4)
	fmt.Println(t)
}
