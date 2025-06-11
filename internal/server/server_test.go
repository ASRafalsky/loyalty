package server

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ASRafalsky/telemetry/pkg/log"
	"github.com/gojek/heimdall/v7/httpclient"
	"github.com/mailru/easyjson"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/ASRafalsky/internal/db/mockdb"
	"github.com/ASRafalsky/internal/models"
	"github.com/ASRafalsky/internal/server/middleware"
)

func TestRegisterPostHandler(t *testing.T) {
	Log, err := log.AddLoggerWith("info", "")
	require.NoError(t, err)
	repo := mockdb.New()
	srv := httptest.NewServer(middleware.WithLogging(newRouter(repo, Log), Log))
	defer srv.Close()

	// Create a new HTTP client with a default timeout
	timeout := 1000 * time.Millisecond
	client := httpclient.NewClient(httpclient.WithHTTPTimeout(timeout))

	header := http.Header{
		"Content-Type": []string{"application/json"},
	}

	user1 := "ololoshka"
	user2 := "trololoshka"
	tt := []struct {
		name          string
		url           string
		data          models.Credential
		header        http.Header
		expStatusCode int
		expHeaders    http.Header
	}{
		{
			name: "success1",
			url:  srv.URL + "/api/user/register",
			data: models.Credential{
				Login:    user1,
				Password: "303011111",
			},
			header: http.Header{
				"Content-Type": []string{"application/json"},
			},
			expStatusCode: http.StatusOK,
		},
		{
			name: "success2",
			url:  srv.URL + "/api/user/register",
			data: models.Credential{
				Login:    user2,
				Password: "303011111222",
			},
			header: http.Header{
				"Content-Type":    []string{"application/json"},
				"Accept-Encoding": []string{"gzip"},
			},
			expStatusCode: http.StatusOK,
		},
		{
			name: "conflict",
			url:  srv.URL + "/api/user/register",
			data: models.Credential{
				Login:    user2,
				Password: "55555555555",
			},
			header: http.Header{
				"Content-Type":    []string{"application/json"},
				"Accept-Encoding": []string{"gzip"},
			},
			expStatusCode: http.StatusConflict,
		},
	}

	authHeaderUser1 := http.Header{}
	authHeaderUser2 := http.Header{}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			buf, err := easyjson.Marshal(tc.data)
			require.NoError(t, err)
			resp, err := client.Post(tc.url, bytes.NewReader(buf), header)
			require.Equal(t, tc.expStatusCode, resp.StatusCode)
			if tc.expStatusCode == http.StatusOK {
				require.True(t, len(resp.Header.Get("Authorization")) > 0 ||
					len(resp.Header.Get("Set-Cookie")) > 0)
				authHeader := http.Header{}
				if len(resp.Header.Get("Authorization")) > 0 {
					authHeader.Set("Authorization", resp.Header.Get("Authorization"))
				} else {
					authHeader.Set("Authorization", resp.Header.Get("Set-Cookie"))
				}

				switch tc.data.Login {
				case user1:
					authHeaderUser1 = authHeader
				case user2:
					authHeaderUser2 = authHeader
				default:
				}

				loginID, err := models.IDHash(tc.data.Login)
				require.NoError(t, err)
				passwdHash, err := repo.GetCred(context.Background(), loginID)
				require.NoError(t, err)
				require.NoError(t, bcrypt.CompareHashAndPassword(passwdHash, []byte(tc.data.Password)))
			}
			require.NoError(t, resp.Body.Close())
		})
	}

	order := "49927398716"
	t.Run("set_order", func(t *testing.T) {
		authHeaderUser1.Set("Content-Type", "text/plain")
		resp, err := client.Post(srv.URL+"/api/user/orders", bytes.NewReader([]byte(order)), authHeaderUser1)
		require.NoError(t, err)
		require.Equal(t, http.StatusAccepted, resp.StatusCode)
	})
	t.Run("set_the_same_order_for_the_same_user", func(t *testing.T) {
		authHeaderUser1.Set("Content-Type", "text/plain")
		resp, err := client.Post(srv.URL+"/api/user/orders", bytes.NewReader([]byte(order)), authHeaderUser1)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
	})
	t.Run("set_the_same_order_for_the_another_user", func(t *testing.T) {
		authHeaderUser2.Set("Content-Type", "text/plain")
		resp, err := client.Post(srv.URL+"/api/user/orders", bytes.NewReader([]byte(order)), authHeaderUser2)
		require.NoError(t, err)
		require.Equal(t, http.StatusConflict, resp.StatusCode)
	})

	t.Run("get_order_for_the_user", func(t *testing.T) {
		resp, err := client.Get(srv.URL+"/api/user/orders", authHeaderUser1)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
	})
}
