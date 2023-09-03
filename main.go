package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/theme"
)

var App fyne.App

func main() {
	App = app.NewWithID("com.x32app.prototype")
	App.Settings().SetTheme(theme.DarkTheme())

	var home homeScreen
	home.setup()
	App.Run()
}
