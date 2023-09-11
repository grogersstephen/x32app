package main

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/grogersstephen/x32app/osc"
)

// All levels will be calculated in terms of FADER_RESOLUTION
var FADER_RESOLUTION float32 = 1024

func updateFaderResolution(newValue string) (msg string) {
	// Check if FADER_RESOLUTION has changed
	// If it's the same as the default, return
	faderResText := strings.TrimSpace(newValue)
	if faderResText == "1024" {
		return msg
	}
	faderResInt, err := strconv.Atoi(faderResText)
	if err != nil {
		return fmt.Sprintf("cannot parse fader resolution into int\nfader resolution unchanged")
	}
	FADER_RESOLUTION = float32(faderResInt)
	return fmt.Sprintf("fader resolution updated to %d", faderResInt)
}
func writeLAddr(addr string) {
	App.Preferences().SetString("LAddr", addr)
}
func writeRAddr(addr string) {
	App.Preferences().SetString("RAddr", addr)
}
func getLAddr() string {
	return App.Preferences().String("LAddr")
}
func getRAddr() string {
	return App.Preferences().String("RAddr")
}

func connect() (conn *net.UDPConn, err error) {
	raddr := getRAddr()
	if raddr == "" {
		return conn, fmt.Errorf("remote addr not set")
	}
	laddr := getLAddr()
	if laddr == "" {
		return conn, fmt.Errorf("local addr not set")
	}
	conn, err = osc.Dial(laddr, raddr)
	return conn, err
}

func getStatus(conn *net.UDPConn) (ss []string, err error) {
	msg := osc.NewMessage("/info")
	a, err := osc.Inquire(conn, msg)
	if err != nil {
		return ss, err
	}
	for i := range a {
		s, ok := a[i].(string)
		if !ok {
			s = ""
		}
		ss = append(ss, s)
	}
	return ss, nil
}

func fadeTo(conn *net.UDPConn, channel int, target float32, fadeDuration time.Duration) error {
	// target float32 is a level from 0 to 1
	currentLevel, err := getChFader(conn, channel)
	if err != nil {
		return err
	}
	err = makeFade(conn, channel, currentLevel, target, fadeDuration)
	return err
}

func getFaderPath(ch int) string {
	var path, zerodigit, chS string
	switch {
	case ch > 32:
		chS = fmt.Sprintf("%d", ch-32)
		path = filepath.Join("/dca", chS, "fader")
	default:
		chS = fmt.Sprintf("%d", ch)
		if len(chS) == 1 {
			zerodigit = "0"
		}
		path = filepath.Join("/ch", zerodigit+chS, "mix/fader")
	}
	return path
}

func getDist(x, y int) int {
	// Returns the absolute distance between two integers
	if x < y {
		return y - x
	}
	return x - y
}

func makeFade(conn *net.UDPConn, channel int, start, stop float32, fadeDuration time.Duration) error {
	fmt.Fprintf(os.Stderr, "in makefade()\n")
	fmt.Fprintf(os.Stderr, "channel: %v\n", channel)
	fmt.Fprintf(os.Stderr, "start: %v\n", start)
	fmt.Fprintf(os.Stderr, "stop: %v\n", stop)
	fmt.Fprintf(os.Stderr, "fadeDuration: %v\n", fadeDuration)
	// start and stop are float32 in terms of 0 to 1
	step := 1
	if start > stop {
		step = -1
	}
	// We must convert to ints and figure in terms of FADER_RESOLUTION
	startI := int(start * FADER_RESOLUTION)
	stopI := int(stop * FADER_RESOLUTION)
	dist := getDist(startI, stopI) // absolute distance between startI and stopI
	// delay is:    desired duration / distance
	delayF := float64(fadeDuration.Milliseconds()) / float64(dist)
	delay, _ := time.ParseDuration(fmt.Sprintf("%vms", delayF))
	margin := int(.01 * float64(dist)) // margins are with 2% of the target
	for i := startI; (stopI-i) > margin || (stopI-i) < -margin; i += step {
		v := float32(i) / FADER_RESOLUTION // convert to a float32 on a scale from 0 to 1
		// Create our message
		// append the float32 to the message
		// send the message
		msg := osc.NewMessage(getFaderPath(channel))
		err := msg.Add(v)
		fmt.Fprintf(os.Stderr, "Msg: %v\n", msg.String())
		if err != nil {
			return err
		}
		err = osc.Send(conn, msg)
		if err != nil {
			return err
		}
		time.Sleep(delay) // sleep the calculated delay
	}
	return nil
}

func getChFader(conn *net.UDPConn, channel int) (float32, error) {
	var chS, addr string
	switch {
	case channel > 32:
		dca := channel - 32
		chS = fmt.Sprintf("%d", dca)
		addr = "/dca/" + chS + "/fader~~~~"
	default:
		chS = fmt.Sprintf("%02d", channel)
		addr = "/ch/" + chS + "/mix/fader~~~~"
	}
	err := osc.SendString(conn, addr)
	if err != nil {
		return -1, err
	}
	msg, err := osc.Listen(conn, 5*time.Second)
	if err != nil {
		return -1, err
	}
	a := msg.DecodeArgument(0)
	f, ok := a.(float32)
	if !ok {
		return -1, fmt.Errorf("cant get fader level")
	}
	return f, nil
}

func getChFaderB(conn *net.UDPConn, channel int) (float32, error) {
	// Build message
	chS := fmt.Sprintf("%02d", channel)
	msg := osc.NewMessage(filepath.Join("/ch/", chS, "/mix/fader"))
	// Send an inquiry
	a, err := osc.Inquire(conn, msg)
	if err != nil {
		return 0, err
	}
	faderValueF, ok := a[0].(float32)
	if !ok {
		return 0, fmt.Errorf("fader value returned not a float32")
	}
	return faderValueF, nil
}
