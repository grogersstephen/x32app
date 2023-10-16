package osc

import (
	"encoding/binary"
	"math"
	"net"
)

func Dial(localPort int, remoteAddr string) (conn net.Conn, err error) {
	// Takes a local port and remote address and returns a net.Conn
	//     remoteAddr should be provided in the form: "ip:port"
	dialer := &net.Dialer{
		LocalAddr: &net.UDPAddr{
			Port: localPort,
		},
	}
	conn, err = dialer.Dial("udp", remoteAddr)
	return conn, err
}

func Inquire(conn net.Conn, msg Message) (reply Message, err error) {
	// Takes a Conn and an osc Message
	//   Sends the message to a server, and listens for a response
	//   Returns the data in the response as a slice of any/interface

	// Send message
	err = Send(conn, msg)
	if err != nil {
		return reply, err
	}

	// Wait for reply
	reply, err = Listen(conn)
	if err != nil {
		return reply, err
	}
	// Decode arguments
	reply.DecodeArguments()
	return reply, nil
}

func Send(conn net.Conn, msg Message) error {
	// Send an OSC message of type Message to the Conn connection

	// Make the packet from the components if it doesn't already exist
	if msg.Packet.Len() == 0 {
		err := msg.MakePacket()
		if err != nil {
			return err
		}
	}

	// Write the bytes to the connection
	_, err := conn.Write(msg.Packet.Bytes())
	return err
}

func Listen(conn net.Conn) (msg Message, err error) {
	// Act as a server and listen for an incoming OSC message
	// Make a []byte of length 512 and read into it
	byt := make([]byte, 512)
	_, err = conn.Read(byt)
	if err != nil {
		return msg, err
	}

	// Write bytes to packet
	msg.Packet.Write(byt)

	// Parse the []byte in msg.Packet and populate the properties of msg
	err = msg.ParseMessage()

	// Return msg
	return msg, err
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
