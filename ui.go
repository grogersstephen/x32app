package main

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type line struct {
	label *widget.Label
	entry *widget.Entry
}

type buttonLine struct {
	button *widget.Button
	entry  *widget.Entry
}

func setupButtonLine(buttonS string, f func(), entryText, entryPlaceholder string) buttonLine {
	var bl buttonLine
	bl.button = widget.NewButton(buttonS, f)
	bl.entry = setupEntry(entryText, entryPlaceholder)
	return bl
}

func setupLine(labelS, entryText, entryPlaceholder string) line {
	var l line
	l.label = setupLabel(labelS)
	l.entry = setupEntry(entryText, entryPlaceholder)
	return l
}

func setupText(s string, c color.Color, textSize int) *canvas.Text {
	t := canvas.NewText(s, c)
	t.Alignment = fyne.TextAlignCenter
	t.TextSize = float32(textSize)
	return t
}

func setupLabel(s string) *widget.Label {
	l := &widget.Label{Alignment: fyne.TextAlignLeading}
	l.TextStyle.Monospace = true
	l.Wrapping = fyne.TextWrapWord
	if len(s) > 0 {
		l.SetText(s)
	}
	return l
}

func setupEntry(s, p string) *widget.Entry {
	e := &widget.Entry{}
	e.TextStyle.Monospace = true
	switch {
	case len(s) > 0:
		e.SetText(s)
	case len(p) > 0:
		e.SetPlaceHolder(p)
	}
	return e
}

/*
type console struct {
	scroller *container.Scroll
	label    *widget.Label
	log      chan string
}
*/

type console struct {
	widget.Label
	scroller *container.Scroll
}

func newConsole(s string) *console {
	c := &console{}
	c.ExtendBaseWidget(c)
	c.SetText(s)
	c.scroller = container.NewVScroll(c)
	return c
}

func (c *console) log(s string) {
	if s == "clr" {
		c.SetText("")
		return
	}
	c.SetText(c.Text + "\n" + s)
	c.scroller.ScrollToBottom()
}
