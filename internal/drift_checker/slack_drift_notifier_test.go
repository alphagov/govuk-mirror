package drift_checker_test

import (
	"encoding/json"
	"io"
	"mirrorer/internal/drift_checker"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

type Teardown = func()

func setupTest(handler http.HandlerFunc) (*httptest.Server, Teardown) {
	var h http.HandlerFunc
	defaultHandler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}
	if handler != nil {
		h = handler
	} else {
		h = defaultHandler
	}

	srv := httptest.NewServer(h)
	return srv, srv.Close
}

func TestSlackDriftNotifier_Notify(t *testing.T) {
	t.Run("makes a POST request to the webhook URL", func(t *testing.T) {
		postRequestSeen := false
		srv, teardown := setupTest(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost && r.URL.Path == "/webhook" {
				postRequestSeen = true
			}
		})
		defer teardown()

		u, err := url.Parse(srv.URL + "/webhook")
		assert.NoError(t, err)
		notifier := drift_checker.NewSlackDriftNotifier(*u)

		err = notifier.Notify(drift_checker.DriftSummary{})
		assert.NoError(t, err)
		assert.True(t, postRequestSeen)
	})

	t.Run("sets the content type to application/json", func(t *testing.T) {
		isApplicationJson := false
		srv, teardown := setupTest(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Content-Type") == "application/json" {
				isApplicationJson = true
			}
		})
		defer teardown()

		u, err := url.Parse(srv.URL + "/webhook")
		assert.NoError(t, err)
		notifier := drift_checker.NewSlackDriftNotifier(*u)

		err = notifier.Notify(drift_checker.DriftSummary{})
		assert.NoError(t, err)
		assert.True(t, isApplicationJson)
	})

	t.Run("the POST body is JSON encoded with the correct fields", func(t *testing.T) {
		var body []byte
		srv, teardown := setupTest(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				_ = r.Body.Close()
			}()

			bodyBytes, err := io.ReadAll(r.Body)
			assert.NoError(t, err)
			body = bodyBytes
		})
		defer teardown()

		u, err := url.Parse(srv.URL + "/webhook")
		assert.NoError(t, err)
		notifier := drift_checker.NewSlackDriftNotifier(*u)

		err = notifier.Notify(drift_checker.DriftSummary{})
		assert.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(body, &result)
		assert.NoError(t, err)

		assert.Contains(t, result, "text")
	})

	t.Run("returns an error if sending the request fails", func(t *testing.T) {
		srv, teardown := setupTest(nil)
		teardown() // stop it before the request is sent
		u, err := url.Parse(srv.URL + "/webhook")
		assert.NoError(t, err)

		notifier := drift_checker.NewSlackDriftNotifier(*u)
		err = notifier.Notify(drift_checker.DriftSummary{})
		assert.Error(t, err)
	})

	t.Run("returns an error if the server returns a non-200 exit code", func(t *testing.T) {
		srv, teardown := setupTest(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		})
		defer teardown()
		u, err := url.Parse(srv.URL + "/webhook")
		assert.NoError(t, err)

		notifier := drift_checker.NewSlackDriftNotifier(*u)
		err = notifier.Notify(drift_checker.DriftSummary{})
		assert.Error(t, err)
	})
}
