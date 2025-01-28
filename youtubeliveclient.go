package youtubelive

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/authhandler"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

var (
	requiredScopes = []string{
		"https://www.googleapis.com/auth/youtube",
	}
)

type ytClient struct {
	ctx       context.Context
	transport http.RoundTripper
	log       *slog.Logger

	clientID     string
	clientSecret string
	redirectURI  string
	scopes       []string

	refreshToken string

	listenR     listenResolve
	service     *youtube.Service
	tokenSource oauth2.TokenSource
	jar         http.CookieJar
}

func newYTClient(ctx context.Context, log *slog.Logger, transport http.RoundTripper, clientID, clientSecret string, scopes []string, refreshToken string, listenAddr string) *ytClient {
	allScopes := make([]string, 0, len(requiredScopes)+len(scopes))
	// Passed in scopes go in first for a very specific work around not important to 99.99% of use cases.
	allScopes = append(allScopes, scopes...)
	allScopes = append(allScopes, requiredScopes...)
	c := &ytClient{
		ctx:          ctx,
		log:          log,
		transport:    transport,
		clientID:     clientID,
		clientSecret: clientSecret,
		scopes:       allScopes,
		refreshToken: refreshToken,
		listenR: listenResolve{
			listenAddr: listenAddr,
		},
	}

	return c
}
func (yt *ytClient) refresh() error {
	if yt.service != nil {
		return nil
	}
	err := yt.listenR.setupListener()
	if err != nil {
		return err
	}
	yt.redirectURI = "http://" + yt.listenR.effectiveAddr + "/callback"
	err = yt.validate()
	if err != nil {
		return err
	}

	yt.ctx = context.WithValue(yt.ctx, oauth2.HTTPClient, &http.Client{Transport: yt.transport, Jar: yt.jar})

	handler, challenge, verifier, err := yt.createAuthPKCEAuth(yt.listenR)
	if err != nil {
		return err
	}
	conf := &oauth2.Config{
		ClientID:     yt.clientID,
		ClientSecret: yt.clientSecret,
		Endpoint:     google.Endpoint,
		RedirectURL:  yt.redirectURI,
		Scopes:       yt.scopes,
	}
	var token *oauth2.Token

	token = &oauth2.Token{
		TokenType:    "bearer",
		RefreshToken: yt.refreshToken,
	}

	tokenSource := RefreshTokenSourceWithPKCE(yt.ctx,
		conf,
		token,
		"",
		handler,
		&authhandler.PKCEParams{
			Challenge:       challenge,
			ChallengeMethod: "S256",
			Verifier:        verifier,
		},
		oauth2.SetAuthURLParam("service", "lso"),
		oauth2.SetAuthURLParam("o2v", "2"),
		oauth2.SetAuthURLParam("ddm", "1"),
		oauth2.SetAuthURLParam("flowName", "GeneralOAuthFlow"),
		oauth2.SetAuthURLParam("force_verify", "true"),
		oauth2.AccessTypeOffline,
		oauth2.ApprovalForce,
	)

	yt.tokenSource = tokenSource
	c := oauth2.NewClient(yt.ctx, yt.tokenSource)
	yt.service, err = youtube.NewService(yt.ctx, option.WithHTTPClient(c))
	if err != nil {
		return err
	}

	return nil
}

func (yt *ytClient) createAuthPKCEAuth(endpoint listenResolve) (authhandler.AuthorizationHandler, string, string, error) {
	verifier, challenge, err := generatePKCE()
	if err != nil {
		return nil, "", "", err
	}

	return func(authCodeURL string) (code string, state string, err error) {
		var authErr error
		ctx, cancel := context.WithTimeout(yt.ctx, time.Minute*5)
		defer cancel()

		mux := http.NewServeMux()
		server := &http.Server{
			Handler: mux,
		}

		mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
			defer cancel()
			w.WriteHeader(http.StatusOK)
			code = r.FormValue("code")
			state = r.FormValue("state")
			if code == "" {
				authErr = fmt.Errorf("no code received")
				http.Error(w, "no code found in the callback", http.StatusBadRequest)
				return
			}
			yt.log.Debug("got auth callback", "code", code, "state", state)
			fmt.Fprintln(w, `<html>
<head><title>Authorization processed</title></head>
<body>
<h2>Authorization processed</h2>
<p>You can now close this window and return to the application.</p>
</body></html>`)
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}

		})

		var wg sync.WaitGroup
		wg.Add(1)

		go func() {
			defer wg.Done()
			yt.log.Info("starting local server", "listen", fmt.Sprintf("http://%s", endpoint))
			if err != nil {
				authErr = err
				return
			}

			if authErr = server.Serve(endpoint.stickyPort); authErr != nil {
				if errors.Is(authErr, http.ErrServerClosed) {
					authErr = nil
					return
				}
				yt.log.Error("unexpected error running http server", "error", authErr)
			}
		}()

		yt.log.Info("opening browser for authorization...")
		err = openBrowser(authCodeURL)
		if err != nil {
			yt.log.Error("error opening browser for authorization", "error", err)
			yt.log.Info(fmt.Sprintf("visit the following url manually: %s", authCodeURL))
		}
		<-ctx.Done()
		ctx2, cancel2 := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel2()
		server.Shutdown(ctx2)
		cancel()
		wg.Wait()
		return code, state, authErr
	}, challenge, verifier, nil
}

func generatePKCE() (verifier, challenge string, err error) {
	b := make([]byte, 32)
	_, err = rand.Read(b)
	if err != nil {
		return "", "", fmt.Errorf("Failed to generate PKCE code verifier: %v", err)
	}
	verifier = base64.RawURLEncoding.EncodeToString(b)
	hash := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(hash[:])
	return verifier, challenge, nil
}

func (yt *ytClient) validate() error {
	var errs error
	if yt.clientID == "" {
		errs = errors.Join(fmt.Errorf("YouTube Client ID is empty"))
	}
	if yt.redirectURI == "" {
		errs = errors.Join(fmt.Errorf("YouTube Redirect URI is empty"))
	}
	if len(yt.scopes) == 0 {
		errs = errors.Join(fmt.Errorf("YouTube Scopes is empty"))
	}
	return errs
}

func (yt *ytClient) Token() (*oauth2.Token, error) {
	err := yt.refresh()
	if err != nil {
		return nil, err
	}
	return yt.tokenSource.Token()

}
