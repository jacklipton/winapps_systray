package ui

import (
	"github.com/gotk3/gotk3/gtk"
)

// ShowError shows a simple error message dialog.
func ShowError(parent *gtk.Window, message string) {
	dialog := gtk.MessageDialogNew(parent, gtk.DIALOG_MODAL, gtk.MESSAGE_ERROR, gtk.BUTTONS_OK, "%s", message)
	dialog.SetTitle("WinApps Error")
	dialog.Run()
	dialog.Destroy()
}

// ShowSetupRequired shows a dialog explaining that setup is needed.
// Returns true if the user clicked "Open Settings".
func ShowSetupRequired(parent *gtk.Window, message string) bool {
	dialog := gtk.MessageDialogNew(parent, gtk.DIALOG_MODAL, gtk.MESSAGE_WARNING, gtk.BUTTONS_NONE, "%s", message)
	dialog.SetTitle("WinApps Setup Required")
	_, _ = dialog.AddButton("Cancel", gtk.RESPONSE_CANCEL)
	btnSettings, _ := dialog.AddButton("Open Settings", gtk.RESPONSE_ACCEPT)
	btnSettings.GrabFocus()

	response := dialog.Run()
	dialog.Destroy()

	return response == gtk.RESPONSE_ACCEPT
}
