package main

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

func (h *homeScreen) connectBPress() {
	// Update FADER_RESOLUTION from UI field
	err := updateFaderResolution(h.faderResolution.entry.Text)
	if err != nil {
		h.console.log <- err.Error()
	}
	// Get addresses from UI fields
	laddr := h.lAddrEntry.entry.Text
	raddr := h.rAddrEntry.entry.Text
	// TODO: check if the laddr and raddr are in address form
	// If the field is not empty, then write to Fyne Prefs
	if laddr != "" {
		App.Preferences().SetString("LAddr", laddr)
	}
	if raddr != "" {
		App.Preferences().SetString("RAddr", raddr)
	}
	go func(console chan string, status chan string) {
		conn, err := connect()
		defer conn.Close()
		if err != nil {
			console <- err.Error()
		}
		console <- fmt.Sprintf("Remote: %v\nLocal: %v\n", conn.RemoteAddr().String(), conn.LocalAddr().String())
		ss, err := getStatus(conn)
		var statusS string
		if err != nil {
			h.console.log <- err.Error()
		} else {
			for j := range ss {
				statusS += ss[j] + "   "
			}
			status <- statusS
		}
	}(h.console.log, h.status.msg)
}

func (h *homeScreen) fadeToPress() {
	// Parse duration from field
	duration, err := time.ParseDuration(h.duration.entry.Text)
	if err != nil {
		h.console.log <- err.Error()
	}
	// Get the float32 value of fadeTo field
	target, err := strconv.ParseFloat(h.fadeTo.entry.Text, 32)
	targetF := float32(target)
	if err != nil {
		h.console.log <- err.Error()
	}
	go func(selectedChannel int, duration time.Duration, targetF float32, console chan string) {
		// Make connection
		fmt.Fprintf(os.Stderr, "about to make connection\n")
		conn, err := connect()
		if err != nil {
			console <- fmt.Sprintf("did not make connection!\n%v", err.Error())
			return
		}
		console <- "made connection!"
		// Call fadeTo() function
		err = fadeTo(conn, selectedChannel, targetF, duration)
		defer conn.Close()
		if err != nil {
			console <- err.Error()
		}
	}(h.selectedChannel, duration, targetF, h.console.log)
}

func (h *homeScreen) fadeOutPress() {
	// Parse duration from field
	duration, err := time.ParseDuration(h.duration.entry.Text)
	if err != nil {
		h.console.log <- err.Error()
	}
	go func(selectedChannel int, console chan string) {
		// Make connection
		conn, err := connect()
		if err != nil {
			console <- err.Error()
			return
		}
		// Call fadeTo() to 0
		err = fadeTo(conn, selectedChannel, 0, duration)
		defer conn.Close()
		if err != nil {
			console <- err.Error()
		}
	}(h.selectedChannel, h.console.log)
}

func (h *homeScreen) closeAppPress() {
	os.Exit(1)
}
