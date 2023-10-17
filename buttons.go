package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

func (h *homeScreen) renameChPress() {
	ch := h.mixer.selectedCh
	fader := h.mixer.faders[ch]
	entry := widget.NewEntry()
	dialog.ShowForm(
		fmt.Sprintf("Rename %s %d", fader.name, fader.channel),
		"Confirm",
		"Cancel",
		[]*widget.FormItem{
			{Text: "Entry", Widget: entry},
		},
		func(confirmRename bool) {
			if confirmRename {
				go func() {
					err := h.mixer.setName(ch, entry.Text)
					if err != nil {
						h.console.log <- err.Error()
					}
					h.renameChButtons()
				}()
			}
		},
		h.win)
}

func (h *homeScreen) fadeToPress() {
	// Get the float32 value of fadeTo field
	target, err := strconv.ParseFloat(h.fadeTo.entry.Text, 32)
	targetF := float32(target)
	if err != nil {
		h.console.log <- err.Error()
		return
	}
	go h.fade(targetF)
}

func (h *homeScreen) fadeOutPress() {
	go h.fade(0)
}

func (h *homeScreen) fade(target float32) {
	// Parse duration from field
	duration, err := time.ParseDuration(h.duration.entry.Text)
	if err != nil {
		h.console.log <- err.Error()
		return
	}

	// fade to target
	err = h.mixer.fadeTo(h.mixer.selectedCh, target, duration)
	if err != nil {
		h.console.log <- err.Error()
	}
}

func (h *homeScreen) connectPress() {
	// Set up ui entry
	entry := widget.NewEntry()
	entry.SetText(App.Preferences().String("RHost"))
	entry.SetPlaceHolder("Set remote ip address")
	// Show Dialog
	dialog.ShowForm(
		"Connect to Mixing Console",
		"Confirm",
		"Cancel",
		[]*widget.FormItem{
			{Text: "IP Address", Widget: entry},
		},
		func(confirmConnect bool) {
			if confirmConnect {
				go func() {
					// Close the current Conn if it exists
					closeConnIfExists(h.mixer.conn)

					// Get ip address from entry
					rhost := entry.Text
					if !isValidIP(rhost) {
						h.console.log <- "invalid ip address"
						return
					}

					// Set the fyne App preference
					App.Preferences().SetString("RHost", rhost)

					// Set the mixer property
					h.mixer.remoteHost = rhost
					// Make the connection
					err := h.mixer.connect()
					if err != nil {
						h.mixer.conn = nil
						h.console.log <- err.Error()
					}
					// Play loading animation while attempting connection
					doneSignal := make(chan bool, 1)
					loadingAnimation(h.console.log, doneSignal)
					// Try to get status
					ss, err := h.mixer.getStatus()
					doneSignal <- true
					h.console.log <- "clr"
					if err != nil {
						h.mixer.conn = nil
						h.console.log <- "bad connection"
						h.console.log <- err.Error()
						return
					}
					// Send the message to the console
					h.status.msg <- strings.Join(ss, " ")
					// Rename buttons
					h.renameChButtons()
					// Start levelMonitor
					go h.mixer.levelMonitor(h.levelLabel.msg)
				}()
			}
		},
		h.win)
}

func (h *homeScreen) renameChButtons() {
	for i, button := range h.channelBank {
		name, err := h.mixer.getName(i)
		if err != nil {
			continue
		}

		button.SetText(name)
	}
}

func (h *homeScreen) closeAppPress() {
	h.mixer.levelMonitorConn.Close()
	h.mixer.conn.Close()
	os.Exit(1)
}
