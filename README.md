
# Go YouTube Live
### A simple youtube live client chat

[![Go Report Card](https://goreportcard.com/badge/github.com/steampoweredtaco/youtubelive)](https://goreportcard.com/report/github.com/steampoweredtaco/youtubelive)
[![Go Reference](https://pkg.go.dev/badge/github.com/steampoweredtaco/youtubelive.svg)](https://pkg.go.dev/github.com/steampoweredtaco/youtubelive)
[![Discord](https://img.shields.io/discord/977648006063091742?label=Discord)](https://discord.gg/HwWDNGBmxY)

`go get github.com/steampoweredtaco/youtubelive`

<p style="text-align: center">
    <a href="#key-features">Key Features</a> •
    <a href="#in-progress-features">In Progress</a> •
    <a href="#get-the-module">Get</a> •
    <a href="#examples">Examples</a> •
    <a href="#contact">Contact</a> •
    <a href="#license">License</a> 
</p>

## Key Features
* Simplified OAuth2 login.
* Monitor when a youtube channel becomes live.
* Channel based interface for getting live chat messages and events and sending commands to the live.

## In progress Features
* Override OAuth2 from browser based workflow to with custom Token provider workflow.

## Usage
### Get the module
`go get github.com/steampoweredtaco/youtubelive`

## Examples
* [Monitor Live Status](examples%2FcheckLive%2Fchecklive.go)
* [Stream chat from a live stream and post a message to chat](examples%2FcheckLive%2Fchecklive.go)
### Basic Example
```go
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
```
## Contact
Currently available on [Discord](https://discord.gg/HwWDNGBmxY)

## License

MIT

