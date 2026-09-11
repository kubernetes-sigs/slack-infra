// handlers_test.go
package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestEventsHandler(t *testing.T) {
	req, err := http.NewRequest("POST", "/webhook", strings.NewReader(`{
	"token": "z26uFbvR1xHJEdHE1OQiO6t8",
	"team_id": "T061EG9RZ",
	"api_app_id": "A0FFV41KK",
	"event": {
		"type": "message",
		"user": "U061F1EUR",
		"channel": "C061EG9SL",
		"text": "honk",
		"ts": "1612790186.002000",
		"channel_type": "channel"
	},
	"type": "event_callback",
	"authed_users": [],
	"authorizations": {},
	"event_id": "Ev9UQ52YNA",
	"event_context": "EC12345",
	"event_time": 1234567890
}`))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()

	h := &handler{client: nil, filters: nil}
	handler := http.Handler(h)

	// TODO: this is a failing test when the slack headers does not match
	// rewrite to pass a fake and make a happy path :)
	req.Header.Add("X-Slack-Signature", "v0=87fbffb089501ba823991cc20058df525767a8a2287b3809f9afff3e3b600dd8")
	req.Header.Add("X-Slack-Request-Timestamp", time.Now().String())
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusInternalServerError {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
}
