package ui

import (
	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/tyler/gmessage/internal/app"
	"github.com/tyler/gmessage/internal/db"
	"github.com/tyler/gmessage/internal/ui/content"
	"github.com/tyler/gmessage/internal/ui/preferences"
	"github.com/tyler/gmessage/internal/ui/sidebar"
)

// Window is the main application window with a split-view layout:
// sidebar (conversation list) on the left, content (messages) on the right.
type Window struct {
	win       *adw.ApplicationWindow
	splitView *adw.NavigationSplitView
	sidebar   *sidebar.Sidebar
	content   *content.Content
	config    *app.Config

	// Callbacks for shortcut-triggered actions
	onShowPreferences func()
	onNewConversation func()
}

// NewWindow builds the main window widget tree and returns a Window.
func NewWindow(application *gtk.Application, cfg *app.Config) *Window {
	win := adw.NewApplicationWindow(application)
	win.SetTitle("GMessage")

	// Restore saved window state
	state := cfg.LoadWindowState()
	win.SetDefaultSize(state.Width, state.Height)
	if state.Maximized {
		win.Maximize()
	}

	// --- Sidebar pane ---
	sb := sidebar.NewSidebar()

	sidebarToolbar := adw.NewToolbarView()
	sidebarToolbar.AddTopBar(adw.NewHeaderBar())
	sidebarToolbar.SetContent(sb.Box)

	sidebarPage := adw.NewNavigationPage(sidebarToolbar, "GMessage")

	// --- Content pane ---
	ct := content.NewContent()

	contentPage := adw.NewNavigationPage(ct, "Messages")

	// --- Split view ---
	splitView := adw.NewNavigationSplitView()
	splitView.SetMinSidebarWidth(280)
	splitView.SetMaxSidebarWidth(400)
	splitView.SetSidebar(sidebarPage)
	splitView.SetContent(contentPage)

	win.SetContent(splitView)

	w := &Window{
		win:       win,
		splitView: splitView,
		sidebar:   sb,
		content:   ct,
		config:    cfg,
	}

	w.setupKeyboardShortcuts()
	w.setupWindowStatePersistence()

	return w
}

// setupKeyboardShortcuts registers global keyboard shortcuts on the window.
func (w *Window) setupKeyboardShortcuts() {
	controller := gtk.NewShortcutController()
	controller.SetScope(gtk.ShortcutScopeGlobal)

	// Ctrl+, — Open preferences
	prefsShortcut := gtk.NewShortcut(
		gtk.NewShortcutTriggerParseString("<Control>comma"),
		gtk.NewCallbackAction(func(_ gtk.Widgetter, _ *glib.Variant) bool {
			w.ShowPreferences()
			return true
		}),
	)
	controller.AddShortcut(prefsShortcut)

	// Ctrl+K — Focus sidebar search
	searchShortcut := gtk.NewShortcut(
		gtk.NewShortcutTriggerParseString("<Control>k"),
		gtk.NewCallbackAction(func(_ gtk.Widgetter, _ *glib.Variant) bool {
			w.sidebar.FocusSearch()
			return true
		}),
	)
	controller.AddShortcut(searchShortcut)

	// Ctrl+N — New conversation
	newConvShortcut := gtk.NewShortcut(
		gtk.NewShortcutTriggerParseString("<Control>n"),
		gtk.NewCallbackAction(func(_ gtk.Widgetter, _ *glib.Variant) bool {
			if w.onNewConversation != nil {
				w.onNewConversation()
			}
			return true
		}),
	)
	controller.AddShortcut(newConvShortcut)

	w.win.AddController(controller)
}

// setupWindowStatePersistence saves window geometry on close.
func (w *Window) setupWindowStatePersistence() {
	w.win.ConnectCloseRequest(func() bool {
		width := w.win.Width()
		height := w.win.Height()
		maximized := w.win.IsMaximized()

		// Don't save maximized dimensions — they're meaningless for restore
		if !maximized {
			_ = w.config.SaveWindowState(app.WindowState{
				Width:     width,
				Height:    height,
				Maximized: false,
			})
		} else {
			// Preserve existing width/height, just flag maximized
			prev := w.config.LoadWindowState()
			_ = w.config.SaveWindowState(app.WindowState{
				Width:     prev.Width,
				Height:    prev.Height,
				Maximized: true,
			})
		}
		return false // don't block close
	})
}

// ShowPreferences opens the preferences dialog.
func (w *Window) ShowPreferences() {
	if w.onShowPreferences != nil {
		w.onShowPreferences()
		return
	}
	// Default: show a bare preferences dialog
	pd := preferences.NewPreferencesDialog()
	pd.Present(w.win)
}

// SetOnShowPreferences overrides the default preferences action.
// Used by main.go to wire up preferences with app-level state.
func (w *Window) SetOnShowPreferences(fn func()) {
	w.onShowPreferences = fn
}

// SetOnNewConversation sets the callback for Ctrl+N.
func (w *Window) SetOnNewConversation(fn func()) {
	w.onNewConversation = fn
}

// Present shows the window.
func (w *Window) Present() {
	w.win.Present()
}

// ApplicationWindow returns the underlying adw.ApplicationWindow.
func (w *Window) ApplicationWindow() *adw.ApplicationWindow {
	return w.win
}

// SetOnConversationSelected sets the callback for conversation selection.
func (w *Window) SetOnConversationSelected(fn func(convID string)) {
	w.sidebar.SetOnConversationSelected(fn)
}

// UpdateConversations updates the sidebar conversation list.
func (w *Window) UpdateConversations(convs []db.Conversation) {
	w.sidebar.UpdateConversations(convs)
}

// SetMessages updates the message view for the active conversation.
func (w *Window) SetMessages(msgs []db.Message) {
	w.content.SetMessages(msgs)
}

// SetConversation switches the content pane to a conversation.
func (w *Window) SetConversation(conv *db.Conversation) {
	w.content.SetConversation(conv)
}

// SetOnSend sets the callback for sending messages.
func (w *Window) SetOnSend(fn func(convID string, req content.SendRequest)) {
	w.content.SetOnSend(fn)
}
