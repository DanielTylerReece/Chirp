package sidebar

import (
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/tyler/gmessage/internal/db"
)

// Sidebar is the left panel containing the conversation list.
type Sidebar struct {
	*gtk.Box // main container

	listBox    *gtk.ListBox
	rows       map[string]*ConversationRow // keyed by conversation ID
	onSelected func(convID string)
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
		// Find which conversation this row belongs to
		for id, cr := range s.rows {
			if cr.row == row {
				s.onSelected(id)
				return
			}
		}
	})

	scrolled.SetChild(s.listBox)
	s.Append(scrolled)

	return s
}

// SetOnConversationSelected registers a callback for conversation selection.
func (s *Sidebar) SetOnConversationSelected(fn func(convID string)) {
	s.onSelected = fn
}

// UpdateConversations replaces the conversation list.
func (s *Sidebar) UpdateConversations(convs []db.Conversation) {
	// Clear existing
	for _, cr := range s.rows {
		s.listBox.Remove(cr.row)
	}
	s.rows = make(map[string]*ConversationRow)

	// Add new rows
	for i := range convs {
		cr := NewConversationRow(&convs[i])
		s.listBox.Append(cr.row)
		s.rows[convs[i].ID] = cr
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
