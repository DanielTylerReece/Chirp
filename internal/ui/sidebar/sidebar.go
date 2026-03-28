package sidebar

import (
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/tyler/gmessage/internal/db"
)

// Sidebar is the left panel containing the conversation list.
type Sidebar struct {
	*gtk.Box // main container

	listBox           *gtk.ListBox
	rows              map[string]*ConversationRow // keyed by conversation ID
	onSelected        func(convID string)
	onNewConversation func()
}

// NewSidebar creates the conversation list sidebar.
func NewSidebar() *Sidebar {
	s := &Sidebar{
		Box:  gtk.NewBox(gtk.OrientationVertical, 0),
		rows: make(map[string]*ConversationRow),
	}

	// Scrolled window for the list
	scrolled := gtk.NewScrolledWindow()
	scrolled.SetVExpand(true)
	scrolled.SetPolicy(gtk.PolicyNever, gtk.PolicyAutomatic)

	s.listBox = gtk.NewListBox()
	s.listBox.SetSelectionMode(gtk.SelectionSingle)
	s.listBox.AddCSSClass("navigation-sidebar")
	s.listBox.ConnectRowSelected(func(row *gtk.ListBoxRow) {
		if row == nil || s.onSelected == nil {
			return
		}
		// Use the row's Name property to find the conversation ID
		convID := row.Name()
		if convID != "" {
			s.onSelected(convID)
		}
	})

	scrolled.SetChild(s.listBox)

	// New conversation button in a semi-transparent bar overlaying bottom of list
	newBtnBar := gtk.NewBox(gtk.OrientationHorizontal, 0)
	newBtnBar.AddCSSClass("new-conversation-bar")
	newBtnBar.SetHAlign(gtk.AlignFill)
	newBtnBar.SetVAlign(gtk.AlignEnd)

	newBtn := gtk.NewButtonFromIconName("list-add-symbolic")
	newBtn.AddCSSClass("new-conversation-button")
	newBtn.AddCSSClass("suggested-action")
	newBtn.SetHAlign(gtk.AlignStart)
	newBtn.SetMarginStart(12)
	newBtn.SetMarginTop(8)
	newBtn.SetMarginBottom(8)
	newBtn.SetTooltipText("New Conversation")
	newBtn.ConnectClicked(func() {
		if s.onNewConversation != nil {
			s.onNewConversation()
		}
	})
	newBtnBar.Append(newBtn)

	overlay := gtk.NewOverlay()
	overlay.SetChild(scrolled)
	overlay.AddOverlay(newBtnBar)
	overlay.SetVExpand(true)
	s.Append(overlay)

	return s
}

// SetOnConversationSelected registers a callback for conversation selection.
func (s *Sidebar) SetOnConversationSelected(fn func(convID string)) {
	s.onSelected = fn
}

// SetOnNewConversation registers a callback for the new conversation button.
func (s *Sidebar) SetOnNewConversation(fn func()) {
	s.onNewConversation = fn
}

// UpdateConversations replaces the conversation list.
func (s *Sidebar) UpdateConversations(convs []db.Conversation) {
	// Build set of incoming conversation IDs
	incoming := make(map[string]bool, len(convs))
	for i := range convs {
		incoming[convs[i].ID] = true
	}

	// Remove rows that no longer exist
	for id, cr := range s.rows {
		if !incoming[id] {
			s.listBox.Remove(cr.row)
			delete(s.rows, id)
		}
	}

	// Update existing or create new rows
	for i := range convs {
		if cr, ok := s.rows[convs[i].ID]; ok {
			cr.Update(&convs[i])
		} else {
			cr := NewConversationRow(&convs[i])
			s.rows[convs[i].ID] = cr
		}
	}

	// Reorder: detach all and re-append in correct order
	for _, cr := range s.rows {
		s.listBox.Remove(cr.row)
	}
	for i := range convs {
		if cr, ok := s.rows[convs[i].ID]; ok {
			s.listBox.Append(cr.row)
		}
	}
}

// SelectConversation selects a conversation by ID.
func (s *Sidebar) SelectConversation(id string) {
	if cr, ok := s.rows[id]; ok {
		s.listBox.SelectRow(cr.row)
	}
}

// FocusSearch moves keyboard focus to the search entry.
// This is a no-op until the search entry is implemented.
func (s *Sidebar) FocusSearch() {
	// TODO: focus the search entry once sidebar search is implemented
}
