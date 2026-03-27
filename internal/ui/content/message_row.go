package content

import (
	"time"

	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotk4/pkg/pango"
	"github.com/tyler/gmessage/internal/db"
)

// MessageRow represents a single chat bubble in the message list.
type MessageRow struct {
	row           *gtk.ListBoxRow
	bubble        *gtk.Box
	bodyLabel     *gtk.Label
	timeLabel     *gtk.Label
	participantID string
	timestampMS   int64
	isFromMe      bool
}

// NewMessageRow creates a chat bubble for a message.
func NewMessageRow(msg *db.Message, consecutive bool) *MessageRow {
	mr := &MessageRow{
		participantID: msg.ParticipantID,
		timestampMS:   msg.TimestampMS,
		isFromMe:      msg.IsFromMe,
	}

	mr.row = gtk.NewListBoxRow()
	mr.row.SetSelectable(false)
	mr.row.SetActivatable(false)

	// Outer box for alignment
	outerBox := gtk.NewBox(gtk.OrientationHorizontal, 0)
	outerBox.SetMarginStart(8)
	outerBox.SetMarginEnd(8)
	outerBox.SetMarginTop(1)
	outerBox.SetMarginBottom(1)

	if !consecutive {
		outerBox.SetMarginTop(4)
	}

	// Bubble container
	mr.bubble = gtk.NewBox(gtk.OrientationVertical, 2)
	mr.bubble.AddCSSClass("messagebubble")

	if msg.IsFromMe {
		mr.bubble.AddCSSClass("sent")
		outerBox.SetHAlign(gtk.AlignEnd)
	} else {
		mr.bubble.AddCSSClass("received")
		outerBox.SetHAlign(gtk.AlignStart)
	}

	if consecutive {
		mr.bubble.AddCSSClass("consecutive")
	}

	// Body text
	if msg.Body != "" {
		mr.bodyLabel = gtk.NewLabel(msg.Body)
		mr.bodyLabel.SetWrap(true)
		mr.bodyLabel.SetWrapMode(pango.WrapWordChar)
		mr.bodyLabel.SetXAlign(0)
		mr.bodyLabel.SetSelectable(true)
		mr.bubble.Append(mr.bodyLabel)
	}

	// Media placeholder
	if msg.MediaID != "" {
		mediaLabel := gtk.NewLabel("[Media: " + msg.MediaMimeType + "]")
		mediaLabel.AddCSSClass("media-placeholder")
		mr.bubble.Append(mediaLabel)
	}

	// Timestamp + status
	infoBox := gtk.NewBox(gtk.OrientationHorizontal, 4)
	infoBox.SetHAlign(gtk.AlignEnd)

	mr.timeLabel = gtk.NewLabel(formatMessageTime(msg.TimestampMS))
	mr.timeLabel.AddCSSClass("message-time")
	mr.timeLabel.SetOpacity(0.6)
	infoBox.Append(mr.timeLabel)

	if msg.IsFromMe {
		statusLabel := gtk.NewLabel(statusIcon(msg.Status))
		statusLabel.SetOpacity(0.6)
		infoBox.Append(statusLabel)
	}

	mr.bubble.Append(infoBox)

	// Reactions
	if msg.Reactions != "" && msg.Reactions != "[]" {
		reactBox := gtk.NewBox(gtk.OrientationHorizontal, 4)
		reactBox.AddCSSClass("reactions-container")
		reactLabel := gtk.NewLabel(msg.Reactions)
		reactLabel.AddCSSClass("reaction-pill")
		reactBox.Append(reactLabel)
		mr.bubble.Append(reactBox)
	}

	outerBox.Append(mr.bubble)
	mr.row.SetChild(outerBox)

	return mr
}

func formatMessageTime(ms int64) string {
	return time.UnixMilli(ms).Format("3:04 PM")
}

func statusIcon(status int) string {
	switch status {
	case 0:
		return "\u23F3" // hourglass — sending
	case 1:
		return "\u2713" // check — sent
	case 2:
		return "\u2713\u2713" // double check — delivered
	case 3:
		return "\u2713\u2713" // double check — read (colored via CSS)
	case 4:
		return "\u2717" // x — failed
	default:
		return ""
	}
}
