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
		faderResolution: 1024,
		conn:            nil,
		monitor: &levelMonitor{
			localPort: 10024,
			conn:      nil,
			updatedAt: time.Now(),
		},
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

func (m *mixer) monitorLevels(levelLog func(s string)) {
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
		fmt.Printf("m.selectedCh: %v\n", m.selectedCh)
		// the following getLevel() is blocking, should be on goroutine?
		m.faders[m.selectedCh].getLevel(m.monitor.conn)
		levelLog(m.faders[m.selectedCh].levelMessage())
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

func (f *fader) activate() {
	f.active = true
}
func (f *fader) deactivate() {
	f.active = false
}

func (f *fader) getLevel(conn net.Conn) (level float32, err error) {
	level, err = getFaderLevel(f.channelID, conn)
	if err != nil {
		return level, err
	}

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
func (f *fader) subLevel(conn net.Conn, levelOut func(s string)) error {
	// This will only be valid for 10 seconds
	// Check that the conn is not nil
	if conn == nil {
		return fmt.Errorf("no connection made")
	}
	// Make message to send
	msg := osc.NewMessage("/subscribe")
	msg.AddString(getFaderPath(f.channelID))
	// Send the message
	for {
		reply, err := osc.Listen(conn)
		if err != nil {
			return err
		}
		level, ok := reply.Arguments[0].Decoded.(float32)
		if !ok {
			return fmt.Errorf("incoming value cannot be parsed as float")
		}
		// Assign the level
		f.level = level
		//
		levelOut(fmt.Sprintf("%.2f", f.level))
	}
}

func closeConnIfExists(conn net.Conn) {
	if conn != nil {
		conn.Close()
		conn = nil
	}
}
