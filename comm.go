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
	name            string
	remoteHost      string
	remotePort      int
	localPort       int
	faders          []*fader
	selectedCh      int
	selectCh        chan int
	faderResolution float32
	conn            net.Conn
	monitor         *levelMonitor
}

type levelMonitor struct {
	localPort int
	conn      net.Conn
	updatedAt time.Time
}

type fader struct {
	name      string
	channel   int
	channelID int
	active    bool // in active motion?
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
		name:            "Behringer X32",
		remoteHost:      "",
		remotePort:      10023,
		localPort:       10023,
		faders:          make([]*fader, faderCount),
		selectedCh:      0,
		selectCh:        make(chan int), // unbuffered
		faderResolution: 1024,
		conn:            nil,
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
			active:    false,
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

func (m *mixer) monitorLevels(msg chan string) {
	// This keeps up with the level of the currently selected channel
	//     Updates msg with label of the channel and its level
	// Create a new connection just for the level monitor
	// Wait until we start a main connection
	var err error
	m.monitor.conn, err = establishConnection(
		m.monitor.localPort,
		fmt.Sprintf("%s:%d", m.remoteHost, m.remotePort),
		5)
	// If a meaningful connection could not be made, abort the monitor
	if err != nil {
		return
	}
	for {
		m.faders[m.selectedCh].getLevel(m.monitor.conn)
		msg <- m.faders[m.selectedCh].levelMessage()
		m.monitor.updatedAt = time.Now()
		time.Sleep(42 * time.Millisecond) // a 41.6667ms interval is equivalent to 24hz
	}
}

func (m *mixer) isMonitorActive() bool {
	// time since monitor.updatedAt was updated is less than a second
	return time.Since(m.monitor.updatedAt) < time.Second
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
	//     This test will return true even if another source is causing the motion
	interval := 100 * time.Millisecond

	// Test fader level twice
	levelBefore, err := m.faders[channelID].getLevel(m.conn)
	if err != nil {
		return true // If the request fails, report fader to be in motion
	}
	// Sleep
	time.Sleep(interval)
	// Test fader level again
	levelAfter, err := m.faders[channelID].getLevel(m.conn)
	if err != nil {
		return true
	}

	// If the levels are not equal, the fader is in motion
	if levelBefore != levelAfter {
		return true
	}
	return false
}

// When run with all, it doesn't always kill the first ch
func (m *mixer) killSwitch(channelIDs ...int) {
	// Send enough kill signals to receive at all 'fader motion' goroutines
	for _, id := range channelIDs {
		m.faders[id].deactivate()
	}
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
	currentLevel, err := m.faders[channelID].getLevel(m.conn)
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

	// Set fader active flag
	m.faders[channelID].activate()

	// Fire off the messages
	var failureCount int // keep count of how many attempts fail to send
	for i := range oscMessages {
		// Check active status
		if !m.faders[channelID].active {
			return fmt.Errorf("fade on ch %d interrupted", channelID)
		}
		// Send message
		osc.Send(m.conn, oscMessages[i])
		// Count failures
		switch err {
		case nil:
			failureCount = 0
		default:
			failureCount++
		}
		if failureCount > 9 { // too many failures in a row
			m.faders[channelID].deactivate()
			return fmt.Errorf("too many failures sending osc msg")
		}
		time.Sleep(delay) // sleep the calculated delay
	}

	m.faders[channelID].deactivate()
	return nil
}

func (f *fader) activate() {
	f.active = true
}
func (f *fader) deactivate() {
	f.active = false
}

func (f *fader) getLevel(conn net.Conn) (level float32, err error) {
	level, err = getFaderLevel(f.channelID, conn)

	// Assign the level
	f.level = level

	return level, nil
}
func getFaderLevel(channelID int, conn net.Conn) (level float32, err error) {
	// Return the level of the given channel's fader
	// Check that the connection is not nil
	if conn == nil {
		return level, fmt.Errorf("no connection made")
	}

	// Send the message
	msg := osc.NewMessage(getFaderPath(channelID))
	reply, err := osc.Inquire(conn, msg)
	if err != nil {
		return level, err
	}

	// Type check the first argument
	level, ok := reply.Arguments[0].Decoded.(float32)
	if !ok {
		return level, fmt.Errorf("could not get fader channelID %d level", channelID)
	}

	return level, nil
}

func closeConnIfExists(conn net.Conn) {
	if conn != nil {
		conn.Close()
		conn = nil
	}
}
