package backend

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"go.mau.fi/mautrix-gmessages/pkg/libgm"
	"go.mau.fi/mautrix-gmessages/pkg/libgm/gmproto"
)

// Compile-time assertion: RealClient implements GMClient.
var _ GMClient = (*RealClient)(nil)

// RealClient wraps libgm.Client and implements GMClient.
type RealClient struct {
	mu           sync.RWMutex
	client       *libgm.Client
	authData     *libgm.AuthData
	logger       zerolog.Logger
	eventHandler func(evt any)

	// SIM card metadata, populated from Settings events
	simsMu   sync.RWMutex
	simCards []*gmproto.SIMCard
}

// NewRealClient creates a new RealClient. If authData is nil, creates fresh auth data.
func NewRealClient(authData *libgm.AuthData, logger zerolog.Logger) *RealClient {
	if authData == nil {
		authData = libgm.NewAuthData()
	}
	rc := &RealClient{
		authData: authData,
		logger:   logger,
	}
	rc.client = libgm.NewClient(authData, nil, logger)
	rc.client.SetEventHandler(rc.handleEvent)
	return rc
}

func (rc *RealClient) handleEvent(evt any) {
	// Intercept Settings events to store SIM card info
	if settings, ok := evt.(*gmproto.Settings); ok {
		rc.simsMu.Lock()
		rc.simCards = settings.GetSIMCards()
		rc.simsMu.Unlock()
	}

	rc.mu.RLock()
	handler := rc.eventHandler
	rc.mu.RUnlock()
	if handler != nil {
		handler(evt)
	}
}

// AuthData returns the current auth data for persistence.
func (rc *RealClient) AuthData() *libgm.AuthData {
	return rc.authData
}

func (rc *RealClient) Connect() error {
	return rc.client.Connect()
}

func (rc *RealClient) Disconnect() {
	rc.client.Disconnect()
}

func (rc *RealClient) IsConnected() bool {
	return rc.client.IsConnected()
}

func (rc *RealClient) IsLoggedIn() bool {
	return rc.client.IsLoggedIn()
}

func (rc *RealClient) StartLogin() (string, error) {
	return rc.client.StartLogin()
}

func (rc *RealClient) SetEventHandler(handler func(evt any)) {
	rc.mu.Lock()
	rc.eventHandler = handler
	rc.mu.Unlock()
}

// GetSIMs returns the available SIM cards discovered from Settings events.
func (rc *RealClient) GetSIMs() []SIMInfo {
	rc.simsMu.RLock()
	defer rc.simsMu.RUnlock()

	var sims []SIMInfo
	for _, card := range rc.simCards {
		sd := card.GetSIMData()
		if sd == nil {
			continue
		}
		sp := sd.GetSIMPayload()
		if sp == nil {
			continue
		}
		sims = append(sims, SIMInfo{
			ParticipantID: card.GetSIMParticipant().GetID(),
			SIMNumber:     sp.GetSIMNumber(),
			CarrierName:   sd.GetCarrierName(),
			PhoneNumber:   sd.GetFormattedPhoneNumber(),
			ColorHex:      sd.GetColorHex(),
		})
	}
	return sims
}

// getSIMCardForNumber returns the stored SIMCard protobuf matching the given SIM number.
func (rc *RealClient) getSIMCardForNumber(simNumber int32) *gmproto.SIMCard {
	rc.simsMu.RLock()
	defer rc.simsMu.RUnlock()
	for _, card := range rc.simCards {
		if sd := card.GetSIMData(); sd != nil {
			if sp := sd.GetSIMPayload(); sp != nil && sp.GetSIMNumber() == simNumber {
				return card
			}
		}
	}
	// Fallback: return first SIM if available
	if len(rc.simCards) > 0 {
		return rc.simCards[0]
	}
	return nil
}

func (rc *RealClient) ListConversations(count int) ([]ConversationData, error) {
	resp, err := rc.client.ListConversations(count, gmproto.ListConversationsRequest_INBOX)
	if err != nil {
		return nil, err
	}

	var convs []ConversationData
	for _, c := range resp.GetConversations() {
		conv := ConversationData{
			ID:                c.GetConversationID(),
			Name:              c.GetName(),
			IsGroup:           c.GetIsGroupChat(),
			LastMessageTS:     c.GetLastMessageTimestamp() / 1000, // libgm returns microseconds
			Unread:            c.GetUnread(),
			IsPinned:          c.Pinned,
			IsArchived:        c.GetStatus() == gmproto.ConversationStatus_ARCHIVED || c.GetStatus() == gmproto.ConversationStatus_KEEP_ARCHIVED,
			IsRCS:             c.GetType() == gmproto.ConversationType_RCS,
			AvatarHexColor:    c.GetAvatarHexColor(),
			DefaultOutgoingID: c.GetDefaultOutgoingID(),
		}

		// Extract last message preview from LatestMessage
		if lm := c.GetLatestMessage(); lm != nil {
			conv.LastMessagePreview = lm.GetDisplayContent()
		}
		if conv.LastMessagePreview == "" {
			conv.LastMessagePreview = "\U0001F4F7 Photo" // fallback for media-only messages
		}

		// Map participants
		for _, p := range c.GetParticipants() {
			pd := ParticipantData{
				Name:           p.GetFullName(),
				IsMe:           p.GetIsMe(),
				AvatarHexColor: p.GetAvatarHexColor(),
				ContactID:      p.GetContactID(),
				PhoneNumber:    "",
			}
			// ID is a SmallInfo with ParticipantID and Number
			if si := p.GetID(); si != nil {
				pd.ID = si.GetParticipantID()
				pd.PhoneNumber = si.GetNumber()
			}
			// Extract SIM number from participant's SimPayload
			if sp := p.GetSimPayload(); sp != nil {
				pd.SIMNumber = sp.GetSIMNumber()
			}
			conv.Participants = append(conv.Participants, pd)
		}

		convs = append(convs, conv)
	}
	return convs, nil
}

func (rc *RealClient) FetchMessages(conversationID string, count int64) ([]MessageData, error) {
	resp, err := rc.client.FetchMessages(conversationID, count, nil)
	if err != nil {
		return nil, err
	}

	var msgs []MessageData
	for _, m := range resp.GetMessages() {
		md := MessageData{
			ID:             m.GetMessageID(),
			ConversationID: m.GetConversationID(),
			ParticipantID:  m.GetParticipantID(),
			TimestampMS:    m.GetTimestamp() / 1000, // libgm returns microseconds
		}

		// Determine if from me: outgoing statuses are < 100, incoming are >= 100
		if ms := m.GetMessageStatus(); ms != nil {
			statusVal := int(ms.GetStatus())
			md.IsFromMe = statusVal > 0 && statusVal < 100
			md.Status = ConvertMessageStatus(ms.GetStatus())
		}

		// Extract text body and media from MessageInfo.
		// When a single protobuf message contains multiple media (e.g., user
		// sent 3 photos at once), we store the first media on the primary
		// MessageData and create synthetic sub-messages for the rest so each
		// photo gets its own row in the DB/UI.
		var extraMedia []MessageData
		mediaIndex := 0
		for _, info := range m.GetMessageInfo() {
			if mc := info.GetMessageContent(); mc != nil {
				md.Body = mc.GetContent()
			}
			if media := info.GetMediaContent(); media != nil {
				if mediaIndex == 0 {
					// First media goes on the primary message
					md.MediaID = media.GetMediaID()
					md.MediaMimeType = media.GetMimeType()
					md.MediaDecryptKey = media.GetDecryptionKey()
					md.MediaSize = media.GetSize()
					if dims := media.GetDimensions(); dims != nil {
						md.MediaWidth = int(dims.GetWidth())
						md.MediaHeight = int(dims.GetHeight())
					}
					md.ThumbnailID = media.GetThumbnailMediaID()
					md.ThumbnailKey = media.GetThumbnailDecryptionKey()
				} else {
					// Additional media → synthetic sub-message
					extra := MessageData{
						ID:              md.ID + fmt.Sprintf("_media_%d", mediaIndex),
						ConversationID:  md.ConversationID,
						ParticipantID:   md.ParticipantID,
						TimestampMS:     md.TimestampMS + int64(mediaIndex), // tiny offset for ordering
						IsFromMe:        md.IsFromMe,
						Status:          md.Status,
						MediaID:         media.GetMediaID(),
						MediaMimeType:   media.GetMimeType(),
						MediaDecryptKey: media.GetDecryptionKey(),
						MediaSize:       media.GetSize(),
						ThumbnailID:     media.GetThumbnailMediaID(),
						ThumbnailKey:    media.GetThumbnailDecryptionKey(),
					}
					if dims := media.GetDimensions(); dims != nil {
						extra.MediaWidth = int(dims.GetWidth())
						extra.MediaHeight = int(dims.GetHeight())
					}
					extraMedia = append(extraMedia, extra)
				}
				mediaIndex++
			}
		}

		// Reply-to
		if rm := m.GetReplyMessage(); rm != nil {
			md.ReplyToID = rm.GetMessageID()
		}

		msgs = append(msgs, md)
		msgs = append(msgs, extraMedia...)
	}
	return msgs, nil
}

// ConvertMessageStatus maps libgm MessageStatusType to our simplified status ints:
// 0=sending, 1=sent, 2=delivered, 3=read, 4=failed
func ConvertMessageStatus(status gmproto.MessageStatusType) int {
	switch status {
	case gmproto.MessageStatusType_OUTGOING_YET_TO_SEND,
		gmproto.MessageStatusType_OUTGOING_SENDING,
		gmproto.MessageStatusType_OUTGOING_RESENDING,
		gmproto.MessageStatusType_OUTGOING_AWAITING_RETRY,
		gmproto.MessageStatusType_OUTGOING_SEND_AFTER_PROCESSING:
		return 0 // sending
	case gmproto.MessageStatusType_OUTGOING_COMPLETE:
		return 1 // sent
	case gmproto.MessageStatusType_OUTGOING_DELIVERED:
		return 2 // delivered
	case gmproto.MessageStatusType_OUTGOING_DISPLAYED:
		return 3 // read
	case gmproto.MessageStatusType_OUTGOING_FAILED_GENERIC,
		gmproto.MessageStatusType_OUTGOING_FAILED_EMERGENCY_NUMBER,
		gmproto.MessageStatusType_OUTGOING_FAILED_TOO_LARGE,
		gmproto.MessageStatusType_OUTGOING_FAILED_RECIPIENT_LOST_RCS,
		gmproto.MessageStatusType_OUTGOING_FAILED_NO_RETRY_NO_FALLBACK,
		gmproto.MessageStatusType_OUTGOING_FAILED_RECIPIENT_DID_NOT_DECRYPT,
		gmproto.MessageStatusType_OUTGOING_CANCELED:
		return 4 // failed
	case gmproto.MessageStatusType_INCOMING_COMPLETE,
		gmproto.MessageStatusType_INCOMING_DELIVERED,
		gmproto.MessageStatusType_INCOMING_DISPLAYED:
		return 2 // delivered (incoming)
	default:
		return 1 // default to sent
	}
}

func (rc *RealClient) SendMessage(conversationID string, text string, simNumber int32) error {
	tmpID := fmt.Sprintf("tmp_%d", time.Now().UnixNano())
	sim := rc.getSIMCardForNumber(simNumber)

	var simPayload *gmproto.SIMPayload
	var participantID string
	if sim != nil {
		if sd := sim.GetSIMData(); sd != nil {
			simPayload = sd.GetSIMPayload()
		}
		if sp := sim.GetSIMParticipant(); sp != nil {
			participantID = sp.GetID()
		}
	}

	_, err := rc.client.SendMessage(&gmproto.SendMessageRequest{
		ConversationID: conversationID,
		TmpID:          tmpID,
		SIMPayload:     simPayload,
		MessagePayload: &gmproto.MessagePayload{
			TmpID:          tmpID,
			TmpID2:         tmpID,
			ConversationID: conversationID,
			ParticipantID:  participantID,
			MessageInfo: []*gmproto.MessageInfo{{
				Data: &gmproto.MessageInfo_MessageContent{
					MessageContent: &gmproto.MessageContent{
						Content: text,
					},
				},
			}},
		},
	})
	return err
}

func (rc *RealClient) SendMediaMessage(conversationID string, text string, mediaData []byte, fileName string, mimeType string, simNumber int32) error {
	media, err := rc.client.UploadMedia(mediaData, fileName, mimeType)
	if err != nil {
		return fmt.Errorf("upload media: %w", err)
	}
	tmpID := fmt.Sprintf("tmp_%d", time.Now().UnixNano())
	sim := rc.getSIMCardForNumber(simNumber)

	var simPayload *gmproto.SIMPayload
	var participantID string
	if sim != nil {
		if sd := sim.GetSIMData(); sd != nil {
			simPayload = sd.GetSIMPayload()
		}
		if sp := sim.GetSIMParticipant(); sp != nil {
			participantID = sp.GetID()
		}
	}

	msgInfos := []*gmproto.MessageInfo{{
		Data: &gmproto.MessageInfo_MediaContent{
			MediaContent: media,
		},
	}}
	if text != "" {
		msgInfos = append(msgInfos, &gmproto.MessageInfo{
			Data: &gmproto.MessageInfo_MessageContent{
				MessageContent: &gmproto.MessageContent{
					Content: text,
				},
			},
		})
	}
	_, err = rc.client.SendMessage(&gmproto.SendMessageRequest{
		ConversationID: conversationID,
		TmpID:          tmpID,
		SIMPayload:     simPayload,
		MessagePayload: &gmproto.MessagePayload{
			TmpID:          tmpID,
			TmpID2:         tmpID,
			ConversationID: conversationID,
			ParticipantID:  participantID,
			MessageInfo:    msgInfos,
		},
	})
	return err
}

func (rc *RealClient) ListContacts() error {
	_, err := rc.client.ListContacts()
	return err
}

func (rc *RealClient) MarkRead(conversationID string, messageID string) error {
	return rc.client.MarkRead(conversationID, messageID)
}

func (rc *RealClient) SetTyping(conversationID string) error {
	return rc.client.SetTyping(conversationID, nil)
}

func (rc *RealClient) DownloadMedia(mediaID string, decryptKey []byte) ([]byte, error) {
	return rc.client.DownloadMedia(mediaID, decryptKey)
}

func (rc *RealClient) FetchParticipantThumbnails(participantIDs []string) (map[string][]byte, error) {
	if len(participantIDs) == 0 {
		return nil, nil
	}
	resp, err := rc.client.GetParticipantThumbnail(participantIDs...)
	if err != nil {
		return nil, fmt.Errorf("get participant thumbnails: %w", err)
	}
	result := make(map[string][]byte)
	for _, t := range resp.GetThumbnail() {
		if data := t.GetData(); data != nil {
			if buf := data.GetImageBuffer(); len(buf) > 0 {
				result[t.GetIdentifier()] = buf
			}
		}
	}
	return result, nil
}

func (rc *RealClient) GetOrCreateConversation(numbers []string) (string, error) {
	var contactNumbers []*gmproto.ContactNumber
	for _, n := range numbers {
		contactNumbers = append(contactNumbers, &gmproto.ContactNumber{
			MysteriousInt: 7, // 7 = user-input number
			Number:        n,
			Number2:       n,
		})
	}
	resp, err := rc.client.GetOrCreateConversation(&gmproto.GetOrCreateConversationRequest{
		Numbers: contactNumbers,
	})
	if err != nil {
		return "", fmt.Errorf("get or create conversation: %w", err)
	}
	conv := resp.GetConversation()
	if conv == nil {
		return "", fmt.Errorf("get or create conversation: nil conversation in response")
	}
	return conv.GetConversationID(), nil
}

func (rc *RealClient) SendReaction(messageID string, emoji string) error {
	_, err := rc.client.SendReaction(&gmproto.SendReactionRequest{
		MessageID: messageID,
		ReactionData: &gmproto.ReactionData{
			Unicode: emoji,
		},
		Action: gmproto.SendReactionRequest_ADD,
	})
	return err
}

func (rc *RealClient) SetCookies(cookies map[string]string) {
	rc.client.AuthData.SetCookies(cookies)
}

func (rc *RealClient) DoGaiaPairing(ctx context.Context, emojiCallback func(string)) error {
	return rc.client.DoGaiaPairing(ctx, emojiCallback)
}
