//go:build integration

package youtubelive

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestYouTubeLive_IsLive(t *testing.T) {

	clientId, clientSecret, token, additionalScopes, err := LoadOauthCredentialsFromDotFile()
	assert.Nil(t, err)
	yt, err := NewYouTubeLive(clientId, clientSecret, RefreshToken(token), AdditionalScopes(additionalScopes...))
	if !assert.NoError(t, err) {
		return
	}

	channelID, err := yt.ChannelIDFromChannelHandle(userHandle)
	assert.NoError(t, err)
	live, err := yt.IsLive(channelID)
	assert.NoError(t, err)
	assert.True(t, live)

}

func TestYouTubeLive_LoggedInChannel(t *testing.T) {

	clientId, clientSecret, token, additionalScopes, err := LoadOauthCredentialsFromDotFile()
	assert.Nil(t, err)
	yt, err := NewYouTubeLive(clientId, clientSecret, RefreshToken(token), AutoAuthenticate(), AdditionalScopes(additionalScopes...))
	if !assert.NoError(t, err) {
		return
	}

	channelTitle, channelID, err := yt.LoggedInChannel()
	assert.NoError(t, err)
	t.Log(channelTitle, channelID)
}
