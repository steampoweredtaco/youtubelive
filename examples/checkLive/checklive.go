package main

import (
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
	channelID, err := ytLive.ChannelIDFromChannelHandle(channel)
	if err != nil {
		panic(err)
	}
	for {
		isLive, err := ytLive.IsLive(channelID)
		if err != nil {
			panic(err)
		}
		fmt.Printf("%s is live? %v\n", channel, isLive)
		time.Sleep(5 * time.Second)
	}

}
