package osc

import (
	"bytes"
	"encoding/binary"
	"math"
)

func prependByte(byt []byte, b byte) []byte {
	byt = append(byt, byte(0))
	copy(byt[1:], byt)
	byt[0] = b
	return byt
}

func fixZeroBytes(byt []byte) []byte {
	byt = trimZeroBytes(byt)
	byt = appendZeroBytes(byt)
	return byt
}

func trimZeroBytes(byt []byte) []byte {
	return bytes.Trim(byt, string(byte(0)))
}
func trimZeroBytesRight(byt []byte) []byte {
	return bytes.TrimRight(byt, string(byte(0)))
}

func appendZeroBytes(byt []byte) []byte {
	// Appends zero bytes to the []byte to make it divisible by 4
	//     If b is already divisible by 4, it adds 4 zero bytes
	zeroCount := zeroBytesToAdd(len(byt))
	byt = append(
		byt,
		make([]byte, zeroCount)...,
	)
	return byt
}

func zeroBytesToAdd(n int) int {
	return 4 - (n % 4)
}

func bytesToInt32(b []byte) int32 {
	return int32(binary.BigEndian.Uint32((b)[:]))
}

func bytesToFloat32(b []byte) float32 {
	return math.Float32frombits(binary.BigEndian.Uint32((b)[:]))
}

func float32ToBytes(f float32) []byte {
	var buf [4]byte
	binary.BigEndian.PutUint32(buf[:], math.Float32bits(f))
	return buf[:]
}

func int32ToBytes(i int32) []byte {
	b := []byte{0, 0, 0, 0}
	binary.BigEndian.PutUint32(b[0:4], uint32(i))
	return b
}

func byteToInt32(b []byte) int32 {
	e := binary.BigEndian.Uint32(b[:])
	return int32(e)
}

func byteToFloat32(b []byte) float32 {
	e := binary.BigEndian.Uint32(b[:])
	return math.Float32frombits(e)
}

func allElementsZero(b []byte) bool {
	for i := 0; i < len(b); i++ {
		if b[i] != 0 {
			return false
		}
	}
	return true
}
