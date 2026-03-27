package content

import (
	"context"
	"log"
	"os"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

// showImageViewer opens a fullscreen-style dialog displaying the image at
// full resolution, with a Save button to write it to disk.
func showImageViewer(parent gtk.Widgetter, imageData []byte, mimeType string) {
	dialog := adw.NewDialog()
	dialog.SetTitle("Image")
	dialog.SetContentWidth(800)
	dialog.SetContentHeight(600)

	content := gtk.NewBox(gtk.OrientationVertical, 8)

	// Large picture
	picture := gtk.NewPicture()
	picture.SetCanShrink(true)
	picture.SetContentFit(gtk.ContentFitContain)
	picture.SetVExpand(true)
	picture.SetHExpand(true)

	gBytes := glib.NewBytesWithGo(imageData)
	texture, err := gdk.NewTextureFromBytes(gBytes)
	if err != nil {
		log.Printf("image_viewer: load texture: %v", err)
		errLabel := gtk.NewLabel("Failed to load image")
		content.Append(errLabel)
	} else {
		picture.SetPaintable(texture)
	}
	content.Append(picture)

	// Save button
	saveBtn := gtk.NewButtonWithLabel("Save Image")
	saveBtn.AddCSSClass("suggested-action")
	saveBtn.AddCSSClass("pill")
	saveBtn.SetHAlign(gtk.AlignCenter)
	saveBtn.SetMarginBottom(16)
	saveBtn.SetMarginTop(8)
	saveBtn.ConnectClicked(func() {
		fileDialog := gtk.NewFileDialog()
		fileDialog.SetTitle("Save Image")

		ext := ".jpg"
		switch mimeType {
		case "image/png":
			ext = ".png"
		case "image/gif":
			ext = ".gif"
		case "image/webp":
			ext = ".webp"
		case "image/bmp":
			ext = ".bmp"
		}
		fileDialog.SetInitialName("image" + ext)

		// Parent window is optional — nil works (same pattern as compose_bar.go)
		fileDialog.Save(context.Background(), nil, func(res gio.AsyncResulter) {
			file, err := fileDialog.SaveFinish(res)
			if err != nil {
				// User cancelled or error
				return
			}
			path := file.Path()
			if path == "" {
				return
			}
			if err := os.WriteFile(path, imageData, 0644); err != nil {
				log.Printf("image_viewer: save file %s: %v", path, err)
			}
		})
	})
	content.Append(saveBtn)

	toolbar := adw.NewToolbarView()
	toolbar.AddTopBar(adw.NewHeaderBar())
	toolbar.SetContent(content)
	dialog.SetChild(toolbar)

	dialog.Present(parent)
}
