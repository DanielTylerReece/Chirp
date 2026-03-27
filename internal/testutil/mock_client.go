package testutil

import (
	"sync"

	"github.com/tyler/gmessage/internal/backend"
)

// Compile-time assertion: MockClient implements backend.GMClient.
var _ backend.GMClient = (*MockClient)(nil)

// MethodCall records a single method invocation on MockClient.
type MethodCall struct {
	Method string
	Args   []any
}

// MockClient is a test double for backend.GMClient.
// It records all calls and returns configurable values.
type MockClient struct {
	mu           sync.Mutex
	Calls        []MethodCall
	connected    bool
	loggedIn     bool
	eventHandler func(evt any)

	// Configurable return values
	ConnectErr          error
	StartLoginURL       string
	StartLoginErr       error
	SendMessageErr      error
	SendMediaMessageErr error
	ListConversationsResult []backend.ConversationData
	ListConversationsErr    error
	FetchMessagesResult     []backend.MessageData
	FetchMessagesErr        error
	ListContactsErr         error
	MarkReadErr             error
	SetTypingErr            error
	SendReactionErr         error
	DownloadMediaResult                []byte
	DownloadMediaErr                   error
	FetchParticipantThumbnailsResult   map[string][]byte
	FetchParticipantThumbnailsErr      error
	SIMsResult                         []backend.SIMInfo
	GetOrCreateConvResult   string
	GetOrCreateConvErr      error
}

// NewMockClient returns a MockClient with sensible defaults.
func NewMockClient() *MockClient {
	return &MockClient{
		StartLoginURL: "https://support.google.com/messages/?p=web_computer#?c=mock-qr-data",
	}
}

func (m *MockClient) record(method string, args ...any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls = append(m.Calls, MethodCall{Method: method, Args: args})
}

// CallCount returns the number of times the named method was called.
func (m *MockClient) CallCount(method string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	count := 0
	for _, c := range m.Calls {
		if c.Method == method {
			count++
		}
	}
	return count
}

// LastCall returns the most recent MethodCall for the named method, or nil.
func (m *MockClient) LastCall(method string) *MethodCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := len(m.Calls) - 1; i >= 0; i-- {
		if m.Calls[i].Method == method {
			return &m.Calls[i]
		}
	}
	return nil
}

// Reset clears all recorded calls.
func (m *MockClient) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls = nil
}

// --- GMClient interface implementation ---

func (m *MockClient) Connect() error {
	m.record("Connect")
	if m.ConnectErr == nil {
		m.connected = true
	}
	return m.ConnectErr
}

func (m *MockClient) Disconnect() {
	m.record("Disconnect")
	m.connected = false
}

func (m *MockClient) IsConnected() bool {
	return m.connected
}

func (m *MockClient) IsLoggedIn() bool {
	return m.loggedIn
}

func (m *MockClient) StartLogin() (string, error) {
	m.record("StartLogin")
	return m.StartLoginURL, m.StartLoginErr
}

func (m *MockClient) SetEventHandler(handler func(evt any)) {
	m.eventHandler = handler
}

func (m *MockClient) ListConversations(count int) ([]backend.ConversationData, error) {
	m.record("ListConversations", count)
	return m.ListConversationsResult, m.ListConversationsErr
}

func (m *MockClient) FetchMessages(conversationID string, count int64) ([]backend.MessageData, error) {
	m.record("FetchMessages", conversationID, count)
	return m.FetchMessagesResult, m.FetchMessagesErr
}

func (m *MockClient) SendMessage(conversationID string, text string, simNumber int32) error {
	m.record("SendMessage", conversationID, text, simNumber)
	return m.SendMessageErr
}

func (m *MockClient) SendMediaMessage(conversationID string, text string, mediaData []byte, fileName string, mimeType string, simNumber int32) error {
	m.record("SendMediaMessage", conversationID, text, mediaData, fileName, mimeType, simNumber)
	return m.SendMediaMessageErr
}

func (m *MockClient) GetSIMs() []backend.SIMInfo {
	m.record("GetSIMs")
	return m.SIMsResult
}

func (m *MockClient) ListContacts() error {
	m.record("ListContacts")
	return m.ListContactsErr
}

func (m *MockClient) MarkRead(conversationID string, messageID string) error {
	m.record("MarkRead", conversationID, messageID)
	return m.MarkReadErr
}

func (m *MockClient) SetTyping(conversationID string) error {
	m.record("SetTyping", conversationID)
	return m.SetTypingErr
}

func (m *MockClient) SendReaction(messageID string, emoji string) error {
	m.record("SendReaction", messageID, emoji)
	return m.SendReactionErr
}

func (m *MockClient) DownloadMedia(mediaID string, decryptKey []byte) ([]byte, error) {
	m.record("DownloadMedia", mediaID, decryptKey)
	return m.DownloadMediaResult, m.DownloadMediaErr
}

func (m *MockClient) FetchParticipantThumbnails(participantIDs []string) (map[string][]byte, error) {
	m.record("FetchParticipantThumbnails", participantIDs)
	return m.FetchParticipantThumbnailsResult, m.FetchParticipantThumbnailsErr
}

func (m *MockClient) GetOrCreateConversation(numbers []string) (string, error) {
	m.record("GetOrCreateConversation", numbers)
	return m.GetOrCreateConvResult, m.GetOrCreateConvErr
}

// --- Test helpers ---

// SimulateEvent fires an event through the registered event handler.
func (m *MockClient) SimulateEvent(evt any) {
	if m.eventHandler != nil {
		m.eventHandler(evt)
	}
}

// SetLoggedIn sets the loggedIn state directly.
func (m *MockClient) SetLoggedIn(v bool) {
	m.loggedIn = v
}

// SetConnected sets the connected state directly.
func (m *MockClient) SetConnected(v bool) {
	m.connected = v
}
