package osc

import (
	"bytes"
	"errors"
	"fmt"
	"io"
)

type Message struct {
	Packet    bytes.Buffer
	Address   []byte
	TypeTags  []byte
	Arguments []argument
}

type argument struct {
	Type    byte
	Data    []byte
	Decoded any
}

func NewMessage(addr string) Message {
	msg := Message{
		Address: []byte(addr),
	}
	return msg
}

func (msg *Message) MakePacket() (err error) {
	// Ensure msg.Packet is empty
	msg.Packet.Reset()

	// Ensure correct count of zero bytes appended to address, and write to packet
	// Each part of an OSC Message must be divisible by 4 bytes,
	//     and if it already is, it must be padded with 4 more zero bytes
	_, err = msg.Packet.Write(fixZeroBytes(msg.Address))
	if err != nil {
		return err
	}

	// If there are type tags, write them to the packet with a leading comma
	if len(msg.TypeTags) > 0 {
		_, err := msg.Packet.Write(
			fixZeroBytes(prependByte(msg.TypeTags, byte(','))))
		if err != nil {
			return err
		}
	}

	// For each argument, write to packet
	for _, arg := range msg.Arguments {
		switch arg.Type {
		// String should be suffixed with correct count of zero bytes
		case 's':
			arg.Data = fixZeroBytes(arg.Data)
		// Int32 and Float32 should always be of length 4 bytes
		case 'i':
			if len(arg.Data) != 4 {
				return fmt.Errorf("int32 not of length 4 bytes")
			}
		case 'f':
			if len(arg.Data) != 4 {
				return fmt.Errorf("float32 not of length 4 bytes")
			}
		}
		_, err = msg.Packet.Write(arg.Data)
		if err != nil {
			return err
		}
	}

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

func (msg *Message) AddString(s string) {
	msg.TypeTags = append(msg.TypeTags, 's')

	msg.Arguments = append(msg.Arguments,
		argument{
			Type:    's',
			Data:    []byte(s),
			Decoded: s})
}

func (msg *Message) AddInt(x int32) error {
	msg.TypeTags = append(msg.TypeTags, 'i')

	msg.Arguments = append(msg.Arguments,
		argument{
			Type:    'i',
			Data:    int32ToBytes(x),
			Decoded: x})

	return nil
}

func (msg *Message) AddFloat(x float32) error {
	msg.TypeTags = append(msg.TypeTags, 'f')

	msg.Arguments = append(msg.Arguments,
		argument{
			Type:    'f',
			Data:    float32ToBytes(x),
			Decoded: x})

	return nil
}

func (msg *Message) ParseMessage() (err error) {
	// Parses an OSC message in the form of []byte into a message type as defined above

	// Read into msg.Address until we hit the comma byte
	msg.Address, err = msg.Packet.ReadBytes(byte(','))
	if err != nil {
		return err
	}

	// Remove the comma byte
	msg.Address = bytes.TrimSuffix(msg.Address, []byte(","))

	// Remove any trailing zero bytes
	//     which may have been read before the comma byte
	msg.Address = trimZeroBytesRight(msg.Address)

	// Read the typetag bytes until we hit a zero byte
	msg.TypeTags, err = msg.Packet.ReadBytes(byte(0))
	if err != nil {
		return err
	}

	// Remove the zero byte
	msg.TypeTags = trimZeroBytesRight(msg.TypeTags)

	// Unread the zero byte
	msg.Packet.UnreadByte()

	// Skip the zero bytes which trail the typetags portion
	//    Dont skip too many zero bytes
	//     as some could be used in a following float32 osc argument
	// Find the length of msg.TypeTags plus one (the removed comma)
	// Find the number of zero bytes that should trail the type tags
	// Skip those zero bytes
	zerosToSkip := zeroBytesToAdd(len(msg.TypeTags) + 1)
	_, err = msg.Packet.Read(make([]byte, zerosToSkip))
	// Only return if the err is not io.EOF
	if err != nil && !errors.Is(err, io.EOF) {
		return err
	}

	// Continue to read the osc arguments using the type tags as a map
	//     For each type tag:
	//         read the following bytes and append to msg.Arguments
	for _, typeTag := range msg.TypeTags {
		switch typeTag {
		case 'i', 'f':
			// In the case of a float32 or int32, read 4 bytes
			byt := make([]byte, 4)
			_, err := msg.Packet.Read(byt)
			if err != nil {
				return err
			}
			// Append the data as a []byte to msg.Arguments
			msg.Arguments = append(msg.Arguments,
				argument{
					Type:    typeTag,
					Data:    byt,
					Decoded: decodeArgument(byt, typeTag)})
			// Skip the next 4 bytes
			//     as there should be zero padding between OSC arguments
			// Create a new []byte... if we use the old one, it will overwrite msg.Arguments
			zbyt := make([]byte, 4)
			_, err = msg.Packet.Read(zbyt)
			if err != nil {
				return err
			}
		case 's':
			// In the case of a string, read until we hit a zero byte
			byt, err := msg.Packet.ReadBytes(byte(0))
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				return err
			}
			// Unread the zero byte delimiter
			err = msg.Packet.UnreadByte()
			if err != nil {
				return err
			}
			// Remove the written zero byte from b
			byt = trimZeroBytesRight(byt)
			// Append the string as a []byte to msg.Arguments
			msg.Arguments = append(msg.Arguments,
				argument{
					Type:    typeTag,
					Data:    byt,
					Decoded: decodeArgument(byt, typeTag)})
			// Find the bytes to skip
			zerosToSkip := make([]byte, zeroBytesToAdd(len(byt)))
			_, err = msg.Packet.Read(zerosToSkip)
			if err != nil {
				return err
			}
		case 'b':
			// blob is "an int32 size count, followed by that many 8-bit bytes of arbitrary binary data"
			b := make([]byte, 4)
			_, err := msg.Packet.Read(b)
			if err != nil {
				return err
			}
			sizeCount, ok := decodeArgument(b, 'i').(int32)
			if !ok {
				return fmt.Errorf("cannot parse arguments")
			}
			data := make([]byte, sizeCount)
			_, err = msg.Packet.Read(data)
			if err != nil {
				return err
			}
			// Append the data as a []byte to msg.Arguments
			msg.Arguments = append(msg.Arguments,
				argument{
					Type:    typeTag,
					Data:    data,
					Decoded: nil})
			// Find the bytes to skip
			zerosToSkip := make([]byte, zeroBytesToAdd(len(data)))
			_, err = msg.Packet.Read(zerosToSkip)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func decodeArgument(byt []byte, typeTag byte) any {
	// Returns the []byte decoded as a float32, int32 or string
	switch typeTag {
	case 'i':
		// int32 must be of length 4 bytes
		if len(byt) != 4 {
			return nil
		}
		return byteToInt32(byt)
	case 'f':
		// float32 must be of length 4 bytes
		if len(byt) != 4 {
			return nil
		}
		return byteToFloat32(byt)
	case 's':
		return string(byt)
	}
	return nil
}

func (msg *Message) DecodeArgument(i int) any {
	// Return the argument decoded as a float32, int32 or string
	//     when provided the index of the argument in the *Message
	return decodeArgument(msg.Arguments[i].Data, msg.TypeTags[i])
}

func (msg *Message) DecodeArguments() {
	// Decode all arguments as a float32, int32 or string
	//     And assigns them to Message.ArgumentsDecoded
	for i := range msg.Arguments {
		msg.Arguments[i].Decoded = msg.DecodeArgument(i)
	}
}
