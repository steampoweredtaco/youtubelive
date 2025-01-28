package youtubelive

import (
	"context"
	"errors"
	"fmt"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/youtube/v3"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"
)

type YouTubeLive struct {
	ctx       context.Context
	closeOnce sync.Once
	log       *slog.Logger

	transport http.RoundTripper
	jar       http.CookieJar

	resolved bool
	yclient  *ytClient

	clientID     string
	clientSecret string
	refreshToken string

	listenAddr       string
	additionalScopes []string
}

func NewYouTubeLive(clientID, clientSecret string, options ...Option) (*YouTubeLive, error) {
	yt := &YouTubeLive{}
	yt.ctx = context.Background()
	yt.log = slog.Default()
	yt.clientID = clientID
	yt.clientSecret = clientSecret
	yt.listenAddr = "127.0.0.1:0"

	var errs error
	for _, option := range options {
		err := option(yt)
		if err != nil {
			errs = errors.Join(errs, err)
			yt.log.Debug("error while processing option", "error", err)
		}
	}
	if errs != nil {
		yt.log.Error("error while processing options", "error", errs)
		return nil, errs
	}
	yt.yclient = &ytClient{
		ctx:          yt.ctx,
		transport:    yt.transport,
		jar:          yt.jar,
		log:          yt.log,
		clientID:     yt.clientID,
		clientSecret: yt.clientSecret,
		listenR:      listenResolve{listenAddr: yt.listenAddr},
		// ordering matters due to a workaround for a rare use case, perhaps add an
		// OverrideScopes() option in the future for this use case instead.
		scopes:       append(append(yt.additionalScopes[:0:0], yt.additionalScopes...), requiredScopes...),
		refreshToken: yt.refreshToken,
		service:      nil,
	}
	return yt, nil
}

// CurrentBroadcastIDFromChannelHandle returns the current live broadcastID which cn be used
// for Attach(). If multiple queries for a user's broadcast is required, to optimize quota usage it is recommended to get the channelID first from
// the channel handle and call CurrentBroadcastIDFromChannelID for this query. Will return NotLiveError when broadcast ID isn't found.
func (yt *YouTubeLive) CurrentBroadcastIDFromChannelHandle(channelName string) (string, error) {
	channelID, err := yt.ChannelIDFromChannelHandle(channelName)
	if err != nil {
		return "", err
	}
	return yt.CurrentBroadcastIDFromChannelID(channelID)
}

// CurrentBroadcastIDFromChannelID returns the current liver broadcastID which can be used for Attach(). Will return NotLiveError when a current live broadcast is not found.
func (yt *YouTubeLive) CurrentBroadcastIDFromChannelID(channelID string) (string, error) {
	err := yt.yclient.refresh()
	if err != nil {
		return "", err
	}
	channelsResp, err := yt.yclient.service.Channels.List([]string{"contentDetails"}).
		Id(channelID).
		Do()
	if err != nil || len(channelsResp.Items) == 0 {
		return "", fmt.Errorf("failed to get channel details: %v", err)
	}

	uploadsPlaylist := channelsResp.Items[0].ContentDetails.RelatedPlaylists.Uploads

	playlistResp, err := yt.yclient.service.PlaylistItems.List([]string{"contentDetails"}).
		PlaylistId(uploadsPlaylist).
		MaxResults(50). // Check last 5 videos
		Do()
	if err != nil {
		return "", fmt.Errorf("failed to get uploads: %v", err)
	}

	for _, item := range playlistResp.Items {
		videoID := item.ContentDetails.VideoId

		videoResp, err := yt.yclient.service.Videos.List([]string{"liveStreamingDetails"}).
			Id(videoID).
			Do()
		if err != nil || len(videoResp.Items) == 0 {
			continue
		}

		details := videoResp.Items[0].LiveStreamingDetails
		if details != nil && details.ActualStartTime != "" && details.ActualEndTime == "" {
			return videoID, nil
		}
	}

	// The above works most of the time and is cheaper than a search, and search sometimes doesn't work when the authenticated
	// user is not the owner and subscriber only chat.  Anyway, fallback to search now and if it still doesn't work there isn't
	// currently a known workaround through the API.
	qResp, err := yt.yclient.service.Search.List([]string{"snippet", "id"}).
		ChannelId(channelID).
		EventType("live").
		Q(channelID).
		Type("video").
		Do()

	if err != nil {
		return "", fmt.Errorf("failed to get search results: %v", err)
	}
	if len(qResp.Items) == 0 {
		return "", NotLiveError
	}
	return qResp.Items[0].Id.VideoId, nil
}

func (yt *YouTubeLive) CurrentBroadcastIDFromChannelIDB(channelID string) (string, error) {
	err := yt.yclient.refresh()
	if err != nil {
		return "", err
	}
	resp, err := yt.yclient.service.Search.List([]string{"id"}).ChannelId(channelID).EventType("live").Type("video").Do()
	if err != nil {
		return "", err
	}
	if len(resp.Items) == 0 {
		return "", NotLiveError
	}
	return resp.Items[0].Id.VideoId, nil
}

func (yt *YouTubeLive) ChannelIDFromChannelHandle(channelName string) (string, error) {
	if len(channelName) < 1 {
		return "", InvalidYouTubeChannelName
	}
	if channelName[0] != '@' {
		channelName = "@" + channelName
	}
	err := yt.yclient.refresh()
	if err != nil {
		return "", err
	}
	resp, err := yt.yclient.service.Channels.List([]string{"id"}).ForHandle(channelName).Do()
	if err != nil {
		return "", err
	}
	if len(resp.Items) == 0 {
		return "", fmt.Errorf("user does not exist")
	}
	return resp.Items[0].Id, nil
}

// IsLive will return true when channel is live.
func (yt *YouTubeLive) IsLive(channelID string) (bool, error) {
	if _, err := yt.CurrentBroadcastIDFromChannelID(channelID); err != nil {
		if errors.Is(err, NotLiveError) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// Attach to a live broadcast.  The returned out channel are all live events and the input channel are for chat events to send to broadcast.  A closed LiveEvent out channel indicates the live broadcast has ended or an error occurred which would require another Attach. By closing the in BotEvent channel, this will the close sending side of the attached connection but the ctx parameter must be canceled to trigger full cleanup of the attached routines.
func (yt *YouTubeLive) Attach(ctx context.Context, broadcastID string) (<-chan LiveEvent, chan<- BotEvent, error) {
	err := yt.yclient.refresh()
	if err != nil {
		return nil, nil, err
	}

	liveChatID, err := yt.getLiveChatID(broadcastID)
	if err != nil {
		return nil, nil, err
	}

	outChan := make(chan LiveEvent, 100)
	inChan := make(chan BotEvent, 100)

	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(ctx)

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(outChan)
		yt.pollLiveChat(ctx, liveChatID, outChan)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		yt.handleBotEvents(ctx, liveChatID, inChan, outChan)
	}()

	go func() {
		<-ctx.Done()
		cancel()
		wg.Wait()
	}()

	return outChan, inChan, nil
}

func (yt *YouTubeLive) getLiveChatID(broadcastID string) (string, error) {
	call := yt.yclient.service.Videos.List([]string{"liveStreamingDetails"}).
		Id(broadcastID).
		MaxResults(1)

	resp, err := call.Do()
	if err != nil {
		return "", fmt.Errorf("failed to get broadcast: %w", err)
	}

	if len(resp.Items) == 0 {
		return "", ErrBroadcastNotFound
	}

	liveChatID := resp.Items[0].LiveStreamingDetails.ActiveLiveChatId
	if liveChatID == "" {
		return "", ErrChatDisabled
	}

	return liveChatID, nil
}

func (yt *YouTubeLive) pollLiveChat(ctx context.Context, liveChatID string, out chan<- LiveEvent) {
	var (
		nextPageToken  string
		pollInterval   = 3 * time.Second // Initial default
		forceFirstPoll = true
	)

	for {
		timer := time.NewTimer(pollInterval)
		if forceFirstPoll {
			timer.Stop() // Immediate first poll
			forceFirstPoll = false
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(pollInterval):
			resp, err := yt.yclient.service.LiveChatMessages.List(liveChatID, []string{"snippet", "authorDetails"}).
				PageToken(nextPageToken).
				Do()
			if err != nil {
				yt.log.Debug("live chat poll failed", "error", err)
				gerr := &googleapi.Error{}
				if errors.As(err, &gerr) {
					// TODO make the return behavior optional
					select {
					case out <- &ErrorEvent{
						Timestamp: time.Now().UTC(),
						Error:     gerr,
					}:
					case <-ctx.Done():
					}

					select {
					case out <- &ChatEndedEvent{
						Timestamp: time.Now().UTC(),
					}:
					case <-ctx.Done():
					}
					return
				}
				// TODO make the return behavior optional
				select {
				case out <- &ErrorEvent{
					Timestamp: time.Now().UTC(),
					Error:     err,
				}:
				case <-ctx.Done():
				}
				continue
			}

			nextPageToken = resp.NextPageToken
			if resp.PollingIntervalMillis > 0 {
				pollInterval = time.Duration(resp.PollingIntervalMillis) * time.Millisecond
			}
			for _, msg := range resp.Items {
				event, err := yt.parseChatMessage(msg, resp.NextPageToken)
				if err != nil {
					yt.log.Warn("failed to parse chat message", "error", err)
					continue
				}
				select {
				case out <- event:
				case <-ctx.Done():
				}
				if _, ok := event.(*ChatEndedEvent); ok {
					return
				}
			}
		}
	}
}

func (yt *YouTubeLive) parseChatMessage(msg *youtube.LiveChatMessage, nextPageToken string) (LiveEvent, error) {
	snippet := msg.Snippet
	ts, err := time.Parse(time.RFC3339, snippet.PublishedAt)
	if err != nil {
		return nil, fmt.Errorf("invalid timestamp: %w", err)
	}
	baseEvent := struct {
		NextPageToken string
		Timestamp     time.Time
		DisplayName   string
		AuthorDetails *youtube.LiveChatMessageAuthorDetails
	}{
		NextPageToken: nextPageToken,
		Timestamp:     ts,
		DisplayName:   msg.AuthorDetails.DisplayName,
		AuthorDetails: msg.AuthorDetails,
	}

	switch snippet.Type {
	case "textMessageEvent":
		return &ChatMessageEvent{
			Message:       snippet.TextMessageDetails.MessageText,
			DisplayName:   baseEvent.DisplayName,
			AuthorDetails: toAuthorDetails(baseEvent.AuthorDetails),
			Timestamp:     baseEvent.Timestamp,
			NextPageToken: baseEvent.NextPageToken,
		}, nil
	case "superChatEvent":
		return &SuperChatEvent{
			Message:       snippet.SuperChatDetails.UserComment,
			Amount:        float64(snippet.SuperChatDetails.AmountMicros) / 1000000,
			Currency:      snippet.SuperChatDetails.Currency,
			DisplayName:   baseEvent.DisplayName,
			AuthorDetails: toAuthorDetails(baseEvent.AuthorDetails),
			Timestamp:     baseEvent.Timestamp,
			NextPageToken: baseEvent.NextPageToken,
		}, nil
	case "superStickerEvent":
		return &SuperStickerEvent{
			StickerID:     snippet.SuperStickerDetails.SuperStickerMetadata.StickerId,
			Amount:        float64(snippet.SuperStickerDetails.AmountMicros) / 1000000,
			Currency:      snippet.SuperStickerDetails.Currency,
			DisplayName:   baseEvent.DisplayName,
			AuthorDetails: toAuthorDetails(baseEvent.AuthorDetails),
			Timestamp:     baseEvent.Timestamp,
			NextPageToken: baseEvent.NextPageToken,
		}, nil
	case "memberMilestoneChatEvent":
		return &MemberMilestoneEvent{
			DisplayName:   baseEvent.DisplayName,
			AuthorDetails: toAuthorDetails(baseEvent.AuthorDetails),
			Level:         strings.ToLower(snippet.MemberMilestoneChatDetails.MemberLevelName),
			Months:        int(snippet.MemberMilestoneChatDetails.MemberMonth),
			Timestamp:     baseEvent.Timestamp,
			NextPageToken: baseEvent.NextPageToken,
		}, nil
	case "membershipGiftingEvent":
		return &MembershipGiftEvent{
			DisplayName:   baseEvent.DisplayName,
			AuthorDetails: toAuthorDetails(baseEvent.AuthorDetails),
			Total:         int(snippet.MembershipGiftingDetails.GiftMembershipsCount),
			Tier:          snippet.MembershipGiftingDetails.GiftMembershipsLevelName,
			Timestamp:     baseEvent.Timestamp,
			NextPageToken: baseEvent.NextPageToken,
		}, nil
	case "giftMembershipReceivedEvent":
		return &MembershipGiftReceivedEvent{
			DisplayText: msg.Snippet.DisplayMessage,
			Level:       msg.Snippet.GiftMembershipReceivedDetails.MemberLevelName,
			GifterID:    msg.Snippet.GiftMembershipReceivedDetails.GifterChannelId,
			Timestamp:   baseEvent.Timestamp,
		}, nil
	case "userBannedEvent":
		details := snippet.UserBannedDetails
		if details == nil {
			return nil, errors.New("moderation event without details")
		}

		banType := "unknown"
		var duration time.Duration

		switch {
		case details.BanType == "PERMANENT":
			banType = "permanent"
		case details.BanType == "TEMPORARY":
			banType = "temporary"
			duration = time.Duration(details.BanDurationSeconds) * time.Second
		}

		return &UserBannedEvent{
			BannedUserID:          details.BannedUserDetails.ChannelId,
			BannedUserDisplayName: details.BannedUserDetails.DisplayName,
			BanType:               banType,
			Duration:              duration,
			ModeratorID:           snippet.AuthorChannelId,
			ModeratorDisplayName:  msg.AuthorDetails.DisplayName,
			Timestamp:             baseEvent.Timestamp,
			NextPageToken:         baseEvent.NextPageToken,
		}, nil
	case "chatEndedEvent":
		return &ChatEndedEvent{
			Timestamp:     baseEvent.Timestamp,
			NextPageToken: baseEvent.NextPageToken,
		}, nil
	default:
		yt.log.Warn("unsupported message type",
			"type", snippet.Type,
			"message_id", msg.Id,
			"display", snippet.DisplayMessage,
		)
		return nil, fmt.Errorf("unsupported message type: %s", snippet.Type)
	}
}

func (yt *YouTubeLive) handleBotEvents(ctx context.Context, liveChatID string, in <-chan BotEvent, out chan<- LiveEvent) {
	for {
		select {
		case <-ctx.Done():
			return
		case evt, ok := <-in:
			if !ok {
				return
			}
			switch e := evt.(type) {
			case BotChatMessage:
				err := yt.sendChatMessage(liveChatID, e.Message)
				if err != nil {
					select {
					case out <- &ErrorEvent{
						Timestamp: time.Now().UTC(),
						Error:     err,
					}:
					case <-ctx.Done():
					}
				}
			case BotDeleteMessage:
				err := yt.deleteChatMessage(e.MessageID)
				if err != nil {
					select {
					case out <- &ErrorEvent{
						Timestamp: time.Now().UTC(),
						Error:     err,
					}:
					case <-ctx.Done():
					}
				}
			default:
				yt.log.Debug("received unknown bot event type", "type", fmt.Sprintf("%T", evt))
			}
		}
	}
}

func (yt *YouTubeLive) sendChatMessage(liveChatID, message string) error {
	msg := &youtube.LiveChatMessage{
		Snippet: &youtube.LiveChatMessageSnippet{
			LiveChatId: liveChatID,
			Type:       "textMessageEvent",
			TextMessageDetails: &youtube.LiveChatTextMessageDetails{
				MessageText: message,
			},
		},
	}

	_, err := yt.yclient.service.LiveChatMessages.Insert([]string{"snippet"}, msg).Do()
	return err
}

func (yt *YouTubeLive) deleteChatMessage(messageID string) error {
	return yt.yclient.service.LiveChatMessages.Delete(messageID).Do()
}

func toAuthorDetails(authorDetails *youtube.LiveChatMessageAuthorDetails) AuthorDetails {
	return AuthorDetails{
		ChannelId:       authorDetails.ChannelId,
		ChannelUrl:      authorDetails.ChannelUrl,
		DisplayName:     authorDetails.DisplayName,
		IsChatModerator: authorDetails.IsChatModerator,
		IsChatOwner:     authorDetails.IsChatOwner,
		IsChatSponsor:   authorDetails.IsChatSponsor,
		IsVerified:      authorDetails.IsVerified,
		ProfileImageUrl: authorDetails.ProfileImageUrl,
	}
}
