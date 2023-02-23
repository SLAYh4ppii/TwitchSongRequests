package api_test

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/saxypandabear/twitchsongrequests/pkg/api"
	"github.com/saxypandabear/twitchsongrequests/pkg/constants"
	"github.com/saxypandabear/twitchsongrequests/pkg/db"
	"github.com/saxypandabear/twitchsongrequests/pkg/users"
	"github.com/stretchr/testify/assert"
)

func TestRevokeUserAccess(t *testing.T) {
	u := db.InMemoryUserStore{
		Data: map[string]*users.User{
			"123": {
				TwitchID: "123",
			},
		},
	}

	re := ""

	h := api.NewUserHandler(&u, re)

	req := httptest.NewRequest("DELETE", "/", nil)
	req.AddCookie(&http.Cookie{
		Name:    constants.TwitchIDCookieKey,
		Value:   base64.StdEncoding.EncodeToString([]byte("123")),
		Expires: time.Now().AddDate(1, 0, 0),
	})

	rr := httptest.NewRecorder()
	api := http.HandlerFunc(h.RevokeUserAccesses)

	c := make(chan struct{})
	go func() {
		api.ServeHTTP(rr, req)
		c <- struct{}{}
	}()

	select {
	case <-c:
		t.Log("finished request")
	case <-time.After(25 * time.Millisecond):
		t.Error("did not receive message in time")
	}

	assert.Equal(t, http.StatusFound, rr.Code)
	assert.Empty(t, u.Data)
}
