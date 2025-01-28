package main

import (
	"context"
	"errors"
	"fmt"
	yt "github.com/steampoweredtaco/youtubelive"
	"strings"
	"time"
)

var (
	channel          = "@christinafixates"
	clientID         = ""
	clientSecret     = ""
	refreshToken     = ""
	additionalScopes = []string{}
)

const (
	// Can set this to false if you want to hardcode the values above (not recommended).  See LoadOauthCredentialsFromDotFile.
	loadFromDotEnv = true
)

func main() {
	if loadFromDotEnv {
		var err error
		clientID, clientSecret, refreshToken, additionalScopes, err = yt.LoadOauthCredentialsFromDotFile()
		if err != nil {
			panic(err)
		}
	}
	ytLive, err := yt.NewYouTubeLive(clientID, clientSecret, yt.RefreshToken(refreshToken))
	if err != nil {
		panic(err)
	}
	broadcastID, err := ytLive.CurrentBroadcastIDFromChannelHandle(channel)
	if err != nil {
		panic(err)
	}

	ctx, cancel := context.WithTimeoutCause(context.Background(), 1*time.Hour, errors.New("demo over"))
	defer cancel()
	events, commands, err := ytLive.Attach(ctx, broadcastID)
	if err != nil {
		panic(err)
	}
	commands <- yt.BotChatMessage{Message: "Looking good."}

	for event := range events {
		switch e := event.(type) {
		case *yt.ChatMessageEvent:
			fmt.Printf("[%s] %s: %s\n",
				e.Timestamp.Local().Format(time.Stamp),
				e.DisplayName,
				e.Message)

		case *yt.SuperChatEvent:
			fmt.Printf("[%s] 💎 Super Chat from %s: %s (%.2f %s)\n",
				e.Timestamp.Local().Format(time.Stamp),
				e.DisplayName,
				e.Message,
				e.Amount,
				e.Currency)

		case *yt.SuperStickerEvent:
			fmt.Printf("[%s] 🎟️ Super Sticker from %s: ID %s (%.2f %s)\n",
				e.Timestamp.Local().Format(time.Stamp),
				e.DisplayName,
				e.StickerID,
				e.Amount,
				e.Currency)

		case *yt.MemberMilestoneEvent:
			if e.Months == 0 {
				fmt.Printf("[%s] 🎉 New %s member: %s\n",
					e.Timestamp.Local().Format(time.Stamp),
					strings.Title(e.Level),
					e.DisplayName)
			} else {
				fmt.Printf("[%s] 🎉 Welcomeback %s member: %s\n",
					e.Timestamp.Local().Format(time.Stamp),
					strings.Title(e.Level),
					e.DisplayName)
			}

		case *yt.MembershipGiftEvent:
			msg := fmt.Sprintf("[%s] 🎁 %s gifted %d %s memberships",
				e.Timestamp.Local().Format(time.Stamp),
				e.DisplayName,
				e.Total,
				e.Tier)

			fmt.Println(msg)

		case *yt.MembershipGiftReceivedEvent:
			msg := fmt.Sprintf("[%s] 🎁 %s",
				e.Timestamp.Local().Format(time.Stamp),
				e.DisplayText)

			fmt.Println(msg)

		case *yt.UserBannedEvent:
			banMsg := fmt.Sprintf("[%s] 🔨 Moderator %s banned %s (%s)",
				e.Timestamp.Local().Format(time.Stamp),
				e.ModeratorDisplayName,
				e.BannedUserDisplayName,
				e.BanType)

			if e.BanType == "temporary" {
				banMsg += fmt.Sprintf(" for %v", e.Duration)
			}
			fmt.Println(banMsg)

		case *yt.ChatEndedEvent:
			fmt.Printf("[%s] ⏹️ Live chat has ended\n",
				e.Timestamp.Local().Format(time.Stamp))
			// After chat ended no more events should be considered.
			return

		case *yt.ErrorEvent:
			fmt.Printf("[%s] ⚠️ %s\n", e.Timestamp.Local().Format(time.Stamp), e.Error)

		default:
			fmt.Printf("⚠️ Unhandled event type: %T\n", e)
		}
	}

}
