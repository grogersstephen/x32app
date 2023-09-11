package main

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

func (h *homeScreen) connectBPress() {
	// Check if FADER_RESOLUTION has changed
	msg := updateFaderResolution(h.faderResolution.entry.Text)
	if msg != "" {
		h.console.log(msg)
	}
	laddr := h.lAddrEntry.entry.Text
	raddr := h.rAddrEntry.entry.Text
	if laddr != "" {
		writeLAddr(laddr)
	}
	if raddr != "" {
		writeRAddr(raddr)
	}
	conn, err := connect()
	defer conn.Close()
	if err != nil {
		h.console.log(err.Error())
	}
	h.console.log(fmt.Sprintf("Remote: %v\nLocal: %v\n", conn.RemoteAddr().String(), conn.LocalAddr().String()))
	status, err := getStatus(conn)
	var statusS string
	if err != nil {
		h.console.log(err.Error())
	} else {
		for j := range status {
			statusS += status[j] + "   "
		}
		h.status.SetText(statusS)
	}
}

func (h *homeScreen) fadeToPress() {
	// Parse duration from field
	dur, err := time.ParseDuration(h.duration.entry.Text)
	if err != nil {
		h.console.log(err.Error())
	}
	// Get the float32 value of fadeTo field
	target, err := strconv.ParseFloat(h.fadeTo.entry.Text, 32)
	targetF := float32(target)
	if err != nil {
		h.console.log(err.Error())
	}
	// Make connection
	fmt.Fprintf(os.Stderr, "about to make connection\n")
	conn, err := connect()
	if err != nil {
		fmt.Fprintf(os.Stderr, "did not make connection!\n")
		h.console.log(err.Error())
		return
	}
	fmt.Fprintf(os.Stderr, "made connection\n")
	// Call fadeTo() function
	err = fadeTo(conn, h.selectedChannel, targetF, dur)
	defer conn.Close()
	if err != nil {
		h.console.log(err.Error())
	}
}

func (h *homeScreen) fadeOutPress() {
	// Parse duration from field
	dur, err := time.ParseDuration(h.duration.entry.Text)
	if err != nil {
		h.console.log(err.Error())
	}
	// Make connection
	conn, err := connect()
	if err != nil {
		h.console.log(err.Error())
		return
	}
	// Call fadeTo() to 0
	err = fadeTo(conn, h.selectedChannel, 0, dur)
	defer conn.Close()
	if err != nil {
		h.console.log(err.Error())
	}
}

func (h *homeScreen) closeAppPress() {
	os.Exit(1)
}
