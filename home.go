package main

import (
	"image/color"

	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
)

type channelBank struct {
	buttons []*widget.Button
}

type homeScreen struct {
	title          *canvas.Text
	connectionInfo *widget.Label
	changeIPB      *widget.Button
	channelBank
	duration   line
	levelLabel *widget.Label
	fadeTo     buttonLine
	fadeOutB   *widget.Button
	closeB     *widget.Button
	status     *widget.Label
}

func (h *homeScreen) setup() {
	h.title = setupText("X32 App", color.White, 16)
	h.connectionInfo = setupLabel("")
	h.changeIPB = widget.NewButton("Change IP", changeIP)
	h.channelBank = channelBank{}
	h.duration = setupLine("Duration: ", "2s")
	h.levelLabel = setupLabel("")
	h.fadeTo = setupButtonLine("Fade To: ", fadeTo, "100")
	h.fadeOutB = widget.NewButton("Fade Out", fadeOut)
	h.closeB = widget.NewButton("close", closeApp)
	h.status = setupLabel("")
}
