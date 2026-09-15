package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSubstituteVariables(t *testing.T) {
	tests := []struct {
		name string
		tmpl string
		vars map[string]string
		want string
	}{
		{
			name: "single substitution",
			tmpl: "Hi SIG {{SIG}}!",
			vars: map[string]string{"SIG": "Storage"},
			want: "Hi SIG Storage!",
		},
		{
			name: "multiple substitutions",
			tmpl: "{{USER}} says hi to SIG {{SIG}} about {{RELEASE}}",
			vars: map[string]string{"USER": "@alice", "SIG": "Node", "RELEASE": "v1.37"},
			want: "@alice says hi to SIG Node about v1.37",
		},
		{
			name: "unmatched placeholder is left untouched",
			tmpl: "Hi SIG {{SIG}}, deadline {{DEADLINE}}",
			vars: map[string]string{"SIG": "Network"},
			want: "Hi SIG Network, deadline {{DEADLINE}}",
		},
		{
			name: "no variables",
			tmpl: "Plain message, no placeholders",
			vars: map[string]string{},
			want: "Plain message, no placeholders",
		},
		{
			name: "repeated placeholder",
			tmpl: "{{SIG}} {{SIG}}",
			vars: map[string]string{"SIG": "Auth"},
			want: "Auth Auth",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := substituteVariables(tt.tmpl, tt.vars)
			if got != tt.want {
				t.Errorf("substituteVariables(%q, %v) = %q, want %q", tt.tmpl, tt.vars, got, tt.want)
			}
		})
	}
}

func TestFetchTemplate(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("Hi SIG {{SIG}}!"))
		}))
		defer srv.Close()

		got, err := fetchTemplate(srv.URL)
		if err != nil {
			t.Fatalf("fetchTemplate returned error: %v", err)
		}
		if got != "Hi SIG {{SIG}}!" {
			t.Errorf("fetchTemplate() = %q, want %q", got, "Hi SIG {{SIG}}!")
		}
	})

	t.Run("non-200 status is an error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer srv.Close()

		if _, err := fetchTemplate(srv.URL); err == nil {
			t.Error("fetchTemplate() expected an error for a 404 response, got nil")
		}
	})
}
