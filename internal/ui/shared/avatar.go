package shared

import (
	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
)

// ConfigureAvatar sets up an adw.Avatar for a conversation: uses "#" for
// unsaved phone numbers, loads a custom image from avatarPath if available,
// and clears any previous custom image otherwise.
func ConfigureAvatar(avatar *adw.Avatar, name string, avatarPath string) {
	text := name
	if LooksLikePhoneNumber(name) {
		text = "#"
	}
	avatar.SetText(text)
	avatar.SetCustomImage(nil)
	if avatarPath != "" {
		if tex, err := gdk.NewTextureFromFilename(avatarPath); err == nil {
			avatar.SetCustomImage(tex)
		}
	}
}

// LooksLikePhoneNumber returns true if the string contains 7+ digits (unsaved contacts).
func LooksLikePhoneNumber(s string) bool {
	digits := 0
	for _, c := range s {
		if c >= '0' && c <= '9' {
			digits++
		}
	}
	return digits >= 7
}
