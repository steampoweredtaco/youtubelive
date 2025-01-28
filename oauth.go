package youtubelive

import (
	"context"
	"errors"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/authhandler"
)

func RefreshTokenSourceWithPKCE(ctx context.Context, config *oauth2.Config, token *oauth2.Token, state string, authHandler authhandler.AuthorizationHandler, pkce *authhandler.PKCEParams, opts ...oauth2.AuthCodeOption) oauth2.TokenSource {
	ts := config.TokenSource(ctx, token)
	return oauth2.ReuseTokenSource(nil, authHandlerSource{config: config, ctx: ctx, authHandler: authHandler, state: state, pkce: pkce, opts: opts, tokenSource: ts})
}

const (
	// Parameter keys for AuthCodeURL method to support PKCE.
	codeChallengeKey       = "code_challenge"
	codeChallengeMethodKey = "code_challenge_method"

	// Parameter key for Exchange method to support PKCE.
	codeVerifierKey = "code_verifier"
)

//
// func TokenSource(ctx context.Context, config *oauth2.Config, state string, authHandler authhandler.AuthorizationHandler) oauth2.TokenSource {
// 	return RefreshTokenSourceWithPKCE(ctx, config, state, authHandler, nil)
// }

type authHandlerSource struct {
	ctx         context.Context
	config      *oauth2.Config
	authHandler authhandler.AuthorizationHandler
	state       string
	pkce        *authhandler.PKCEParams
	opts        []oauth2.AuthCodeOption
	tokenSource oauth2.TokenSource
}

func (source authHandlerSource) Token() (*oauth2.Token, error) {
	t, err := source.tokenSource.Token()
	if err == nil {
		return t, nil
	}

	var authCodeUrlOptions []oauth2.AuthCodeOption
	if source.pkce != nil && source.pkce.Challenge != "" && source.pkce.ChallengeMethod != "" {
		authCodeUrlOptions = []oauth2.AuthCodeOption{oauth2.SetAuthURLParam(codeChallengeKey, source.pkce.Challenge),
			oauth2.SetAuthURLParam(codeChallengeMethodKey, source.pkce.ChallengeMethod)}
		authCodeUrlOptions = append(authCodeUrlOptions, source.opts...)
	}
	url := source.config.AuthCodeURL(source.state, authCodeUrlOptions...)
	code, state, err := source.authHandler(url)
	if err != nil {
		return nil, err
	}
	if state != source.state {
		return nil, errors.New("state mismatch in 3-legged-OAuth flow")
	}

	var exchangeOptions []oauth2.AuthCodeOption
	if source.pkce != nil && source.pkce.Verifier != "" {
		exchangeOptions = []oauth2.AuthCodeOption{oauth2.SetAuthURLParam(codeVerifierKey, source.pkce.Verifier)}
	}
	t, err = source.config.Exchange(source.ctx, code, exchangeOptions...)
	if err != nil {
		return nil, err
	}
	source.tokenSource = source.config.TokenSource(source.ctx, t)
	return t, nil
}
