package preferences

import (
	"fmt"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

// PreferencesDialog is the application preferences using AdwPreferencesDialog.
type PreferencesDialog struct {
	*adw.PreferencesDialog

	// Widgets that need updating
	phoneRow  *adw.ActionRow
	cacheRow  *adw.ActionRow
	notifSwitch *adw.SwitchRow

	// Callbacks set by the caller
	OnUnpair     func()
	OnDeepSync   func()
	OnClearCache func()
}

// NewPreferencesDialog builds the preferences dialog and returns it.
// Call Present(parent) to show it.
func NewPreferencesDialog() *PreferencesDialog {
	pd := &PreferencesDialog{}
	pd.PreferencesDialog = adw.NewPreferencesDialog()
	pd.SetTitle("Preferences")
	pd.SetSearchEnabled(false)

	pd.addAccountPage()
	pd.addNotificationsPage()
	pd.addStoragePage()

	return pd
}

// --- Account page ---

func (pd *PreferencesDialog) addAccountPage() {
	page := adw.NewPreferencesPage()
	page.SetTitle("Account")
	page.SetIconName("system-users-symbolic")

	// Paired Device group
	group := adw.NewPreferencesGroup()
	group.SetTitle("Paired Device")

	pd.phoneRow = adw.NewActionRow()
	pd.phoneRow.SetTitle("Phone")
	pd.phoneRow.SetSubtitle("Not paired")
	group.Add(pd.phoneRow)

	unpairRow := adw.NewActionRow()
	unpairRow.SetTitle("Unpair")
	unpairRow.SetSubtitle("Remove pairing with phone")
	unpairBtn := gtk.NewButtonWithLabel("Unpair")
	unpairBtn.SetVAlign(gtk.AlignCenter)
	unpairBtn.AddCSSClass("destructive-action")
	unpairBtn.ConnectClicked(func() {
		if pd.OnUnpair != nil {
			pd.OnUnpair()
		}
	})
	unpairRow.AddSuffix(unpairBtn)
	group.Add(unpairRow)

	page.Add(group)
	pd.Add(page)
}

// --- Notifications page ---

func (pd *PreferencesDialog) addNotificationsPage() {
	page := adw.NewPreferencesPage()
	page.SetTitle("Notifications")
	page.SetIconName("preferences-system-notifications-symbolic")

	group := adw.NewPreferencesGroup()
	group.SetTitle("Notifications")

	pd.notifSwitch = adw.NewSwitchRow()
	pd.notifSwitch.SetTitle("Enable notifications")
	pd.notifSwitch.SetActive(true)
	group.Add(pd.notifSwitch)

	page.Add(group)
	pd.Add(page)
}

// --- Storage page ---

func (pd *PreferencesDialog) addStoragePage() {
	page := adw.NewPreferencesPage()
	page.SetTitle("Storage")
	page.SetIconName("drive-harddisk-symbolic")

	// Cache group
	cacheGroup := adw.NewPreferencesGroup()
	cacheGroup.SetTitle("Cache")

	pd.cacheRow = adw.NewActionRow()
	pd.cacheRow.SetTitle("Media cache")
	pd.cacheRow.SetSubtitle("Calculating...")
	cacheGroup.Add(pd.cacheRow)

	clearRow := adw.NewActionRow()
	clearRow.SetTitle("Clear cache")
	clearRow.SetSubtitle("Remove all cached media files")
	clearBtn := gtk.NewButtonWithLabel("Clear")
	clearBtn.SetVAlign(gtk.AlignCenter)
	clearBtn.ConnectClicked(func() {
		if pd.OnClearCache != nil {
			pd.OnClearCache()
		}
	})
	clearRow.AddSuffix(clearBtn)
	cacheGroup.Add(clearRow)

	page.Add(cacheGroup)

	// Sync group
	syncGroup := adw.NewPreferencesGroup()
	syncGroup.SetTitle("Sync")

	syncRow := adw.NewActionRow()
	syncRow.SetTitle("Full sync")
	syncRow.SetSubtitle("Re-download all conversations and messages")
	syncBtn := gtk.NewButtonWithLabel("Sync")
	syncBtn.SetVAlign(gtk.AlignCenter)
	syncBtn.ConnectClicked(func() {
		if pd.OnDeepSync != nil {
			pd.OnDeepSync()
		}
	})
	syncRow.AddSuffix(syncBtn)
	syncGroup.Add(syncRow)

	page.Add(syncGroup)
	pd.Add(page)
}

// --- Public setters for dynamic state ---

// SetPhoneInfo updates the paired device row with phone details.
func (pd *PreferencesDialog) SetPhoneInfo(info string) {
	if info == "" {
		pd.phoneRow.SetSubtitle("Not paired")
	} else {
		pd.phoneRow.SetSubtitle(info)
	}
}

// SetCacheSize updates the cache row subtitle with the current size.
func (pd *PreferencesDialog) SetCacheSize(bytes int64) {
	pd.cacheRow.SetSubtitle(formatBytes(bytes))
}

// NotificationsEnabled returns the current state of the notification toggle.
func (pd *PreferencesDialog) NotificationsEnabled() bool {
	return pd.notifSwitch.Active()
}

// SetNotificationsEnabled sets the notification toggle state.
func (pd *PreferencesDialog) SetNotificationsEnabled(enabled bool) {
	pd.notifSwitch.SetActive(enabled)
}

// ConnectNotificationsChanged calls fn when the notification switch changes.
func (pd *PreferencesDialog) ConnectNotificationsChanged(fn func(enabled bool)) {
	pd.notifSwitch.Connect("notify::active", func() {
		fn(pd.notifSwitch.Active())
	})
}

// formatBytes returns a human-readable size string.
func formatBytes(b int64) string {
	switch {
	case b >= 1<<30:
		return fmt.Sprintf("%.1f GB used", float64(b)/float64(1<<30))
	case b >= 1<<20:
		return fmt.Sprintf("%.1f MB used", float64(b)/float64(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%.1f KB used", float64(b)/float64(1<<10))
	default:
		return fmt.Sprintf("%d bytes used", b)
	}
}
