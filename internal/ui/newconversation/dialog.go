package newconversation

import (
	"unicode"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

// ContactResult represents a contact that can be selected for a new conversation.
type ContactResult struct {
	Name           string
	PhoneNumber    string
	ConversationID string // if set, opens existing conversation directly
}

// Dialog is the new-conversation dialog with contact search, selection chips,
// and a create button.
type Dialog struct {
	dialog      *adw.Dialog
	searchEntry *gtk.SearchEntry
	resultsList *gtk.ListBox
	selectedBox *gtk.FlowBox
	selected    []ContactResult
	createBtn   *gtk.Button

	// Last search results (for row activation lookup)
	currentResults []ContactResult

	onSearch func(query string) []ContactResult
	onCreate func(contacts []ContactResult)
}

// NewDialog creates the new-conversation dialog. Call Show() to present it.
func NewDialog() *Dialog {
	d := &Dialog{}

	// Main content box
	content := gtk.NewBox(gtk.OrientationVertical, 8)
	content.SetMarginTop(16)
	content.SetMarginBottom(16)
	content.SetMarginStart(16)
	content.SetMarginEnd(16)

	// "To:" label + selected contacts chips
	toBox := gtk.NewBox(gtk.OrientationHorizontal, 8)
	toBox.SetVAlign(gtk.AlignStart)
	toLabel := gtk.NewLabel("To:")
	toLabel.AddCSSClass("heading")
	toLabel.SetVAlign(gtk.AlignCenter)
	toBox.Append(toLabel)

	d.selectedBox = gtk.NewFlowBox()
	d.selectedBox.SetSelectionMode(gtk.SelectionNone)
	d.selectedBox.SetMaxChildrenPerLine(5)
	d.selectedBox.SetHExpand(true)
	d.selectedBox.SetMinChildrenPerLine(1)
	toBox.Append(d.selectedBox)
	content.Append(toBox)

	// Search entry
	d.searchEntry = gtk.NewSearchEntry()
	d.searchEntry.SetPlaceholderText("Search contacts or enter phone number...")
	d.searchEntry.ConnectSearchChanged(func() {
		d.doSearch()
	})
	// Enter key: add raw phone number if it looks like one
	d.searchEntry.ConnectActivate(func() {
		query := d.searchEntry.Text()
		if query == "" {
			return
		}
		// If there are results, select the first one
		if len(d.currentResults) > 0 {
			d.addSelected(d.currentResults[0])
			return
		}
		// Otherwise treat as raw phone number
		if looksLikePhone(query) {
			d.addSelected(ContactResult{Name: query, PhoneNumber: query})
		}
	})
	content.Append(d.searchEntry)

	// Results list (scrollable)
	scrolled := gtk.NewScrolledWindow()
	scrolled.SetVExpand(true)
	scrolled.SetPolicy(gtk.PolicyNever, gtk.PolicyAutomatic)

	d.resultsList = gtk.NewListBox()
	d.resultsList.SetSelectionMode(gtk.SelectionNone)
	d.resultsList.AddCSSClass("boxed-list")
	d.resultsList.ConnectRowActivated(func(row *gtk.ListBoxRow) {
		phone := row.Name()
		for _, r := range d.currentResults {
			if r.PhoneNumber == phone {
				d.addSelected(r)
				return
			}
		}
		// Fallback: raw phone number
		if phone != "" {
			d.addSelected(ContactResult{Name: phone, PhoneNumber: phone})
		}
	})
	scrolled.SetChild(d.resultsList)
	content.Append(scrolled)

	// Create button
	d.createBtn = gtk.NewButtonWithLabel("Start Conversation")
	d.createBtn.AddCSSClass("suggested-action")
	d.createBtn.AddCSSClass("pill")
	d.createBtn.SetHAlign(gtk.AlignCenter)
	d.createBtn.SetMarginTop(8)
	d.createBtn.SetSensitive(false)
	d.createBtn.ConnectClicked(func() {
		if d.onCreate != nil && len(d.selected) > 0 {
			d.onCreate(d.selected)
			d.dialog.ForceClose()
		}
	})
	content.Append(d.createBtn)

	// Build the dialog
	d.dialog = adw.NewDialog()
	d.dialog.SetTitle("New Conversation")
	d.dialog.SetContentWidth(400)
	d.dialog.SetContentHeight(500)

	toolbar := adw.NewToolbarView()
	toolbar.AddTopBar(adw.NewHeaderBar())
	toolbar.SetContent(content)

	d.dialog.SetChild(toolbar)

	return d
}

func (d *Dialog) doSearch() {
	query := d.searchEntry.Text()
	if query == "" || d.onSearch == nil {
		d.resultsList.RemoveAll()
		d.currentResults = nil
		return
	}

	results := d.onSearch(query)
	d.resultsList.RemoveAll()
	d.currentResults = results

	for _, r := range results {
		contact := r // capture
		row := d.makeResultRow(contact)
		d.resultsList.Append(row)
	}

	// Also add a "raw number" option if input looks like a phone number
	if looksLikePhone(query) {
		raw := ContactResult{Name: query, PhoneNumber: query}
		row := d.makeResultRow(raw)
		d.resultsList.Append(row)
		d.currentResults = append(d.currentResults, raw)
	}
}

func (d *Dialog) makeResultRow(contact ContactResult) *gtk.ListBoxRow {
	row := gtk.NewListBoxRow()
	row.SetActivatable(true)

	box := gtk.NewBox(gtk.OrientationVertical, 2)
	box.SetMarginTop(8)
	box.SetMarginBottom(8)
	box.SetMarginStart(12)
	box.SetMarginEnd(12)

	nameLabel := gtk.NewLabel(contact.Name)
	nameLabel.SetXAlign(0)
	nameLabel.AddCSSClass("heading")
	box.Append(nameLabel)

	if contact.PhoneNumber != "" && contact.PhoneNumber != contact.Name {
		phoneLabel := gtk.NewLabel(contact.PhoneNumber)
		phoneLabel.SetXAlign(0)
		phoneLabel.AddCSSClass("dim-label")
		box.Append(phoneLabel)
	}

	row.SetChild(box)
	row.SetName(contact.PhoneNumber)

	return row
}

func (d *Dialog) addSelected(contact ContactResult) {
	// Check if already selected
	for _, s := range d.selected {
		if s.PhoneNumber == contact.PhoneNumber {
			return
		}
	}
	d.selected = append(d.selected, contact)
	d.updateSelectedChips()
	d.createBtn.SetSensitive(true)
	d.searchEntry.SetText("")
}

func (d *Dialog) removeSelected(phone string) {
	var filtered []ContactResult
	for _, s := range d.selected {
		if s.PhoneNumber != phone {
			filtered = append(filtered, s)
		}
	}
	d.selected = filtered
	d.updateSelectedChips()
	d.createBtn.SetSensitive(len(d.selected) > 0)
}

func (d *Dialog) updateSelectedChips() {
	d.selectedBox.RemoveAll()

	for _, s := range d.selected {
		contact := s // capture
		chip := gtk.NewBox(gtk.OrientationHorizontal, 4)
		chip.AddCSSClass("contact-chip")

		label := gtk.NewLabel(contact.Name)
		chip.Append(label)

		removeBtn := gtk.NewButtonFromIconName("window-close-symbolic")
		removeBtn.AddCSSClass("flat")
		removeBtn.AddCSSClass("circular")
		removeBtn.ConnectClicked(func() {
			d.removeSelected(contact.PhoneNumber)
		})
		chip.Append(removeBtn)

		d.selectedBox.Insert(chip, -1)
	}
}

// looksLikePhone returns true if the string contains at least 7 digits.
func looksLikePhone(s string) bool {
	digits := 0
	for _, c := range s {
		if unicode.IsDigit(c) {
			digits++
		}
	}
	return digits >= 7
}

// SetOnSearch sets the callback invoked when the user types in the search entry.
// It should return matching contacts from the local database.
func (d *Dialog) SetOnSearch(fn func(query string) []ContactResult) {
	d.onSearch = fn
}

// SetOnCreate sets the callback invoked when the user clicks "Start Conversation".
func (d *Dialog) SetOnCreate(fn func(contacts []ContactResult)) {
	d.onCreate = fn
}

// Show presents the dialog over the given parent widget.
func (d *Dialog) Show(parent gtk.Widgetter) {
	d.dialog.Present(parent)
}
