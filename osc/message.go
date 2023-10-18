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
	Arguments []argument
}

type argument struct {
	TypeTag byte
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

	// If there are arguments, write the type tags to the packet with a leading comma
	if len(msg.Arguments) > 0 {
		if err := msg.Packet.WriteByte(','); err != nil {
			return err
		}
	}
	for _, arg := range msg.Arguments {
		if err := msg.Packet.WriteByte(arg.TypeTag); err != nil {
			return err
		}
	}
	if len(msg.Arguments) > 0 {
		zerosToAdd := zeroBytesToAdd(len(msg.Arguments) + 1)
		for i := 0; i < zerosToAdd; i++ {
			msg.Packet.WriteByte(0)
		}
	}

	// For each argument, write to packet
	for _, arg := range msg.Arguments {
		switch arg.TypeTag {
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

func (msg *Message) AddString(s string) {
	msg.Arguments = append(msg.Arguments,
		argument{
			TypeTag: 's',
			Data:    []byte(s),
			Decoded: s})
}

func (msg *Message) AddInt(x int32) {
	msg.Arguments = append(msg.Arguments,
		argument{
			TypeTag: 'i',
			Data:    int32ToBytes(x),
			Decoded: x})
}

func (msg *Message) AddFloat(x float32) {
	msg.Arguments = append(msg.Arguments,
		argument{
			TypeTag: 'f',
			Data:    float32ToBytes(x),
			Decoded: x})
}

func (msg *Message) ParseMessage() (err error) {
	// Parses an OSC message in the form of []byte into a message type as defined above

	// Read into msg.Address until the comma byte
	msg.Address, err = msg.Packet.ReadBytes(byte(','))
	// If EOF, return with no error
	if err == io.EOF {
		return nil
	}
	if err != nil {
		return err
	}

	// Remove the comma byte
	msg.Address = bytes.TrimSuffix(msg.Address, []byte(","))

	// Remove any trailing zero bytes
	//     which may have been read before the comma byte
	msg.Address = trimZeroBytesRight(msg.Address)

	// For each type tag, append a new argument
	var typeTag byte
	for {
		typeTag, err = msg.Packet.ReadByte()
		if err != nil {
			return err
		}
		// When we hit a zero byte, break the loop
		if typeTag == byte(0) {
			break
		}
		msg.Arguments = append(msg.Arguments, argument{TypeTag: typeTag})
	}

	// Unread the zero byte
	msg.Packet.UnreadByte()

	// Skip the zero bytes which trail the typetags portion
	//    Dont skip too many zero bytes
	//     as some could be used in a following float32 osc argument
	// Find the length of msg.Arguments plus one (the removed comma)
	// Find the number of zero bytes that should trail the type tags
	// Skip those zero bytes
	zerosToSkip := zeroBytesToAdd(len(msg.Arguments) + 1)
	_, err = msg.Packet.Read(make([]byte, zerosToSkip))
	// Only return if the err is not io.EOF
	if err != nil && !errors.Is(err, io.EOF) {
		return err
	}

	// Continue to read the osc arguments using the type tags as a map
	//     For each type tag:
	//         read the following bytes and append to msg.Arguments
	for i := range msg.Arguments {
		switch msg.Arguments[i].TypeTag {
		case 'i', 'f':
			// In the case of a float32 or int32, read 4 bytes
			byt := make([]byte, 4)
			_, err := msg.Packet.Read(byt)
			if err != nil {
				return err
			}

			// Set the data and decoded data for each argument
			msg.Arguments[i].Data = byt
			msg.Arguments[i].Decoded = decodeArgument(byt, msg.Arguments[i].TypeTag)

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

			// Set the data and decoded data for each argument
			msg.Arguments[i].Data = byt
			msg.Arguments[i].Decoded = decodeArgument(byt, msg.Arguments[i].TypeTag)

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

			// Set the data and decoded data for each argument
			msg.Arguments[i].Data = data
			msg.Arguments[i].Decoded = nil

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
