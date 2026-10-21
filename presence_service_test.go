package main

import (
	"context"
	"testing"
)

type fakeAPI struct {
	created, published bool
	users              []string
}

func (f *fakeAPI) CreateChannel(context.Context, string) error { f.created = true; return nil }
func (f *fakeAPI) Publish(context.Context, string, string, any, string) error {
	f.published = true
	return nil
}
func (f *fakeAPI) Presence(context.Context, string) ([]string, error) { return f.users, nil }
func (f *fakeAPI) IssueToken(context.Context, string, []string) (string, error) {
	return "desk-token", nil
}

func TestIntakeDecision(t *testing.T) {
	tests := []struct {
		name, due string
		want      bool
	}{{"deadline is tracked", "2026-10-01", true}, {"no deadline", "", false}}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			api := &fakeAPI{users: []string{"alex"}}
			got, err := (Service{api: api}).Intake(context.Background(), Matter{ID: "42", DueDate: tc.due})
			if err != nil || got["deadline_follow_up"] != tc.want || got["client_token"] != "desk-token" || !api.created || !api.published {
				t.Fatalf("result=%v err=%v", got, err)
			}
		})
	}
}
