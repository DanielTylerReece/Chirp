package content

import (
	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/tyler/gmessage/internal/db"
)

// Content is the right pane showing messages for the selected conversation.
type Content struct {
	*gtk.Box

	// Header
	headerBar    *adw.HeaderBar
	headerAvatar *adw.Avatar
	headerName   *gtk.Label
	headerStatus *gtk.Label

	// Message list
	messageList *MessageList

	// Compose
	composeBar *ComposeBar

	// State
	activeConvID string

	// Callbacks
	onSend      func(convID string, req SendRequest)
	mediaLoader func(mediaID string, decryptKey []byte) ([]byte, error) // downloads media
}

// NewContent creates the content area with header, message list, and compose bar.
func NewContent() *Content {
	c := &Content{
		Box: gtk.NewBox(gtk.OrientationVertical, 0),
	}

	// Header area
	headerBox := gtk.NewBox(gtk.OrientationHorizontal, 8)
	headerBox.AddCSSClass("chat-header")
	headerBox.SetMarginStart(8)

	c.headerAvatar = adw.NewAvatar(32, "", true)
	headerBox.Append(c.headerAvatar)

	nameBox := gtk.NewBox(gtk.OrientationVertical, 0)
	c.headerName = gtk.NewLabel("")
	c.headerName.SetXAlign(0)
	c.headerName.AddCSSClass("chat-header-name")
	nameBox.Append(c.headerName)

	c.headerStatus = gtk.NewLabel("")
	c.headerStatus.SetXAlign(0)
	c.headerStatus.AddCSSClass("chat-header-status")
	nameBox.Append(c.headerStatus)

	headerBox.Append(nameBox)

	c.headerBar = adw.NewHeaderBar()
	c.headerBar.SetTitleWidget(headerBox)

	c.Append(c.headerBar)

	// Message list
	c.messageList = NewMessageList()
	c.Append(c.messageList.Box)

	// Compose bar
	c.composeBar = NewComposeBar()
	c.composeBar.SetOnSend(func(req SendRequest) {
		if c.activeConvID != "" && c.onSend != nil {
			c.onSend(c.activeConvID, req)
		}
	})
	c.Append(c.composeBar.OuterBox())

	return c
}

// SetConversation switches to a conversation.
func (c *Content) SetConversation(conv *db.Conversation) {
	c.activeConvID = conv.ID
	c.headerName.SetText(conv.Name)
	c.headerAvatar.SetText(conv.Name)
	if conv.IsRCS {
		c.headerStatus.SetText("RCS")
	} else {
		c.headerStatus.SetText("SMS")
	}
}

// SetSIMs configures the SIM selector in the compose bar.
func (c *Content) SetSIMs(sims []SIMOption, defaultSIMNumber int32) {
	c.composeBar.SetSIMs(sims, defaultSIMNumber)
}

// SetMediaLoader sets the function used to download media for display.
func (c *Content) SetMediaLoader(fn func(mediaID string, decryptKey []byte) ([]byte, error)) {
	c.mediaLoader = fn
	c.messageList.mediaLoader = fn
}

// SetMessages populates the message list.
func (c *Content) SetMessages(msgs []db.Message) {
	c.messageList.SetMessages(msgs)
}

// AppendMessage adds a new message to the bottom.
func (c *Content) AppendMessage(msg *db.Message) {
	c.messageList.AppendMessage(msg)
}

// SetOnSend sets the callback for when user sends a message.
func (c *Content) SetOnSend(fn func(convID string, req SendRequest)) {
	c.onSend = fn
}

// ReplyTo sets up a reply to a specific message.
func (c *Content) ReplyTo(messageID string, previewText string) {
	c.composeBar.SetReply(messageID, previewText)
}

// ActiveConversationID returns the currently displayed conversation.
func (c *Content) ActiveConversationID() string {
	return c.activeConvID
}
