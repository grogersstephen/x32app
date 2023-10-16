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

	// Set the default ports
	// TODO: save default ports elsewhere to reference if these variables are not set
	App.Preferences().SetInt("LevelMonitorPort", 10024)
	App.Preferences().SetInt("LPort", 10023)
	App.Preferences().SetInt("RPort", 10023)

	var home homeScreen
	home.setup()

	App.Run()
}
