package youtubelive

import (
	"errors"
)

var (
	NotLiveError              = errors.New("user not live")
	InvalidYouTubeChannelName = errors.New("youtube channel name is invalid, cannot be blank")
	NoChannelsForUser         = errors.New("no channels for user")

	ErrBroadcastNotFound = errors.New("broadcast not found")
	ErrChatDisabled      = errors.New("live chat disabled")

	NotLoggedIn = errors.New("user not logged in")
)
