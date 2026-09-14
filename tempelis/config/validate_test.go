package config

import (
	"strings"
	"testing"
)

func TestValidate(t *testing.T) {
	base := func() *Config {
		return &Config{
			Users:    map[string]string{"alice": "U1"},
			Channels: []Channel{{Name: "some-channel"}},
		}
	}
	tests := []struct {
		name    string
		group   Usergroup
		wantErr string
	}{
		{"ok", Usergroup{Name: "some-group", Description: "fine", Members: []string{"alice"}}, ""},
		{"unknown member", Usergroup{Name: "g", Description: "fine", Members: []string{"bob"}}, "unknown user names: bob"},
		{"description too long", Usergroup{Name: "g", Description: strings.Repeat("á", 141), Members: []string{"alice"}}, "141 characters"},
		{"handle equals channel", Usergroup{Name: "some-channel", Description: "fine", Members: []string{"alice"}}, "same name as a channel"},
		{"external skipped", Usergroup{Name: "some-channel", External: true}, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := base()
			c.Usergroups = []Usergroup{tc.group}
			err := c.Validate()
			if tc.wantErr == "" && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tc.wantErr)) {
				t.Fatalf("want error containing %q, got %v", tc.wantErr, err)
			}
		})
	}
}
