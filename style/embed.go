// Package style embeds the application CSS stylesheets.
package style

import _ "embed"

//go:embed style.css
var CSS string

//go:embed style-dark.css
var DarkCSS string
