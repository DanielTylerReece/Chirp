package content

import (
	"log"

	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

// MediaWidget displays an image inline in a message bubble.
type MediaWidget struct {
	*gtk.Box
	picture *gtk.Picture
	loading bool
}

// NewMediaWidget creates a new media display widget.
func NewMediaWidget() *MediaWidget {
	mw := &MediaWidget{
		Box: gtk.NewBox(gtk.OrientationVertical, 0),
	}
	mw.AddCSSClass("media-widget")

	mw.picture = gtk.NewPicture()
	mw.picture.SetCanShrink(true)
	mw.picture.SetContentFit(gtk.ContentFitContain)
	mw.picture.SetSizeRequest(-1, 200) // min height
	mw.Append(mw.picture)

	return mw
}

// LoadFromBytes loads image data into the widget.
func (mw *MediaWidget) LoadFromBytes(data []byte, mimeType string) {
	bytes := glib.NewBytesWithGo(data)
	texture, err := gdk.NewTextureFromBytes(bytes)
	if err != nil {
		log.Printf("media_widget: load from bytes (%s): %v", mimeType, err)
		return
	}
	mw.picture.SetPaintable(texture)
}

// LoadFromFile loads an image from a local file path.
func (mw *MediaWidget) LoadFromFile(path string) {
	mw.picture.SetFilename(path)
}

// SetLoading shows a loading state.
func (mw *MediaWidget) SetLoading(loading bool) {
	mw.loading = loading
	// Could show a spinner overlay in a future iteration
}
