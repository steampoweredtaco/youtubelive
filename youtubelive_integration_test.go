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
