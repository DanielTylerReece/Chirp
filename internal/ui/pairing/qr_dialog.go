package pairing

import (
	"fmt"
	"log"
	"strings"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/tyler/gmessage/internal/backend"
	"rsc.io/qr"
)

// QRDialog shows pairing options: QR code scanning or Google Account sign-in.
type QRDialog struct {
	dialog *adw.Dialog

	// QR page
	picture     *gtk.Picture
	qrStatus    *gtk.Label
	qrSpinner   *gtk.Spinner
	onCancel    func()

	// Google page
	cookieText       *gtk.TextView
	googleStatus     *gtk.Label
	emojiLabel       *gtk.Label
	signInBtn        *gtk.Button
	onGoogleLogin    func(cookies map[string]string)
	onFirefoxImport  func() (map[string]string, error)
}

// NewQRDialog creates a pairing dialog with QR and Google Account tabs.
func NewQRDialog() *QRDialog {
	qd := &QRDialog{}

	// === QR Code Page ===
	qrBox := gtk.NewBox(gtk.OrientationVertical, 16)
	qrBox.SetMarginTop(24)
	qrBox.SetMarginBottom(24)
	qrBox.SetMarginStart(24)
	qrBox.SetMarginEnd(24)
	qrBox.SetHAlign(gtk.AlignCenter)
	qrBox.SetVAlign(gtk.AlignCenter)

	qd.qrSpinner = gtk.NewSpinner()
	qd.qrSpinner.SetSizeRequest(48, 48)
	qd.qrSpinner.Start()
	qrBox.Append(qd.qrSpinner)

	qd.picture = gtk.NewPicture()
	qd.picture.SetCanShrink(true)
	qd.picture.SetSizeRequest(280, 280)
	qd.picture.SetVisible(false)
	qrBox.Append(qd.picture)

	qd.qrStatus = gtk.NewLabel("Connecting to Google Messages...")
	qd.qrStatus.SetWrap(true)
	qd.qrStatus.SetJustify(gtk.JustifyCenter)
	qd.qrStatus.SetMaxWidthChars(40)
	qd.qrStatus.AddCSSClass("body")
	qrBox.Append(qd.qrStatus)

	instrLabel := gtk.NewLabel("Open Google Messages on your phone\nSettings > Device pairing > QR code scanner")
	instrLabel.SetWrap(true)
	instrLabel.SetJustify(gtk.JustifyCenter)
	instrLabel.SetOpacity(0.6)
	qrBox.Append(instrLabel)

	// === Google Account Page ===
	googleBox := gtk.NewBox(gtk.OrientationVertical, 12)
	googleBox.SetMarginTop(24)
	googleBox.SetMarginBottom(24)
	googleBox.SetMarginStart(24)
	googleBox.SetMarginEnd(24)

	googleTitle := gtk.NewLabel("Sign in with Google Account")
	googleTitle.AddCSSClass("title-3")
	googleBox.Append(googleTitle)

	step1 := gtk.NewLabel("1. Sign into messages.google.com in your browser")
	step1.SetXAlign(0)
	step1.SetWrap(true)
	googleBox.Append(step1)

	openBtn := gtk.NewButtonWithLabel("Open Google Messages")
	openBtn.AddCSSClass("suggested-action")
	openBtn.AddCSSClass("pill")
	openBtn.SetHAlign(gtk.AlignCenter)
	openBtn.ConnectClicked(func() {
		gtk.ShowURI(nil, "https://messages.google.com", 0)
	})
	googleBox.Append(openBtn)

	// Browser-specific instructions with toggle
	chromeSteps := "2. Open DevTools (F12) > Application > Cookies > messages.google.com\n3. Copy the cookie values and paste below"
	firefoxSteps := "2. Open DevTools (F12) > Storage > Cookies > messages.google.com\n3. Copy the cookie values and paste below"

	stepsLabel := gtk.NewLabel(chromeSteps)
	stepsLabel.SetXAlign(0)
	stepsLabel.SetWrap(true)

	browserToggle := gtk.NewBox(gtk.OrientationHorizontal, 0)
	browserToggle.SetHAlign(gtk.AlignCenter)
	browserToggle.SetMarginTop(4)
	browserToggle.AddCSSClass("linked")

	chromeBtn := gtk.NewToggleButton()
	chromeBtn.SetLabel("Chrome")
	chromeBtn.SetActive(true)

	firefoxBtn := gtk.NewToggleButton()
	firefoxBtn.SetLabel("Firefox")
	firefoxBtn.SetGroup(chromeBtn)

	browserToggle.Append(chromeBtn)
	browserToggle.Append(firefoxBtn)
	googleBox.Append(browserToggle)
	googleBox.Append(stepsLabel)

	// Firefox/Zen auto-import button (only visible when Firefox selected)
	firefoxImportBtn := gtk.NewButtonWithLabel("Import from Firefox / Zen")
	firefoxImportBtn.AddCSSClass("pill")
	firefoxImportBtn.SetHAlign(gtk.AlignCenter)
	firefoxImportBtn.SetMarginTop(4)
	firefoxImportBtn.SetVisible(false)
	firefoxImportBtn.ConnectClicked(func() {
		if qd.onFirefoxImport != nil {
			cookies, err := qd.onFirefoxImport()
			if err != nil {
				qd.ShowError(err.Error())
				return
			}
			// Populate the text area with the imported cookies
			var lines []string
			for k, v := range cookies {
				lines = append(lines, k+":\""+v+"\"")
			}
			qd.cookieText.Buffer().SetText(strings.Join(lines, "\n"))
			qd.googleStatus.SetText("Cookies imported successfully")
			qd.googleStatus.RemoveCSSClass("error")
			qd.googleStatus.SetVisible(true)
		}
	})
	googleBox.Append(firefoxImportBtn)

	orLabel := gtk.NewLabel("— or paste cookies manually —")
	orLabel.SetOpacity(0.5)
	orLabel.SetMarginTop(8)
	orLabel.SetVisible(false)
	googleBox.Append(orLabel)

	// Show/hide import button based on browser toggle
	chromeBtn.ConnectToggled(func() {
		if chromeBtn.Active() {
			stepsLabel.SetText(chromeSteps)
			firefoxImportBtn.SetVisible(false)
			orLabel.SetVisible(false)
		}
	})
	firefoxBtn.ConnectToggled(func() {
		if firefoxBtn.Active() {
			stepsLabel.SetText(firefoxSteps)
			firefoxImportBtn.SetVisible(true)
			orLabel.SetVisible(true)
		}
	})

	// Cookie text area (fallback)
	qd.cookieText = gtk.NewTextView()
	qd.cookieText.SetWrapMode(gtk.WrapWordChar)
	qd.cookieText.SetTopMargin(8)
	qd.cookieText.SetBottomMargin(8)
	qd.cookieText.SetLeftMargin(8)
	qd.cookieText.SetRightMargin(8)
	qd.cookieText.SetMonospace(true)

	cookieScroll := gtk.NewScrolledWindow()
	cookieScroll.SetChild(qd.cookieText)
	cookieScroll.SetPolicy(gtk.PolicyNever, gtk.PolicyAutomatic)
	cookieScroll.SetMinContentHeight(80)
	cookieScroll.SetMaxContentHeight(120)
	cookieScroll.AddCSSClass("card")
	googleBox.Append(cookieScroll)

	requiredLabel := gtk.NewLabel("Paste all cookies from .google.com and messages.google.com")
	requiredLabel.SetXAlign(0)
	requiredLabel.SetOpacity(0.5)
	requiredLabel.AddCSSClass("caption")
	googleBox.Append(requiredLabel)

	// Sign in button
	qd.signInBtn = gtk.NewButtonWithLabel("Sign In")
	qd.signInBtn.AddCSSClass("suggested-action")
	qd.signInBtn.AddCSSClass("pill")
	qd.signInBtn.SetHAlign(gtk.AlignCenter)
	qd.signInBtn.SetMarginTop(8)
	qd.signInBtn.ConnectClicked(func() {
		qd.doGoogleLogin()
	})
	googleBox.Append(qd.signInBtn)

	// Emoji display (hidden initially)
	qd.emojiLabel = gtk.NewLabel("")
	qd.emojiLabel.AddCSSClass("title-1")
	qd.emojiLabel.SetVisible(false)
	qd.emojiLabel.SetMarginTop(8)
	googleBox.Append(qd.emojiLabel)

	// Status label
	qd.googleStatus = gtk.NewLabel("")
	qd.googleStatus.SetWrap(true)
	qd.googleStatus.SetJustify(gtk.JustifyCenter)
	qd.googleStatus.SetVisible(false)
	googleBox.Append(qd.googleStatus)

	// === View Stack with Switcher ===
	stack := adw.NewViewStack()
	qrPage := stack.AddTitledWithIcon(qrBox, "qr", "QR Code", "qr-code-symbolic")
	qrPage.SetIconName("camera-photo-symbolic")
	googlePage := stack.AddTitledWithIcon(googleBox, "google", "Google Account", "user-info-symbolic")
	_ = googlePage

	switcher := adw.NewViewSwitcher()
	switcher.SetStack(stack)
	switcher.SetPolicy(adw.ViewSwitcherPolicyWide)

	// === Dialog ===
	qd.dialog = adw.NewDialog()
	qd.dialog.SetTitle("Pair with Phone")
	qd.dialog.SetContentWidth(450)
	qd.dialog.SetContentHeight(550)

	headerBar := adw.NewHeaderBar()
	headerBar.SetTitleWidget(switcher)

	toolbar := adw.NewToolbarView()
	toolbar.AddTopBar(headerBar)
	toolbar.SetContent(stack)

	qd.dialog.SetChild(toolbar)

	return qd
}

func (qd *QRDialog) doGoogleLogin() {
	buf := qd.cookieText.Buffer()
	startIter := buf.StartIter()
	endIter := buf.EndIter()
	text := buf.Text(startIter, endIter, false)

	cookies, err := backend.ParseCookies(text)
	if err != nil {
		qd.ShowError(err.Error())
		return
	}

	qd.signInBtn.SetSensitive(false)
	qd.signInBtn.SetLabel("Connecting...")
	qd.googleStatus.SetVisible(false)

	if qd.onGoogleLogin != nil {
		qd.onGoogleLogin(cookies)
	}
}

// SetQRCode generates and displays a QR code from the given URL.
func (qd *QRDialog) SetQRCode(url string) {
	code, err := qr.Encode(url, qr.M)
	if err != nil {
		log.Printf("pairing: qr encode: %v", err)
		qd.SetStatus(fmt.Sprintf("Failed to generate QR code: %v", err))
		return
	}

	pngData := code.PNG()
	gBytes := glib.NewBytes(pngData)
	texture, err := gdk.NewTextureFromBytes(gBytes)
	if err != nil {
		log.Printf("pairing: texture from bytes: %v", err)
		qd.SetStatus(fmt.Sprintf("Failed to display QR code: %v", err))
		return
	}

	qd.qrSpinner.Stop()
	qd.qrSpinner.SetVisible(false)
	qd.picture.SetPaintable(texture)
	qd.picture.SetVisible(true)
}

// SetStatus updates the QR page status label.
func (qd *QRDialog) SetStatus(text string) {
	qd.qrStatus.SetText(text)
}

// ShowEmoji displays the verification emoji on the Google Account page.
func (qd *QRDialog) ShowEmoji(emoji string) {
	qd.signInBtn.SetLabel("Confirm on your phone")
	qd.signInBtn.SetSensitive(false)
	qd.emojiLabel.SetText(emoji)
	qd.emojiLabel.SetVisible(true)
	qd.googleStatus.SetText("Tap the matching emoji on your phone to complete pairing")
	qd.googleStatus.SetVisible(true)
}

// ShowError displays an error on the Google Account page.
func (qd *QRDialog) ShowError(msg string) {
	qd.signInBtn.SetSensitive(true)
	qd.signInBtn.SetLabel("Sign In")
	qd.emojiLabel.SetVisible(false)
	qd.googleStatus.SetText(msg)
	qd.googleStatus.AddCSSClass("error")
	qd.googleStatus.SetVisible(true)
}

// SetOnCancel sets the callback for when the user cancels pairing.
func (qd *QRDialog) SetOnCancel(fn func()) {
	qd.onCancel = fn
}

// SetOnGoogleLogin sets the callback for Google Account sign-in.
func (qd *QRDialog) SetOnGoogleLogin(fn func(cookies map[string]string)) {
	qd.onGoogleLogin = fn
}

// SetOnFirefoxImport sets the callback for importing cookies from Firefox.
func (qd *QRDialog) SetOnFirefoxImport(fn func() (map[string]string, error)) {
	qd.onFirefoxImport = fn
}

// Close dismisses the dialog.
func (qd *QRDialog) Close() {
	qd.dialog.ForceClose()
}

// Show presents the dialog as a sheet over the parent window.
func (qd *QRDialog) Show(parent gtk.Widgetter) {
	qd.dialog.Present(parent)
}
