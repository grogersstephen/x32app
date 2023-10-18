package main

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/grogersstephen/x32app/osc"
)

type mixer struct {
	name             string
	remoteHost       string
	remotePort       int
	localPort        int
	levelMonitorPort int
	faders           []*fader
	selectedCh       int
	selectCh         chan int
	faderResolution  float32
	conn             net.Conn
	levelMonitorConn net.Conn
}

type fader struct {
	name      string
	channel   int
	channelID int
	level     float32
}

func newX32() *mixer {
	// Following channel id from unofficial x32 osc protocol
	// 0 - 31 are channels
	// 32 - 39 are aux in
	// 40 - 47 are fx returns
	// 48 - 63 are bus sends
	// 64 - 69 are matrices
	// 70 is main stereo
	// 71 is main mono
	// dca's have no channel ids, so we will assign them:
	//     72 - 79
	faderCount := 80
	// Initialize a mixer with defaults
	m := &mixer{
		name:             "Behringer X32",
		remotePort:       10023,
		localPort:        10023,
		levelMonitorPort: 10024,
		faders:           make([]*fader, faderCount),
		selectedCh:       0,
		selectCh:         make(chan int, 1),
		faderResolution:  1024,
		conn:             nil,
	}
	channelIDMap := map[int]string{
		0:  "channel",
		32: "aux",
		40: "fx",
		48: "bus",
		64: "matrix",
		70: "mains",
		71: "mono",
		72: "dca"}
	// set up channels
	name := "channel"
	channel := 1
	for i := 0; i < faderCount; i++ {
		// When 'i' is a valid key in channelIDMap, the 'name' variable will update
		val, ok := channelIDMap[i]
		if ok {
			name = val
			channel = 1
		}
		m.faders[i] = &fader{
			channelID: i,
			name:      name,
			channel:   channel,
			level:     0,
		}
		channel++
	}
	return m
}

func establishConnection(localPort int, remoteAddr string, tries int) (conn net.Conn, err error) {
	// Verify the validity of addresses and port numbers
	if !isValidIP(fmt.Sprintf(":%d", localPort)) {
		return nil, fmt.Errorf("invalid local port")
	}
	if !isValidIP(remoteAddr) {
		return nil, fmt.Errorf("invalide remote address")
	}
	// Iterate until a successful dial or until tries count
	for i := 0; i < tries; i++ {
		conn, err = osc.Dial(localPort, remoteAddr)
		if err == nil {
			break
		}
		time.Sleep(250 * time.Millisecond)
	}
	return conn, err
}

func (m *mixer) selChanMonitor() {
	// When we receive from selectCh, we'll assign to mixer.selectedCh
	for {
		m.selectedCh = <-m.selectCh
	}
}

func (m *mixer) levelMonitor(msg chan string) {
	// This keeps up with the level of the currently selected channel
	// Create a new connection just for the level monitor
	// Wait until we start a main connection
	var err error
	m.levelMonitorConn, err = establishConnection(
		m.levelMonitorPort,
		fmt.Sprintf("%s:%d", m.remoteHost, m.remotePort),
		5)
	// If a meaningful connection could not be made, abort the monitor
	if err != nil {
		return
	}
	for {
		level, _ := getFader(m.levelMonitorConn, m.selectedCh)
		m.faders[m.selectedCh].level = level
		msg <- m.faders[m.selectedCh].getLevelMessage()
		time.Sleep(10 * time.Millisecond)
	}
}

func (m *mixer) setFaderResolution(newValue string) error {
	faderResText := strings.TrimSpace(newValue)
	faderResInt, err := strconv.Atoi(faderResText)
	if err != nil {
		return fmt.Errorf("cannot parse fader resolution into int: fader resolution unchanged")
	}
	m.faderResolution = float32(faderResInt)
	return nil
}

func (m *mixer) connect() (err error) {
	m.conn, err = establishConnection(
		m.localPort,
		fmt.Sprintf("%s:%d", m.remoteHost, m.remotePort),
		5)
	return err
}

func (m *mixer) getStatus() (status []string, err error) {
	msg := osc.NewMessage("/info")
	reply, err := osc.Inquire(m.conn, msg)
	if err != nil {
		return status, err
	}
	for _, arg := range reply.Arguments {
		s, ok := arg.Decoded.(string)
		if !ok {
			s = ""
		}
		status = append(status, s)
	}
	return status, nil
}

func (m *mixer) getName(ch int) (string, error) {
	// Get the OSC method for the name of the channel
	namePath := getNamePath(ch)
	// Create the OSC message
	msg := osc.NewMessage(namePath)
	// Make the inquiry
	reply, err := osc.Inquire(m.conn, msg)
	if err != nil {
		return "", err
	}
	s, ok := reply.Arguments[0].Decoded.(string)
	if !ok {
		return "", fmt.Errorf("cannot get name")
	}
	return s, nil
}

func (m *mixer) setName(ch int, name string) error {
	namePath := getNamePath(ch)
	msg := osc.NewMessage(namePath)
	msg.AddString(name)
	err := osc.Send(m.conn, msg)
	return err
}

func (m *mixer) isInMotion(channelID int) bool {
	// Tests to see if the fader of the given channelID is currently in motion
	interval := 100 * time.Millisecond

	// Test fader level twice
	levelBefore, err := getFader(m.conn, channelID)
	if err != nil {
		return true // If the request fails, report fader to be in motion
	}
	time.Sleep(interval)
	levelAfter, err := getFader(m.conn, channelID)
	if err != nil {
		return true
	}

	// If the levels are not equal, the fader is in motion
	if levelBefore != levelAfter {
		return true
	}
	return false
}

func (m *mixer) fadeTo(channelID int, target float32, fadeDuration time.Duration) error {
	// Fade given channel
	//     from its current level to the given target level
	//     over the duration define by fadeDuration
	//   The target should be a value between 0 and 1

	if m.isInMotion(channelID) {
		return fmt.Errorf("fader currently in motion")
	}

	// Get current level of the fader
	currentLevel, err := m.getFader(channelID)
	if err != nil {
		return err
	}

	// Call mixer.makeFade
	err = m.makeFade(channelID, currentLevel, target, fadeDuration)

	return err
}

func (m *mixer) makeFade(channelID int, start, stop float32, fadeDuration time.Duration) (err error) {
	// Send a series of osc messages to the mixer.conn
	//     which cause the fader of the given channelID to fade from
	//     the value indicated by start to the value indicated by stop
	//     over the duration of fadeDuration
	// start and stop should be float32 in terms of 0 to 1
	if start < 0 || start > 1 {
		return fmt.Errorf("invalid start value")
	}
	if stop < 0 || stop > 1 {
		return fmt.Errorf("invalid stop value")
	}

	// initialize startI and stopI in terms of mixer.faderResolution
	startI := int(start * m.faderResolution)
	stopI := int(stop * m.faderResolution)

	// Get absolute distance between startI and stopI
	dist := getDist(startI, stopI)

	// Calculate delay between each step: duration / distance
	delayF := float64(fadeDuration.Milliseconds()) / float64(dist)
	delay, _ := time.ParseDuration(fmt.Sprintf("%vms", delayF))

	// define the step value
	step := 1
	if start > stop {
		step = -1
	}
	// Write a list of osc messages
	var oscMessages []osc.Message
	// Start at startI and inc/dec until stopI
	for i := startI; i != stopI; i += step {
		// Create a new osc.Message with the address of the channelID
		msg := osc.NewMessage(getFaderPath(channelID))
		// divide i by mixer.faderResolution to get a value on scale 0 - 1
		v := float32(i) / m.faderResolution
		// append the value to the osc.Message
		msg.AddFloat(v)
		err = msg.MakePacket()
		if err != nil {
			return err
		}

		// Append the OSC Message to the list
		oscMessages = append(oscMessages, msg)
	}

	// Fire off the messages
	var failureCount int // keep count of how many fail to send
	for i := range oscMessages {
		osc.Send(m.conn, oscMessages[i])
		switch err {
		case nil:
			failureCount = 0
		default:
			failureCount++
		}
		if failureCount > 9 { // too many failures in a row
			return fmt.Errorf("too many failures sending osc msg")
		}
		time.Sleep(delay) // sleep the calculated delay
	}

	return nil
}

func (m *mixer) getFader(channelID int) (level float32, err error) {
	return getFader(m.conn, channelID)
}

func getFader(conn net.Conn, channelID int) (level float32, err error) {
	// Return the level of the fader of given channelID 0 - 79
	// Check that connection is not nil
	if conn == nil {
		return level, fmt.Errorf("no connection made")
	}

	// Send the message
	msg := osc.NewMessage(getFaderPath(channelID))
	reply, err := osc.Inquire(conn, msg)
	if err != nil {
		return level, err
	}

	// Type check the first argument and assign to level
	level, ok := reply.Arguments[0].Decoded.(float32)
	if !ok {
		return level, fmt.Errorf("could not get fader level")
	}

	return level, err
}

func closeConnIfExists(conn net.Conn) {
	if conn != nil {
		conn.Close()
		conn = nil
	}
}
