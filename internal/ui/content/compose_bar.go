package content

import (
	"context"
	"log"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotk4/pkg/pango"
)

// SendRequest carries all compose data when the user sends a message.
type SendRequest struct {
	Text      string
	MediaData []byte
	MediaName string
	MediaMime string
	ReplyToID string
}

// ComposeBar is the message input area at the bottom of the content pane.
type ComposeBar struct {
	*gtk.Box

	// Main input row
	textView   *gtk.TextView
	sendButton *gtk.Button
	attachBtn  *gtk.Button

	// Overlay previews (shown above compose row when active)
	outerBox          *gtk.Box // vertical: previews + input row
	attachmentPreview *gtk.Box // shown above compose when a file is attached
	replyPreview      *gtk.Box // shown above compose when replying

	// Pending attachment
	attachedFile []byte
	attachedName string
	attachedMime string

	// Pending reply
	replyToID   string
	replyToText string

	onSend func(req SendRequest)
}

// NewComposeBar creates the compose bar with text input and send button.
func NewComposeBar() *ComposeBar {
	cb := &ComposeBar{}

	// Outer vertical box holds preview bars + input row
	cb.outerBox = gtk.NewBox(gtk.OrientationVertical, 0)
	cb.outerBox.AddCSSClass("compose-bar-outer")

	// Reply preview (hidden by default)
	cb.replyPreview = gtk.NewBox(gtk.OrientationHorizontal, 8)
	cb.replyPreview.AddCSSClass("reply-preview")
	cb.replyPreview.SetMarginStart(8)
	cb.replyPreview.SetMarginEnd(8)
	cb.replyPreview.SetMarginTop(4)
	cb.replyPreview.SetVisible(false)
	cb.outerBox.Append(cb.replyPreview)

	// Attachment preview (hidden by default)
	cb.attachmentPreview = gtk.NewBox(gtk.OrientationHorizontal, 8)
	cb.attachmentPreview.AddCSSClass("attachment-preview")
	cb.attachmentPreview.SetMarginStart(8)
	cb.attachmentPreview.SetMarginEnd(8)
	cb.attachmentPreview.SetMarginTop(4)
	cb.attachmentPreview.SetVisible(false)
	cb.outerBox.Append(cb.attachmentPreview)

	// Input row (horizontal: attach + text + send)
	cb.Box = gtk.NewBox(gtk.OrientationHorizontal, 8)
	cb.Box.SetMarginStart(8)
	cb.Box.SetMarginEnd(8)
	cb.Box.SetMarginTop(8)
	cb.Box.SetMarginBottom(8)
	cb.Box.AddCSSClass("compose-bar")

	// Attachment button
	cb.attachBtn = gtk.NewButtonFromIconName("mail-attachment-symbolic")
	cb.attachBtn.AddCSSClass("flat")
	cb.attachBtn.ConnectClicked(func() {
		cb.AttachFile()
	})
	cb.Box.Append(cb.attachBtn)

	// Text input
	cb.textView = gtk.NewTextView()
	cb.textView.SetWrapMode(gtk.WrapWordChar)
	cb.textView.SetAcceptsTab(false)
	cb.textView.AddCSSClass("compose-entry")
	cb.textView.SetVExpand(false)

	// Scrolled wrapper for text view (limits height)
	scrolled := gtk.NewScrolledWindow()
	scrolled.SetChild(cb.textView)
	scrolled.SetPolicy(gtk.PolicyNever, gtk.PolicyAutomatic)
	scrolled.SetMaxContentHeight(120)
	scrolled.SetPropagateNaturalHeight(true)
	scrolled.SetHExpand(true)
	cb.Box.Append(scrolled)

	// Send button
	cb.sendButton = gtk.NewButtonFromIconName("go-up-symbolic")
	cb.sendButton.AddCSSClass("compose-send-button")
	cb.sendButton.AddCSSClass("suggested-action")
	cb.sendButton.ConnectClicked(cb.doSend)
	cb.Box.Append(cb.sendButton)

	// Enter to send, Shift+Enter for newline
	keyController := gtk.NewEventControllerKey()
	keyController.ConnectKeyPressed(func(keyval, keycode uint, state gdk.ModifierType) bool {
		if keyval == gdk.KEY_Return && state&gdk.ShiftMask == 0 {
			cb.doSend()
			return true // consumed
		}
		return false
	})
	cb.textView.AddController(keyController)

	cb.outerBox.Append(cb.Box)

	return cb
}

// OuterBox returns the full compose bar widget (previews + input row).
func (cb *ComposeBar) OuterBox() *gtk.Box {
	return cb.outerBox
}

// SetOnSend sets the callback for when the user sends a message.
func (cb *ComposeBar) SetOnSend(fn func(req SendRequest)) {
	cb.onSend = fn
}

func (cb *ComposeBar) doSend() {
	buf := cb.textView.Buffer()
	start, end := buf.Bounds()
	text := strings.TrimSpace(buf.Text(start, end, false))

	// Nothing to send if no text and no attachment
	if text == "" && len(cb.attachedFile) == 0 {
		return
	}

	req := SendRequest{
		Text:      text,
		MediaData: cb.attachedFile,
		MediaName: cb.attachedName,
		MediaMime: cb.attachedMime,
		ReplyToID: cb.replyToID,
	}

	if cb.onSend != nil {
		cb.onSend(req)
	}

	// Clear everything
	buf.SetText("")
	cb.ClearAttachment()
	cb.ClearReply()
}

// AttachFile opens a file dialog and attaches the selected file.
func (cb *ComposeBar) AttachFile() {
	dialog := gtk.NewFileDialog()
	dialog.SetTitle("Attach File")

	// Parent is optional — pass nil; the dialog will still appear correctly.
	dialog.Open(context.Background(), nil, func(res gio.AsyncResulter) {
		file, err := dialog.OpenFinish(res)
		if err != nil {
			// User cancelled or error — silently ignore
			return
		}

		path := file.Path()
		if path == "" {
			return
		}

		data, err := os.ReadFile(path)
		if err != nil {
			log.Printf("compose_bar: read file %s: %v", path, err)
			return
		}

		name := filepath.Base(path)
		mimeType := mime.TypeByExtension(filepath.Ext(path))
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}

		cb.attachedFile = data
		cb.attachedName = name
		cb.attachedMime = mimeType
		cb.showAttachmentPreview()
	})
}

// SetReply sets the reply context (shown above compose bar).
func (cb *ComposeBar) SetReply(messageID string, previewText string) {
	cb.replyToID = messageID
	cb.replyToText = previewText
	cb.showReplyPreview()
}

// ClearReply removes the reply context.
func (cb *ComposeBar) ClearReply() {
	cb.replyToID = ""
	cb.replyToText = ""
	// Remove all children from the preview box
	for child := cb.replyPreview.FirstChild(); child != nil; child = cb.replyPreview.FirstChild() {
		cb.replyPreview.Remove(child)
	}
	cb.replyPreview.SetVisible(false)
}

// ClearAttachment removes the pending attachment.
func (cb *ComposeBar) ClearAttachment() {
	cb.attachedFile = nil
	cb.attachedName = ""
	cb.attachedMime = ""
	// Remove all children from the preview box
	for child := cb.attachmentPreview.FirstChild(); child != nil; child = cb.attachmentPreview.FirstChild() {
		cb.attachmentPreview.Remove(child)
	}
	cb.attachmentPreview.SetVisible(false)
}

func (cb *ComposeBar) showAttachmentPreview() {
	// Clear existing content
	for child := cb.attachmentPreview.FirstChild(); child != nil; child = cb.attachmentPreview.FirstChild() {
		cb.attachmentPreview.Remove(child)
	}

	icon := gtk.NewImageFromIconName("mail-attachment-symbolic")
	cb.attachmentPreview.Append(icon)

	label := gtk.NewLabel(cb.attachedName)
	label.SetEllipsize(pango.EllipsizeEnd)
	label.SetHExpand(true)
	label.SetXAlign(0)
	cb.attachmentPreview.Append(label)

	closeBtn := gtk.NewButtonFromIconName("window-close-symbolic")
	closeBtn.AddCSSClass("flat")
	closeBtn.ConnectClicked(func() {
		cb.ClearAttachment()
	})
	cb.attachmentPreview.Append(closeBtn)

	cb.attachmentPreview.SetVisible(true)
}

func (cb *ComposeBar) showReplyPreview() {
	// Clear existing content
	for child := cb.replyPreview.FirstChild(); child != nil; child = cb.replyPreview.FirstChild() {
		cb.replyPreview.Remove(child)
	}

	icon := gtk.NewImageFromIconName("mail-reply-sender-symbolic")
	cb.replyPreview.Append(icon)

	label := gtk.NewLabel(cb.replyToText)
	label.SetEllipsize(pango.EllipsizeEnd)
	label.SetHExpand(true)
	label.SetXAlign(0)
	label.AddCSSClass("dim-label")
	cb.replyPreview.Append(label)

	closeBtn := gtk.NewButtonFromIconName("window-close-symbolic")
	closeBtn.AddCSSClass("flat")
	closeBtn.ConnectClicked(func() {
		cb.ClearReply()
	})
	cb.replyPreview.Append(closeBtn)

	cb.replyPreview.SetVisible(true)
}

// SetText sets the compose text (for drafts).
func (cb *ComposeBar) SetText(text string) {
	cb.textView.Buffer().SetText(text)
}
