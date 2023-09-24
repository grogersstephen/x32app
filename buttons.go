package main

import (
	"fmt"
	"log"
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
	go func() {
		conn, err := connect()
		defer conn.Close()
		if err != nil {
			h.console.log <- err.Error()
		}
		h.console.log <- fmt.Sprintf("Remote: %v\nLocal: %v\n", conn.RemoteAddr().String(), conn.LocalAddr().String())
		ss, err := getStatus(conn)
		var statusS string
		if err != nil {
			h.console.log <- err.Error()
		} else {
			for j := range ss {
				statusS += ss[j] + "   "
			}
			h.status.msg <- statusS
		}
	}()
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
	go func() {
		done := false
		doneC := make(chan bool)
		chLabel := fmt.Sprintf("Channel %d", h.selectedChannel)
		if h.selectedChannel > 32 {
			chLabel = fmt.Sprintf("DCA %d", h.selectedChannel-32)
		}
		// Make connection
		conn, err := connect()
		if err != nil {
			h.console.log <- fmt.Sprintf("did not make connection!\n%v", err.Error())
			return
		}
		defer conn.Close()
		h.console.log <- "made connection!"
		// Call fadeTo() function
		go func() {
			err = fadeTo(conn, h.selectedChannel, targetF, duration)
			if err != nil {
				h.console.log <- err.Error()
			}
			doneC <- true
		}()
		for !done {
			currentLevel, err := getChFader(conn, h.selectedChannel)
			if err != nil {
				continue
			}
			h.levelLabel.msg <- fmt.Sprintf("%s : %.2f\n", chLabel, currentLevel)
			select {
			case done = <-doneC:
				log.Printf("done")
			default:
				log.Printf("nah")
			}
		}
	}()
}

func (h *homeScreen) fadeOutPress() {
	// Parse duration from field
	duration, err := time.ParseDuration(h.duration.entry.Text)
	if err != nil {
		h.console.log <- err.Error()
	}
	go func() {
		done := false
		doneC := make(chan bool)
		chLabel := fmt.Sprintf("Channel %d", h.selectedChannel)
		if h.selectedChannel > 32 {
			chLabel = fmt.Sprintf("DCA %d", h.selectedChannel-32)
		}
		// Make connection
		conn, err := connect()
		if err != nil {
			h.console.log <- err.Error()
			return
		}
		defer conn.Close()
		// Call fadeTo() to 0
		go func() {
			err = fadeTo(conn, h.selectedChannel, 0, duration)
			if err != nil {
				h.console.log <- err.Error()
			}
			doneC <- true
		}()
		for !done {
			currentLevel, err := getChFader(conn, h.selectedChannel)
			if err != nil {
				continue
			}
			h.levelLabel.msg <- fmt.Sprintf("%s : %.2f\n", chLabel, currentLevel)
			select {
			case done = <-doneC:
				log.Printf("done")
			default:
				log.Printf("nah")
			}
		}
	}()
}

func (h *homeScreen) closeAppPress() {
	os.Exit(1)
}
