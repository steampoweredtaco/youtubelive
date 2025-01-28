package youtubelive

import (
	"fmt"
	"time"
)

type LiveEvent interface {
	ID() string
}

type BotEvent interface {
}

type AuthorDetails struct {
	ChannelId       string `json:"channelId,omitempty"`
	ChannelUrl      string `json:"channelUrl,omitempty"`
	DisplayName     string `json:"displayName,omitempty"`
	IsChatModerator bool   `json:"isChatModerator,omitempty"`
	IsChatOwner     bool   `json:"isChatOwner,omitempty"`
	IsChatSponsor   bool   `json:"isChatSponsor,omitempty"`
	IsVerified      bool   `json:"isVerified,omitempty"`
	ProfileImageUrl string `json:"profileImageUrl,omitempty"`
}

type ChatMessageEvent struct {
	Message       string
	DisplayName   string
	AuthorDetails AuthorDetails
	Timestamp     time.Time
	NextPageToken string
}

func (c ChatMessageEvent) ID() string {
	return fmt.Sprintf("chat-%s-%d", c.DisplayName, c.Timestamp.UnixNano())
}

type SuperChatEvent struct {
	Message       string
	Amount        float64
	Currency      string
	DisplayName   string
	AuthorDetails AuthorDetails
	Timestamp     time.Time
	NextPageToken string
}

func (s SuperChatEvent) ID() string {
	return fmt.Sprintf("superchat-%s-%d", s.DisplayName, s.Timestamp.UnixNano())
}

type SuperStickerEvent struct {
	StickerID     string
	Amount        float64
	Currency      string
	DisplayName   string
	AuthorDetails AuthorDetails
	Timestamp     time.Time
	NextPageToken string
}

func (s SuperStickerEvent) ID() string {
	return fmt.Sprintf("sticker-%s-%d", s.DisplayName, s.Timestamp.UnixNano())
}

type MemberMilestoneEvent struct {
	DisplayName   string
	AuthorDetails AuthorDetails
	Level         string // "new", "returning", "creator"
	Timestamp     time.Time
	NextPageToken string
	Months        int
}

func (s MemberMilestoneEvent) ID() string {
	return fmt.Sprintf("join-%s-%d", s.DisplayName, s.Timestamp.UnixNano())
}

type MembershipGiftEvent struct {
	DisplayName   string
	AuthorDetails AuthorDetails
	Total         int
	Tier          string
	Timestamp     time.Time
	NextPageToken string
}

func (s MembershipGiftEvent) ID() string {
	return fmt.Sprintf("gift-%s-%d", s.DisplayName, s.Timestamp.UnixNano())
}

type ChatEndedEvent struct {
	Timestamp     time.Time
	NextPageToken string
}

func (s ChatEndedEvent) ID() string {
	return fmt.Sprintf("end-%d", s.Timestamp.UnixNano())
}

type StreamEndEvent struct{}

func (s StreamEndEvent) ID() string {
	return "stream-end"
}

type BotChatMessage struct {
	Message string
}

type BotDeleteMessage struct {
	MessageID string
}

type UserBannedEvent struct {
	BannedUserID          string
	BanType               string // "permanent" or "temporary"
	Duration              time.Duration
	ModeratorID           string
	Timestamp             time.Time
	NextPageToken         string
	BannedUserDisplayName string
	ModeratorDisplayName  string
}

func (u UserBannedEvent) ID() string {
	return fmt.Sprintf("ban-%s-%s-%d", u.ModeratorID, u.BannedUserID, u.Timestamp.Unix())
}

type MembershipGiftReceivedEvent struct {
	DisplayText string
	Level       string
	GifterID    string
	Timestamp   time.Time
}

func (m MembershipGiftReceivedEvent) ID() string {
	return fmt.Sprintf("giftreceived-%s-%d", m.GifterID, m.Timestamp.Unix())
}

type ErrorEvent struct {
	Timestamp time.Time
	Error     error
}

func (e ErrorEvent) ID() string {
	return fmt.Sprintf("error-%d", e.Timestamp.Unix())
}
