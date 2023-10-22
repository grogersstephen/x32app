package main

import (
	"fmt"
	"net"
	"time"

	"github.com/grogersstephen/x32app/osc"
)

func (m *mixer) makeFade(channelID int, start, stop float32, fadeDuration time.Duration) error {
	// Send a series of osc messages to the mixer.conn
	//     which cause the fader of the given channelID to fade from
	//     the value indicated by start to the value indicated by stop
	//     over the duration of fadeDuration

	// Get start and stop in terms of faderResolution
	startI, err := unitToFaderValue(start, m.faderResolution)
	if err != nil {
		return fmt.Errorf("invalid start value")
	}
	stopI, err := unitToFaderValue(stop, m.faderResolution)
	if err != nil {
		return fmt.Errorf("invalid stop value")
	}

	// Get the messages
	messages, err := m.getFadeMessages(channelID, startI, stopI, fadeDuration)
	if err != nil {
		return err
	}

	// Find the desired interval delay between each message
	interval := getInterval(getDist(startI, stopI), fadeDuration)

	// Set fader active flag
	m.faders[channelID].activate()

	// Trigger the messages
	err = m.faders[channelID].triggerFade(m.conn, messages, interval)
	if err != nil {
		return err
	}

	// Set fader inactive
	m.faders[channelID].deactivate()
	return nil
}

func (f *fader) triggerFade(conn net.Conn, messages []osc.Message, interval time.Duration) error {
	var failureCount int // keep count of how many attempts fail to send
	for i := range messages {
		// Check active status
		if !f.active {
			return fmt.Errorf("fade interrupted")
		}
		// Send message
		err := osc.Send(conn, messages[i])
		// Count failures
		switch err {
		case nil:
			failureCount = 0
		default:
			failureCount++
		}
		if failureCount > 9 { // too many failures in a row
			f.deactivate()
			return fmt.Errorf("too many failures sending osc msg")
		}
		time.Sleep(interval)
	}
	return nil
}
func (m *mixer) getFadeMessages(channelID int, startI, stopI int, fadeDuration time.Duration) (messages []osc.Message, err error) {

	// define the step value
	step := 1
	if startI > stopI {
		step = -1
	}

	// Write a list of osc messages
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
			return messages, err
		}

		// Append the OSC Message to the list
		messages = append(messages, msg)
	}

	return messages, err
}

func getInterval(dist int, d time.Duration) time.Duration {
	// Calculate interval between each step: duration / distance
	intervalF := float64(d.Milliseconds()) / float64(dist)
	interval, _ := time.ParseDuration(fmt.Sprintf("%vms", intervalF))
	return interval
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

func unitToFaderValue(u float32, faderResolution float32) (int, error) {
	// Converts a 'unit interval' (values in set [0,1]) float32 to an int in terms of faderResolution
	if u < 0 || u > 1 {
		return -1, fmt.Errorf("invalid unit interval value")
	}
	return int(u * faderResolution), nil
}
