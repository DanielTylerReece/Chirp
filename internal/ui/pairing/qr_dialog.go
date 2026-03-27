package pairing

import (
	"fmt"
	"log"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"rsc.io/qr"
)

// QRDialog shows a QR code for phone pairing.
type QRDialog struct {
	dialog      *adw.Dialog
	picture     *gtk.Picture
	statusLabel *gtk.Label
	spinner     *gtk.Spinner
	contentBox  *gtk.Box
	onCancel    func()
}

// NewQRDialog creates a pairing dialog. Call Show() to display.
func NewQRDialog() *QRDialog {
	qd := &QRDialog{}

	// Main vertical box
	qd.contentBox = gtk.NewBox(gtk.OrientationVertical, 16)
	qd.contentBox.SetMarginTop(24)
	qd.contentBox.SetMarginBottom(24)
	qd.contentBox.SetMarginStart(24)
	qd.contentBox.SetMarginEnd(24)
	qd.contentBox.SetHAlign(gtk.AlignCenter)
	qd.contentBox.SetVAlign(gtk.AlignCenter)

	// Spinner (shown while waiting for QR URL)
	qd.spinner = gtk.NewSpinner()
	qd.spinner.SetSizeRequest(48, 48)
	qd.spinner.Start()
	qd.contentBox.Append(qd.spinner)

	// QR code picture (hidden initially)
	qd.picture = gtk.NewPicture()
	qd.picture.SetCanShrink(true)
	qd.picture.SetSizeRequest(280, 280)
	qd.picture.SetVisible(false)
	qd.contentBox.Append(qd.picture)

	// Status label
	qd.statusLabel = gtk.NewLabel("Connecting to Google Messages...")
	qd.statusLabel.SetWrap(true)
	qd.statusLabel.SetJustify(gtk.JustifyCenter)
	qd.statusLabel.SetMaxWidthChars(40)
	qd.statusLabel.AddCSSClass("body")
	qd.contentBox.Append(qd.statusLabel)

	// Cancel button
	cancelBtn := gtk.NewButtonWithLabel("Cancel")
	cancelBtn.SetHAlign(gtk.AlignCenter)
	cancelBtn.SetMarginTop(8)
	cancelBtn.ConnectClicked(func() {
		if qd.onCancel != nil {
			qd.onCancel()
		}
		qd.Close()
	})
	qd.contentBox.Append(cancelBtn)

	// Build the dialog using adw.Dialog
	qd.dialog = adw.NewDialog()
	qd.dialog.SetTitle("Pair with Phone")
	qd.dialog.SetContentWidth(400)
	qd.dialog.SetContentHeight(500)

	// Wrap content in a toolbar view with header bar
	toolbar := adw.NewToolbarView()
	toolbar.AddTopBar(adw.NewHeaderBar())
	toolbar.SetContent(qd.contentBox)

	qd.dialog.SetChild(toolbar)

	return qd
}

// SetQRCode generates and displays a QR code from the given URL.
func (qd *QRDialog) SetQRCode(url string) {
	code, err := qr.Encode(url, qr.M)
	if err != nil {
		log.Printf("pairing: qr encode: %v", err)
		qd.SetStatus(fmt.Sprintf("Failed to generate QR code: %v", err))
		return
	}

	pngData := code.PNG()

	gBytes := glib.NewBytes(pngData)
	texture, err := gdk.NewTextureFromBytes(gBytes)
	if err != nil {
		log.Printf("pairing: texture from bytes: %v", err)
		qd.SetStatus(fmt.Sprintf("Failed to display QR code: %v", err))
		return
	}

	qd.spinner.Stop()
	qd.spinner.SetVisible(false)
	qd.picture.SetPaintable(texture)
	qd.picture.SetVisible(true)
}

// SetStatus updates the status label.
func (qd *QRDialog) SetStatus(text string) {
	qd.statusLabel.SetText(text)
}

// SetOnCancel sets the callback for when the user cancels pairing.
func (qd *QRDialog) SetOnCancel(fn func()) {
	qd.onCancel = fn
}

// Close dismisses the dialog.
func (qd *QRDialog) Close() {
	qd.dialog.ForceClose()
}

// Show presents the dialog as a sheet over the parent window.
func (qd *QRDialog) Show(parent gtk.Widgetter) {
	qd.dialog.Present(parent)
}
