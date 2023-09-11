package osc

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
)

type Message struct {
	Packet          []byte
	Address         []byte
	TypeTags        []byte
	Arguments       [][]byte
	ArgumentsParsed []any
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

func zeroBytesToAdd(l int) int {
	// The parts of an OSC packet must be divisible by 4 bytes
	zta := 4 - (l % 4)
	// If there are no zero bytes to add, we must pad with 4 zeros
	if zta == 0 {
		zta = 4
	}
	return zta
}

func addZeros(b *[]byte) {
	// The parts of an OSC packet must be divisible by 4 bytes
	*b = append(*b, make([]byte, zeroBytesToAdd(len(*b)))...)
}

func NewMessage(addr string) Message {
	msg := Message{
		Address: []byte(addr),
	}
	return msg
}

func (msg *Message) MakePacket() (err error) {
	var n int
	// We'll use a bytes.Buffer to write to, then dump to msg.Packet
	var buf bytes.Buffer
	// Get the address in bytes and pad with zeros
	addrBytes := []byte(msg.Address)
	addZeros(&addrBytes)
	n, err = buf.Write(addrBytes)
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("could not write AddrBytes")
	}

	// Get the tags in bytes, prefix with comma, and pad with zeros
	typeTagCount := len(msg.TypeTags)
	tagBytes := make([]byte, typeTagCount+1)
	if typeTagCount > 0 {
		tagBytes[0] = ','
	}
	for i := range msg.TypeTags {
		tagBytes[i+1] = msg.TypeTags[i]
	}
	addZeros(&tagBytes)

	n, err = buf.Write(tagBytes)
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("Could not write TagBytes")
	}

	for _, arg := range msg.Arguments {
		n, err = buf.Write([]byte(arg))
		if err != nil {
			return err
		}
		if n == 0 {
			return fmt.Errorf("Could not write ArgBytes")
		}
	}

	// Assign the contents of the buffer to msg.Packet
	msg.Packet = buf.Bytes()

	return nil
}

func (msg *Message) Add(l any) error {
	switch fmt.Sprintf("%T", l) {
	case "int":
		msg.AddInt(int32(l.(int)))
	case "int32":
		msg.AddInt(l.(int32))
	case "int64":
		msg.AddInt(int32(l.(int64)))
	case "float32":
		msg.AddFloat(l.(float32))
	case "float64":
		msg.AddFloat(float32(l.(float64)))
	case "string":
		msg.AddString(l.(string))
	default:
		return fmt.Errorf("Cannot determine type of argument")
	}
	return nil
}

func (msg *Message) AddString(s string) error {
	// Add a s to the typetags and the string to the arguments
	msg.TypeTags = append(msg.TypeTags, 's')

	msg.Arguments = append(msg.Arguments, []byte(s))

	return nil
}

func (msg *Message) AddInt(n int32) error {
	msg.TypeTags = append(msg.TypeTags, 'i')

	// A int32 should consist of four bytes (32 bits)
	intBytes := int32ToBytes(n)
	addZeros(&intBytes) // This will pad with 4 more zero bytes

	msg.Arguments = append(msg.Arguments, intBytes)

	return nil
}

func (msg *Message) AddFloat(data float32) error {
	// A tag consists of a comma character followed by the tags
	msg.TypeTags = append(msg.TypeTags, 'f')

	// A float32 should consist of four bytes (32 bits)
	floatBytes := float32ToBytes(data)
	addZeros(&floatBytes) // This will pad with 4 more zero bytes

	msg.Arguments = append(msg.Arguments, floatBytes)

	return nil
}

func (msg *Message) ParseMessage() error {
	// Parses an OSC message in the form of []byte into a message type as defined above
	var buf bytes.Buffer
	buf.Write(msg.Packet)

	// Read until we hit the comma byte
	addr, err := buf.ReadBytes(byte(','))
	if err != nil {
		return err
	}

	// Remove the comma byte
	addr = bytes.TrimSuffix(addr, []byte(","))

	// Remove any trailing zero bytes
	addr = bytes.TrimRightFunc(addr, func(r rune) bool { return r == rune(0) })

	// Write into our msg.Address
	msg.Address = addr

	// Read the typetag bytes until we hit a zero byte
	typetags, err := buf.ReadBytes(byte(0))
	if err != nil {
		return err
	}

	// Remove the zero byte
	typetags = bytes.TrimSuffix(typetags, []byte{0})

	// Unread the zero byte
	buf.UnreadByte()

	// Write into msg.TypeTags
	msg.TypeTags = typetags

	// We must skip exactly the right amount of zero bytes
	//     as some zero bytes could be used in a float32
	// We'll take the length of typetags plus one indicating the comma we deleted
	//     and find the number of zero bytes to add from there
	z := zeroBytesToAdd(len(typetags) + 1)
	for i := 0; i < z; i++ {
		_, err := buf.ReadByte()
		// If we get to the end of file, break the for loop
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
	}

	// For each type tag, we'll read differently
	for i := range typetags {
		switch msg.TypeTags[i] {
		case 'i', 'f':
			// In the case of a float32 or int32, we'll read 4 bytes
			b := make([]byte, 4)
			n, err := buf.Read(b)
			if err != nil {
				return err
			}
			if n != 4 {
				return fmt.Errorf("invalid data length")
			}
			// Append the int or float as a []byte to msg.Arguments
			msg.Arguments = append(msg.Arguments, b)
			// Skip the next 4 bytes
			//     as there should be zero padding between OSC arguments
			// Create a new []byte... if we use the old one, it will overwrite msg.Arguments
			zb := make([]byte, 4)
			buf.Read(zb)
		case 's':
			fmt.Println("*#*It is a string")
			// In the case of a string, we'll read until we hit a zero byte
			b, err := buf.ReadBytes(byte(0))
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				return err
			}
			// Unread the zero byte delimiter
			err = buf.UnreadByte()
			if err != nil {
				return err
			}
			// Remove the written zero byte from b
			b = bytes.TrimSuffix(b, []byte{0})
			// Append the string as a []byte to msg.Arguments
			msg.Arguments = append(msg.Arguments, b)
			// Find the bytes to skip
			zb := make([]byte, zeroBytesToAdd(len(b)))
			n, err := buf.Read(zb)
			if err != nil {
				return err
			}
			if n != len(zb) {
				return fmt.Errorf("invalid string length when parsing")
			}
		case 'b':
			// TODO: logic for blob
			// For now, we'll just put a warning string in the arguments place
			msg.Arguments = append(msg.Arguments, []byte("cannot yet parse blob"))
		}
	}

	return nil
}

func (msg *Message) DecodeArgument(i int) any {
	// Returns the argument decoded as a float32, int32 or string
	//     when provided the index of the argument in question
	arg := msg.Arguments[i]
	typetag := msg.TypeTags[i]
	switch typetag {
	case 'i':
		// int32 must be of length 4 bytes
		if len(arg) != 4 {
			return nil
		}
		return byteToInt32(arg)
	case 'f':
		// float32 must be of length 4 bytes
		if len(arg) != 4 {
			return nil
		}
		return byteToFloat32(arg)
	case 's':
		return string(arg)
	}
	return nil
}

/*
func (msg *Message) ParseMessage() error {
	var err error

	// If there is no data in the packet bytes buffer, return err
	if msg.Packet.Len() == 0 {
		err = fmt.Errorf("Received Empty Packet")
		return err
	}

	// The OSC Address is the portion before the ','
	//     Write string bytes to msg.Addr until we hit the ','
	msg.Addr, err = msg.Packet.ReadString(',')
	if err != nil {
		return err
	}
	// Trim off the comma we just wrote to msg.Addr
	msg.Addr = strings.TrimSuffix(msg.Addr, ",")
	// Trim off the trailing zeros
	msg.Addr = strings.TrimFunc(msg.Addr, func(r rune) bool {
		return r == 0
	})

	// Tags are single characters indicating the type of data
	//     In the message. 'i': int32, 'f': float32, 's': string
	//     Add the tags until we hit a zero byte
	msg.Tags, err = msg.Packet.ReadString(0)
	// There should be at least one null byte after the tags
	//     to make the tag portion of a length divisible by 4
	//     If already divisible by 4, there will be 4 null bytes
	if err != nil {
		return fmt.Errorf("No null byte following tags")
	}

	// Trim off the zero byte we just wrote to msg.Tags
	msg.Tags = strings.TrimSuffix(msg.Tags, string(0))
	// Unread that zero byte
	msg.Packet.UnreadByte()

	// Inc index over padded zero bytes after the tags
	//   len of tags plus one to account for ','
	msg.Packet.Next(
		zeroBytesToAdd(len(msg.Tags) + 1))

	// If we're out of bounds, exit
	if msg.Packet.Len() == 0 {
		return fmt.Errorf("Out of bounds")
	}

	// We'll make as many iterations as we have tags
	//
	for tagIndex, tag := range msg.Tags {
		msg.Args = append(msg.Args, []byte{})
		if tag == 's' {
			// Read until hit a zero byte
			msg.Args[tagIndex], err = msg.Packet.ReadBytes(0)
			// Trim off the zero byte just written
			msg.Args[tagIndex] = bytes.TrimSuffix(msg.Args[tagIndex], []byte{0})
			msg.Packet.UnreadByte() // Unread that zero byte
			msg.Packet.Next(
				zeroBytesToAdd(len(msg.Args[tagIndex])))
		}
		if tag == 'i' || tag == 'f' {
			ibuf := make([]byte, 4)
			n, err := msg.Packet.Read(ibuf)
			if err != nil {
				return err
			}
			if n != 4 {
				return fmt.Errorf("Didn't read all 32 bits")
			}
			msg.Args[tagIndex] = ibuf
		}
		if tag == 'b' {
			err = fmt.Errorf(
				"Contains a blob\nNot yet sure how to parse")
			return err
		}
	}

	// If there are still nonzero bytes left
	var byt byte
	for err = nil; err == nil; byt, err = msg.Packet.ReadByte() {
		if byt != 0 {
			err = fmt.Errorf("More data than expected")
		}
	}

	return nil
}
*/
