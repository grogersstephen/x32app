package main

import (
	"image/color"

	fyne "fyne.io/fyne/v2/"
	"fyne.io/fyne/v2/canvas"
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

func setupButtonLine(buttonS string, f func(), entryS string) buttonLine {
	var bl buttonLine
	bl.button = widget.NewButton(buttonS, f)
	bl.entry = setupEntry(entryS)
	return bl
}

func setupLine(labelS string, entryS string) line {
	var l line
	l.label = setupLabel(labelS)
	l.entry = setupEntry(entryS)
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

func setupEntry(s string) *widget.Entry {
	e := &widget.Entry{}
	e.TextStyle.Monospace = true
	if len(s) > 0 {
		e.SetText(s)
	}
	return e
}
