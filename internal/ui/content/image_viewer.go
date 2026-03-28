package content

import (
	"context"
	"io"
	"log"
	"os"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

// showImageViewer opens a fullscreen-style dialog displaying the image from
// a file path, with a Save button to copy it to disk.
func showImageViewer(parent gtk.Widgetter, imagePath string, mimeType string) {
	dialog := adw.NewDialog()
	dialog.SetTitle("Image")
	dialog.SetContentWidth(800)
	dialog.SetContentHeight(600)

	content := gtk.NewBox(gtk.OrientationVertical, 8)

	// Large picture — loaded from file, no bytes held in Go heap
	picture := gtk.NewPicture()
	picture.SetCanShrink(true)
	picture.SetContentFit(gtk.ContentFitContain)
	picture.SetVExpand(true)
	picture.SetHExpand(true)

	texture, err := gdk.NewTextureFromFilename(imagePath)
	if err != nil {
		log.Printf("image_viewer: load texture from %s: %v", imagePath, err)
		errLabel := gtk.NewLabel("Failed to load image")
		content.Append(errLabel)
	} else {
		picture.SetPaintable(texture)
	}
	content.Append(picture)

	// Save button — copies file instead of writing bytes
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

		fileDialog.Save(context.Background(), nil, func(res gio.AsyncResulter) {
			file, err := fileDialog.SaveFinish(res)
			if err != nil {
				return
			}
			destPath := file.Path()
			if destPath == "" {
				return
			}
			if err := copyFile(imagePath, destPath); err != nil {
				log.Printf("image_viewer: save file %s: %v", destPath, err)
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

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
