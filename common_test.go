package youtubelive

import (
	"context"
	"log/slog"
	"time"
)

const userHandle = "christinafixates"

func loadYtClient() (*ytClient, func(), error) {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(3*time.Minute))
	clientId, clientSecret, refreshToken, scopes, err := LoadOauthCredentialsFromDotFile()
	if err != nil {
		cancel()
		return nil, nil, err
	}

	y := newYTClient(ctx,
		slog.Default(),
		nil,
		clientId,
		clientSecret,
		scopes,
		refreshToken,
		"127.0.0.1:8919")
	err = y.refresh()
	if err != nil {
		cancel()
		return nil, nil, err
	}
	return y, cancel, nil
}
