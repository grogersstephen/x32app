package osc

import (
	"encoding/binary"
	"math"
	"net"
	"strings"
	"time"
)

func Dial(localAddr, remoteAddr string) (conn *net.UDPConn, err error) {
	// Give localAddr and remoteAddr in the form: "ip:port"
	local, err := net.ResolveUDPAddr("udp", localAddr)
	if err != nil {
		return conn, err
	}
	remote, err := net.ResolveUDPAddr("udp", remoteAddr)
	if err != nil {
		return conn, err
	}
	conn, err = net.DialUDP("udp", local, remote)
	return conn, err
}

func Inquire(conn *net.UDPConn, msg Message) (a []any, err error) {
	// Send message
	err = Send(conn, msg)
	if err != nil {
		return a, err
	}
	// Wait for reply
	reply, err := Listen(conn, 4*time.Second)
	if err != nil {
		return a, err
	}
	for i := range reply.Arguments {
		a = append(a, reply.DecodeArgument(i))
	}
	return a, nil
}

func SendString(conn *net.UDPConn, s string) error {
	var sb strings.Builder
	conn.SetWriteDeadline(time.Now().Add(4 * time.Second))
	for i := range s {
		if s[i] == '~' {
			sb.WriteByte(byte(0))
			continue
		}
		sb.WriteByte(s[i])
	}
	_, err := conn.Write([]byte(sb.String()))
	return err
}

func Send(conn *net.UDPConn, msg Message) error {
	// Send an OSC message of type Message to the UDPConn connection
	// Make the packet from the components if it doesn't already exist
	if len(msg.Packet) == 0 {
		err := msg.MakePacket()
		if err != nil {
			return err
		}
	}

	// Sends a message using the Conn from net.DialUDP
	// Write the bytes to the connection
	conn.SetWriteDeadline(time.Now().Add(4 * time.Second))
	_, err := conn.Write(msg.Packet)
	return err
}

func Listen(conn *net.UDPConn, timeout time.Duration) (msg *Message, err error) {
	// Set the deadline from our timeout
	conn.SetReadDeadline(time.Now().Add(timeout))
	// Make a byt of length 256 and read into it
	byt := make([]byte, 256)
	_, err = conn.Read(byt)
	if err != nil {
		return msg, err
	}

	// Make message and write to Packet
	msg = &Message{
		Packet: byt,
	}

	// Parse the []byte in msg.Packet and populate the properties of msg
	err = msg.ParseMessage()

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
