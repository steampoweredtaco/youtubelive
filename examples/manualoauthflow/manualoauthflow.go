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

	// Manual Oauth2 flow is the behavior because yt.AutoAuthenticate option is unused
	ytLive, err := yt.NewYouTubeLive(clientID, clientSecret, yt.RefreshToken("badtoken"), yt.OnNewRefreshToken(func(refreshToken string) {
		fmt.Println("Refresh Token:", refreshToken)
	}) /* , yt.AutoAuthenticate() */)
	if err != nil {
		panic(err)
	}

	// Showing that an error is produced if a function is used without the login workflow first
	broadcastID, err := ytLive.CurrentBroadcastIDFromChannelHandle(channel)
	if errors.Is(err, yt.NotLoggedIn) {
		fmt.Println("YouTube not logged in:", err)
	} else if err != nil {
		panic(err)
	}

	refreshToken, err = yourOauthWorkflow()
	if err != nil {
		panic(err)
	}
	ytLive.SetRefreshToken(refreshToken)
	// alternatively you can use yt.Login() to use the builtin oauth2 workflow that will spawn
	// a browser to complete the 3 legged Oauth2 workflow.  This workflow would invoke the OnNewRefreshToken option used.
	//
	// err = ytLive.Login()
	broadcastID, err = ytLive.CurrentBroadcastIDFromChannelHandle(channel)
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

// yourOauthWorkflow is whatever your config management system needs to do to get a token for the user.
func yourOauthWorkflow() (string, error) {
	// for the example to work make sure refreshToken is correctly set in const or set in the .env file as refresh_token.
	return refreshToken, nil
}
