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
