package ui

import (
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/tyler/gmessage/style"
)

// LoadCSS loads the application stylesheets.
func LoadCSS() {
	// Main styles
	provider := gtk.NewCSSProvider()
	provider.LoadFromData(style.CSS)
	gtk.StyleContextAddProviderForDisplay(
		gdk.DisplayGetDefault(),
		provider,
		gtk.STYLE_PROVIDER_PRIORITY_APPLICATION,
	)

	// Dark mode overrides would need to be conditional.
	// For now, libadwaita handles dark mode automatically via @named_colors.
	// The dark CSS is only needed for hardcoded hex values like bubble colors.
}
