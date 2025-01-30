package youtubelive

import (
	"net/http"
)

type Option func(*YouTubeLive) error

// HttpTransport specify the transport to use for the http client. If you need to
// specialize the transport or protocol usage this is how you do it. Do not mix any other
// transport specific options with this option, the behavior may be undefined
func HttpTransport(transport http.RoundTripper) Option {
	return func(yt *YouTubeLive) error {
		yt.transport = transport
		return nil
	}
}

// CookieJar specify initial cookies if needed.
func CookieJar(jar http.CookieJar) Option {
	return func(yt *YouTubeLive) error {
		yt.jar = jar
		return nil
	}
}

func RefreshToken(refreshToken string) Option {
	return func(yt *YouTubeLive) error {
		yt.refreshToken = refreshToken
		return nil
	}
}

func OathListenAddr(oathListenAddr string) Option {
	return func(yt *YouTubeLive) error {
		yt.listenAddr = oathListenAddr
		return nil
	}
}

func AdditionalScopes(scopes ...string) Option {
	return func(yt *YouTubeLive) error {
		yt.additionalScopes = scopes
		return nil
	}
}

// AutoAuthenticate will automatically call the OAuth2 workflow when the refresh token is no longer valid and there is no valid token.
// Otherwise, NotLoggedIn error will be returned by methods that requires a valid token. Using RefreshToken option, or the Login, ForceLogin, or SetRefreshToken method must be called prior to resolve the NotLoggedIn error when AutoAuthenticate is not used.
func AutoAuthenticate() Option {
	return func(yt *YouTubeLive) error {
		yt.autoAuth = true
		return nil
	}
}
