package content

import (
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

// TypingIndicator shows animated dots when someone is typing.
type TypingIndicator struct {
	*gtk.Box
	visible bool
}

// NewTypingIndicator creates a typing indicator with three dots.
func NewTypingIndicator() *TypingIndicator {
	ti := &TypingIndicator{
		Box: gtk.NewBox(gtk.OrientationHorizontal, 4),
	}
	ti.AddCSSClass("messagebubble")
	ti.AddCSSClass("received")
	ti.SetHAlign(gtk.AlignStart)
	ti.SetMarginStart(8)
	ti.SetMarginBottom(4)

	for i := 0; i < 3; i++ {
		dot := gtk.NewBox(gtk.OrientationHorizontal, 0)
		dot.AddCSSClass("typing-dot")
		ti.Append(dot)
	}

	ti.SetVisible(false)
	return ti
}

// Show makes the typing indicator visible.
func (ti *TypingIndicator) Show() {
	ti.SetVisible(true)
	ti.visible = true
}

// Hide hides the typing indicator.
func (ti *TypingIndicator) Hide() {
	ti.SetVisible(false)
	ti.visible = false
}

// IsShown returns whether the typing indicator is currently visible.
func (ti *TypingIndicator) IsShown() bool {
	return ti.visible
}
