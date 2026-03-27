package sidebar

import (
	"fmt"
	"time"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotk4/pkg/pango"
	"github.com/tyler/gmessage/internal/db"
)

// ConversationRow is a single conversation entry in the sidebar list.
type ConversationRow struct {
	row          *gtk.ListBoxRow
	avatar       *adw.Avatar
	nameLabel    *gtk.Label
	previewLabel *gtk.Label
	timeLabel    *gtk.Label
	unreadLabel  *gtk.Label
	convID       string
}

// NewConversationRow builds a row widget for a conversation.
func NewConversationRow(conv *db.Conversation) *ConversationRow {
	cr := &ConversationRow{convID: conv.ID}

	// Layout:
	// [Avatar 40px] [Name          Timestamp]
	//               [Preview         Badge  ]

	cr.row = gtk.NewListBoxRow()
	cr.row.AddCSSClass("conversation-row")

	hbox := gtk.NewBox(gtk.OrientationHorizontal, 12)
	hbox.SetMarginTop(8)
	hbox.SetMarginBottom(8)
	hbox.SetMarginStart(12)
	hbox.SetMarginEnd(12)

	// Avatar
	cr.avatar = adw.NewAvatar(40, conv.Name, true)
	hbox.Append(cr.avatar)

	// Right side: name/time on top, preview/badge on bottom
	vbox := gtk.NewBox(gtk.OrientationVertical, 2)
	vbox.SetHExpand(true)

	// Top row: name + timestamp
	topBox := gtk.NewBox(gtk.OrientationHorizontal, 0)
	cr.nameLabel = gtk.NewLabel(conv.Name)
	cr.nameLabel.SetXAlign(0)
	cr.nameLabel.SetHExpand(true)
	cr.nameLabel.SetEllipsize(pango.EllipsizeEnd)
	cr.nameLabel.AddCSSClass("conversation-name")
	topBox.Append(cr.nameLabel)

	cr.timeLabel = gtk.NewLabel(formatTimestamp(conv.LastMessageTS))
	cr.timeLabel.AddCSSClass("conversation-time")
	topBox.Append(cr.timeLabel)

	vbox.Append(topBox)

	// Bottom row: preview + unread badge
	bottomBox := gtk.NewBox(gtk.OrientationHorizontal, 8)
	cr.previewLabel = gtk.NewLabel(conv.LastMessagePreview)
	cr.previewLabel.SetXAlign(0)
	cr.previewLabel.SetHExpand(true)
	cr.previewLabel.SetEllipsize(pango.EllipsizeEnd)
	cr.previewLabel.AddCSSClass("conversation-preview")
	bottomBox.Append(cr.previewLabel)

	if conv.UnreadCount > 0 {
		cr.unreadLabel = gtk.NewLabel(fmt.Sprintf("%d", conv.UnreadCount))
		cr.unreadLabel.AddCSSClass("unread-badge")
		bottomBox.Append(cr.unreadLabel)
	}

	vbox.Append(bottomBox)
	hbox.Append(vbox)
	cr.row.SetChild(hbox)

	return cr
}

// Update refreshes the row with new conversation data.
func (cr *ConversationRow) Update(conv *db.Conversation) {
	cr.nameLabel.SetText(conv.Name)
	cr.previewLabel.SetText(conv.LastMessagePreview)
	cr.timeLabel.SetText(formatTimestamp(conv.LastMessageTS))
	cr.avatar.SetText(conv.Name)
}

// formatTimestamp converts a millisecond epoch to a human-readable relative time string.
func formatTimestamp(ms int64) string {
	if ms == 0 {
		return ""
	}
	t := time.UnixMilli(ms)
	now := time.Now()

	if t.YearDay() == now.YearDay() && t.Year() == now.Year() {
		return t.Format("3:04 PM")
	}
	yesterday := now.AddDate(0, 0, -1)
	if t.YearDay() == yesterday.YearDay() && t.Year() == yesterday.Year() {
		return "Yesterday"
	}
	if now.Sub(t) < 7*24*time.Hour {
		return t.Format("Mon")
	}
	return t.Format("Jan 2")
}
