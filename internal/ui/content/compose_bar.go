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
	SIMNumber int32 // 1-indexed SIM slot (0 = use conversation default)
}

// SIMOption represents one selectable SIM for the compose bar.
type SIMOption struct {
	SIMNumber int32
	Label     string // Short display label (e.g., "US Mobile (865) 320-5104")
}

// ComposeBar is the message input area at the bottom of the content pane.
type ComposeBar struct {
	*gtk.Box

	// Main input row
	textView   *gtk.TextView
	scrolled   *gtk.ScrolledWindow
	sendButton *gtk.Button
	attachBtn  *gtk.Button
	emojiBtn   *gtk.MenuButton // Emoji picker
	simButton  *gtk.Button     // SIM selector
	simPopover *gtk.Popover    // SIM dropdown

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

	// SIM selection
	availableSIMs []SIMOption
	selectedSIM   int // index into availableSIMs, -1 = none/default

	onSend func(req SendRequest)
}

// NewComposeBar creates the compose bar with text input and send button.
func NewComposeBar() *ComposeBar {
	cb := &ComposeBar{
		selectedSIM: -1,
	}

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

	// Input row (horizontal: attach + text + right column)
	cb.Box = gtk.NewBox(gtk.OrientationHorizontal, 4)
	cb.Box.SetMarginStart(8)
	cb.Box.SetMarginEnd(8)
	cb.Box.SetMarginTop(4)
	cb.Box.SetMarginBottom(4)
	cb.Box.AddCSSClass("compose-bar")

	// Left column: emoji picker on top, attach button on bottom
	leftCol := gtk.NewBox(gtk.OrientationVertical, 0)
	leftCol.SetVAlign(gtk.AlignEnd)

	// Emoji picker
	emojiChooser := gtk.NewEmojiChooser()
	emojiChooser.ConnectEmojiPicked(func(text string) {
		cb.textView.Buffer().InsertAtCursor(text)
	})
	cb.emojiBtn = gtk.NewMenuButton()
	cb.emojiBtn.SetIconName("face-smile-symbolic")
	cb.emojiBtn.AddCSSClass("flat")
	cb.emojiBtn.AddCSSClass("compose-icon-btn")
	cb.emojiBtn.SetHasFrame(false)
	cb.emojiBtn.SetPopover(emojiChooser)
	leftCol.Append(cb.emojiBtn)

	// Attachment button
	cb.attachBtn = gtk.NewButtonFromIconName("mail-attachment-symbolic")
	cb.attachBtn.AddCSSClass("flat")
	cb.attachBtn.AddCSSClass("compose-icon-btn")
	cb.attachBtn.ConnectClicked(func() {
		cb.AttachFile()
	})
	leftCol.Append(cb.attachBtn)

	cb.Box.Append(leftCol)

	// Text input
	cb.textView = gtk.NewTextView()
	cb.textView.SetWrapMode(gtk.WrapWordChar)
	cb.textView.SetAcceptsTab(false)
	cb.textView.AddCSSClass("compose-text")
	cb.textView.SetVExpand(false)
	cb.textView.SetTopMargin(6)
	cb.textView.SetBottomMargin(6)
	cb.textView.SetLeftMargin(12)
	cb.textView.SetRightMargin(12)

	scrolled := gtk.NewScrolledWindow()
	scrolled.SetChild(cb.textView)
	scrolled.SetPolicy(gtk.PolicyNever, gtk.PolicyAutomatic)
	scrolled.SetHExpand(true)
	scrolled.SetVExpand(false)
	scrolled.AddCSSClass("compose-entry")
	cb.scrolled = scrolled

	// Dynamic height: grow as text wraps, cap at 5 rows
	cb.textView.Buffer().ConnectChanged(func() {
		cb.updateHeight()
	})

	cb.Box.Append(scrolled)

	// Right column: SIM picker on top, send button on bottom
	rightCol := gtk.NewBox(gtk.OrientationVertical, 0)
	rightCol.SetVAlign(gtk.AlignEnd)

	cb.simPopover = gtk.NewPopover()
	cb.simButton = gtk.NewButtonWithLabel("📱")
	cb.simButton.AddCSSClass("flat")
	cb.simButton.AddCSSClass("sim-selector")
	cb.simButton.SetVisible(false)
	cb.simButton.SetTooltipText("Select SIM card")
	cb.simButton.SetMarginBottom(8)
	cb.simButton.ConnectClicked(func() {
		cb.simPopover.Popup()
	})
	cb.simPopover.SetParent(cb.simButton)
	rightCol.Append(cb.simButton)

	cb.sendButton = gtk.NewButtonFromIconName("go-up-symbolic")
	cb.sendButton.AddCSSClass("compose-send-button")
	cb.sendButton.AddCSSClass("suggested-action")
	cb.sendButton.ConnectClicked(cb.doSend)
	rightCol.Append(cb.sendButton)

	cb.Box.Append(rightCol)

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
		SIMNumber: cb.SelectedSIMNumber(),
	}

	if cb.onSend != nil {
		cb.onSend(req)
	}

	// Clear everything
	buf.SetText("")
	cb.ClearAttachment()
	cb.ClearReply()
}

// updateHeight resizes the scrolled window to fit the text content,
// starting at 1 row and growing up to 5 rows.
func (cb *ComposeBar) updateHeight() {
	const lineHeight = 20
	const maxLines = 5
	const oneRowHeight = lineHeight + 4 // padding

	buf := cb.textView.Buffer()
	iter := buf.StartIter()
	lines := 1
	for cb.textView.ForwardDisplayLine(iter) {
		lines++
		if lines >= maxLines {
			break
		}
	}

	height := lines*lineHeight + 4
	if height < oneRowHeight {
		height = oneRowHeight
	}
	if height > maxLines*lineHeight+4 {
		height = maxLines*lineHeight + 4
	}
	cb.scrolled.SetSizeRequest(-1, height)
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

// SetSIMs configures the available SIMs and selects the default.
// Builds a popover dropdown listing each SIM. Selecting one updates the active SIM.
func (cb *ComposeBar) SetSIMs(sims []SIMOption, defaultSIMNumber int32) {
	cb.availableSIMs = sims
	cb.selectedSIM = -1

	if len(sims) < 2 {
		cb.simButton.SetVisible(false)
		if len(sims) == 1 {
			cb.selectedSIM = 0
		}
		return
	}

	// Find the default SIM
	for i, s := range sims {
		if s.SIMNumber == defaultSIMNumber {
			cb.selectedSIM = i
			break
		}
	}
	if cb.selectedSIM < 0 {
		cb.selectedSIM = 0
	}

	// Build popover with SIM list
	listBox := gtk.NewListBox()
	listBox.SetSelectionMode(gtk.SelectionNone)

	for i, sim := range sims {
		idx := i
		s := sim
		row := gtk.NewListBoxRow()
		label := gtk.NewLabel(s.Label)
		label.SetXAlign(0)
		label.SetMarginTop(6)
		label.SetMarginBottom(6)
		label.SetMarginStart(12)
		label.SetMarginEnd(12)
		row.SetChild(label)

		// Highlight current selection
		if idx == cb.selectedSIM {
			row.AddCSSClass("sim-selected")
		}
		listBox.Append(row)
	}

	listBox.ConnectRowActivated(func(row *gtk.ListBoxRow) {
		cb.selectedSIM = row.Index()
		cb.updateSIMTooltip()
		cb.simPopover.Popdown()
	})

	cb.simPopover.SetChild(listBox)

	cb.updateSIMTooltip()
	cb.simButton.SetVisible(true)
}

// updateSIMTooltip refreshes the SIM button tooltip with the current selection.
func (cb *ComposeBar) updateSIMTooltip() {
	if cb.selectedSIM < 0 || cb.selectedSIM >= len(cb.availableSIMs) {
		return
	}
	sim := cb.availableSIMs[cb.selectedSIM]
	cb.simButton.SetTooltipText("SIM: " + sim.Label)
}

// SelectedSIMNumber returns the currently selected SIM number (0 if none).
func (cb *ComposeBar) SelectedSIMNumber() int32 {
	if cb.selectedSIM < 0 || cb.selectedSIM >= len(cb.availableSIMs) {
		return 0
	}
	return cb.availableSIMs[cb.selectedSIM].SIMNumber
}
