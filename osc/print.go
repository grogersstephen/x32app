package osc

import (
	"strings"
)

func (msg *Message) String() string {
	var sb strings.Builder
	for _, c := range msg.Packet.Bytes() {
		if c == 0 {
			sb.WriteString("~")
			continue
		}
		sb.WriteByte(c)
	}
	return sb.String()
}
