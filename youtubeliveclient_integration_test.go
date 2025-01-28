//go:build integration

package youtubelive

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"net"
	"net/http"
	"testing"
)

func TestRefresh(t *testing.T) {
	y, cancel, err := loadYtClient()
	if !assert.NoError(t, err) {
		return
	}
	defer cancel()

	token, err := y.Token()
	assert.NoError(t, err)
	fmt.Println("refresh token: ", token.RefreshToken, " new: ", token.RefreshToken, " access: ", token.AccessToken)
}

func TestIsolation(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:8919") // Explicit IP
	if err != nil {
		t.Errorf("Failed to listen: %v", err)

	}
	t.Logf("Listening on %s", listener.Addr().String())

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		t.Log("Callback received!")
		w.Write([]byte("OK"))
	})
}
