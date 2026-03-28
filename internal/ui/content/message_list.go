package content

import (
	"time"

	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/tyler/gmessage/internal/db"
)

// MessageList is the scrollable area containing chat bubbles.
type MessageList struct {
	*gtk.Box

	scrolled       *gtk.ScrolledWindow
	listBox        *gtk.ListBox
	rows           []*MessageRow
	mediaLoader    func(mediaID string, decryptKey []byte) (string, error)
	lastFingerprint string // tracks message IDs+statuses to skip redundant rebuilds
}

// NewMessageList creates an empty scrollable message list.
func NewMessageList() *MessageList {
	ml := &MessageList{
		Box: gtk.NewBox(gtk.OrientationVertical, 0),
	}
	ml.Box.SetVExpand(true)

	ml.scrolled = gtk.NewScrolledWindow()
	ml.scrolled.SetVExpand(true)
	ml.scrolled.SetPolicy(gtk.PolicyNever, gtk.PolicyAutomatic)

	ml.listBox = gtk.NewListBox()
	ml.listBox.SetSelectionMode(gtk.SelectionNone)
	ml.listBox.AddCSSClass("message-list")

	ml.scrolled.SetChild(ml.listBox)
	ml.Box.Append(ml.scrolled)

	return ml
}

// msgFingerprint builds a fast identity string from message IDs and statuses.
func msgFingerprint(msgs []db.Message) string {
	// Preallocate: ~20 chars per message (ID hash + status digit + separator)
	buf := make([]byte, 0, len(msgs)*20)
	for i := range msgs {
		buf = append(buf, msgs[i].ID...)
		buf = append(buf, byte('0'+msgs[i].Status))
		buf = append(buf, ',')
	}
	return string(buf)
}

// SetMessages replaces all messages in the list, skipping if nothing changed.
func (ml *MessageList) SetMessages(msgs []db.Message) {
	fp := msgFingerprint(msgs)
	if fp == ml.lastFingerprint {
		return // nothing changed
	}
	ml.lastFingerprint = fp

	// Clear existing rows and date separators
	ml.listBox.RemoveAll()
	ml.rows = nil

	var lastSender string
	var lastTime int64
	var lastDate string

	for i := range msgs {
		msg := &msgs[i]

		// Date separator
		msgDate := time.UnixMilli(msg.TimestampMS).Format("2006-01-02")
		if msgDate != lastDate {
			ml.addDateSeparator(msg.TimestampMS)
			lastDate = msgDate
			lastSender = ""
			lastTime = 0
		}

		// Determine grouping: same sender within 60 seconds
		consecutive := msg.ParticipantID == lastSender && (msg.TimestampMS-lastTime) < 60000

		row := NewMessageRow(msg, consecutive, ml.mediaLoader)
		ml.listBox.Append(row.row)
		ml.rows = append(ml.rows, row)

		lastSender = msg.ParticipantID
		lastTime = msg.TimestampMS
	}

	// Scroll to bottom after layout
	ml.scrollToBottom()
}

// AppendMessage adds a single message to the end.
func (ml *MessageList) AppendMessage(msg *db.Message) {
	var consecutive bool
	if len(ml.rows) > 0 {
		last := ml.rows[len(ml.rows)-1]
		consecutive = last.participantID == msg.ParticipantID && (msg.TimestampMS-last.timestampMS) < 60000
	}

	row := NewMessageRow(msg, consecutive, ml.mediaLoader)
	ml.listBox.Append(row.row)
	ml.rows = append(ml.rows, row)

	ml.scrollToBottom()
}

func (ml *MessageList) addDateSeparator(timestampMS int64) {
	t := time.UnixMilli(timestampMS)
	now := time.Now()

	var text string
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	yesterday := today.AddDate(0, 0, -1)
	msgDay := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())

	if msgDay.Equal(today) {
		text = "Today"
	} else if msgDay.Equal(yesterday) {
		text = "Yesterday"
	} else if now.Sub(t) < 7*24*time.Hour {
		text = t.Format("Monday")
	} else {
		text = t.Format("January 2, 2006")
	}

	label := gtk.NewLabel(text)
	label.AddCSSClass("date-separator")

	row := gtk.NewListBoxRow()
	row.SetSelectable(false)
	row.SetActivatable(false)
	row.SetChild(label)
	ml.listBox.Append(row)
}

func (ml *MessageList) scrollToBottom() {
	adj := ml.scrolled.VAdjustment()
	if adj != nil {
		adj.SetValue(adj.Upper() - adj.PageSize())
	}
}
