package main

import (
	"context"
	"errors"
	"fmt"
	yt "github.com/steampoweredtaco/youtubelive"
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
	// Send true to send initial message on connection
	sendMessage = false
	message     = "Looking good."
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
	if sendMessage {
		commands <- yt.BotChatMessage{Message: message}
	}

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

		case *yt.ChatEndedEvent:
			fmt.Printf("[%s] ⏹️ Live chat has ended\n",
				e.Timestamp.Local().Format(time.Stamp))
			return

		case *yt.ErrorEvent:
			fmt.Printf("[%s] ⚠️ %s\n", e.Timestamp.Local().Format(time.Stamp), e.Error)

		default:
			fmt.Printf("⚠️ Unhandled event type: %T\n", e)
		}
	}
}
