package main

import (
	"fmt"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

type homeScreen struct {
	title        *canvas.Text
	connectB     *widget.Button
	channelBank  []*widget.Button
	dcaBank      []*widget.Button
	auxBank      []*widget.Button
	duration     line
	levelLabel   *widget.Label
	fadeTo       buttonLine
	fadeOutB     *widget.Button
	killCurrentB *widget.Button
	killAllB     *widget.Button
	renameChB    *widget.Button
	closeB       *widget.Button
	status       *widget.Label
	console      *console
	mixer        *mixer
	win          fyne.Window
}

func (h *homeScreen) setupDCABank() {
	h.dcaBank = make([]*widget.Button, 8)
	for i := 0; i < 8; i++ {
		channelID := 72 + i
		// Create our button which will only change the selected Ch
		button := widget.NewButton(
			fmt.Sprintf("DCA%d", i+1),
			func() {
				h.mixer.selectedCh = channelID
			},
		)
		// Add our button to the bank
		h.dcaBank[i] = button
	}
}

func (h *homeScreen) setupAUXBank() {
	h.auxBank = make([]*widget.Button, 8)
	for i := 0; i < 8; i++ {
		channelID := 32 + i
		button := widget.NewButton(
			fmt.Sprintf("AUX%d", i+1),
			func() {
				h.mixer.selectedCh = channelID
			},
		)
		h.auxBank[i] = button
	}
}

func (h *homeScreen) setupChannelBank() {
	h.channelBank = make([]*widget.Button, 32)
	for i := 0; i < 32; i++ {
		channelID := i
		// Create our button which will only change the selected Ch
		button := widget.NewButton(
			fmt.Sprintf("%02d", i+1),
			func() {
				fmt.Printf("channelID clicked: %v\n", channelID)
				h.mixer.selectedCh = channelID
			},
		)
		// Add our button to the bank
		h.channelBank[i] = button
	}
}

// TODO: make settings page
//     options:
//          fader resolution
//
func (h *homeScreen) setup() {
	// Set up the title at the top of the screen
	h.title = setupText("X32 App", color.White, 16)
	// Set up the connect button to load connect screen
	h.connectB = widget.NewButton("\nConnect\n", h.connectPress)
	// Set up the duration elements, label and entry
	h.duration = setupLine("Duration: ", "2s", "")
	// Set up the levelLabel which will show the fader level of the selected channel
	h.levelLabel = widget.NewLabel("")
	// Set up Fade To button
	h.fadeTo = setupButtonLine("\nFade To(0.00 to 1.00): \n", h.fadeToPress, "1", "")
	// Set up Fade Out button
	h.fadeOutB = widget.NewButton("\nFade Out\n", h.fadeOutPress)
	// Set up Kill current button
	h.killCurrentB = widget.NewButton("\nSTOPP\n", h.killCurrent)
	// Set up Kill all button
	h.killAllB = widget.NewButton("\nSTOP ALL\n", h.killAll)
	// Set up Rename Ch Button
	h.renameChB = widget.NewButton("\nRename\n", h.renameChPress)
	// Set up close button
	h.closeB = widget.NewButton("close", h.closeAppPress)
	// Set up status line which will show the X32 information
	h.status = widget.NewLabel("Application Started")
	// Setup the console
	h.console = newConsole("")
	// Set up the mixer with channel, dca, and bus send counts
	h.mixer = newX32()
	// Set up the fader select button banks
	h.setupChannelBank()
	h.setupDCABank()
	h.setupAUXBank()
	// Set up the window and content
	h.win = App.NewWindow("main")
	h.win.SetContent(h.getContent())
	h.win.Show()
}

func (h *homeScreen) getContent() *fyne.Container {
	h.console.log("")
	content :=
		container.New(layout.NewVBoxLayout(),
			h.title,
			h.connectB,
			h.status,
			container.NewGridWithColumns(8,
				h.channelBank[0], h.channelBank[1], h.channelBank[2], h.channelBank[3],
				h.channelBank[4], h.channelBank[5], h.channelBank[6], h.channelBank[7],
				h.channelBank[8], h.channelBank[9], h.channelBank[10], h.channelBank[11],
				h.channelBank[12], h.channelBank[13], h.channelBank[14], h.channelBank[15],
			),
			container.NewGridWithColumns(8,
				h.dcaBank[0], h.dcaBank[1], h.dcaBank[2], h.dcaBank[3],
				h.dcaBank[4], h.dcaBank[5], h.dcaBank[6], h.dcaBank[7],
				h.auxBank[0], h.auxBank[1], h.auxBank[2], h.auxBank[3],
				h.auxBank[4], h.auxBank[5], h.auxBank[6], h.auxBank[7],
			),
			container.NewGridWithColumns(1,
				h.levelLabel,
				container.NewGridWithColumns(2,
					h.duration.label,
					h.duration.entry,
				),
			),
			container.NewGridWithColumns(2,
				h.fadeTo.button,
				h.fadeTo.entry,
			),
			container.NewGridWithColumns(2,
				h.fadeOutB,
				h.killCurrentB,
			),
			h.killAllB,
			//h.renameChB,
			h.console.scroller,
			container.NewGridWithColumns(2,
				layout.NewSpacer(),
				h.closeB),
		)
	return content
}
