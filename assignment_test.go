package mcacascade

import "testing"

func TestParseAssignmentResponseAssignedWithTarget(t *testing.T) {
	body := []byte(`{"success":true,"message":"ok","data":{
		"assigned":true,"group_id":42,"match_type":"1v1",
		"formed_at":"2026-11-11T12:00:00Z","team_id":"A",
		"room":"arena-1",
		"join_target":{"host":"10.0.0.5","port":5124}}}`)
	asn, err := ParseAssignmentResponse(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !asn.Assigned {
		t.Fatal("Assigned = false, want true")
	}
	if asn.GroupID != 42 || asn.MatchType != "1v1" || asn.TeamID != "A" {
		t.Errorf("identity fields = %d/%s/%s, want 42/1v1/A", asn.GroupID, asn.MatchType, asn.TeamID)
	}
	if asn.JoinHost != "10.0.0.5" || asn.JoinPort != 5124 {
		t.Errorf("join target = %s:%d, want 10.0.0.5:5124", asn.JoinHost, asn.JoinPort)
	}
	if got := asn.JoinTargetKey(); got != "10.0.0.5:5124" {
		t.Errorf("JoinTargetKey = %q, want 10.0.0.5:5124", got)
	}
}

func TestParseAssignmentResponseAssignedNoTarget(t *testing.T) {
	body := []byte(`{"success":true,"data":{"assigned":true,"group_id":7,"match_type":"2v2"}}`)
	asn, err := ParseAssignmentResponse(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !asn.Assigned {
		t.Fatal("Assigned = false, want true")
	}
	if asn.HasJoinTarget() {
		t.Error("HasJoinTarget = true, want false (join_target omitted)")
	}
	if got := asn.JoinTargetKey(); got != "" {
		t.Errorf("JoinTargetKey = %q, want empty", got)
	}
}

func TestParseAssignmentResponseNotAssignedPlain(t *testing.T) {
	body := []byte(`{"success":true,"data":{"assigned":false}}`)
	asn, err := ParseAssignmentResponse(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if asn.Assigned || asn.Reason != "" {
		t.Errorf("got assigned=%v reason=%q, want false/empty", asn.Assigned, asn.Reason)
	}
}

func TestParseAssignmentResponseNotAssignedNoRoom(t *testing.T) {
	body := []byte(`{"success":true,"data":{"assigned":false,"reason":"no_room_available"}}`)
	asn, err := ParseAssignmentResponse(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if asn.Assigned {
		t.Fatal("Assigned = true, want false")
	}
	if asn.Reason != ReasonNoRoom {
		t.Errorf("Reason = %q, want %q", asn.Reason, ReasonNoRoom)
	}
}

func TestParseAssignmentResponseEnvelopeFailures(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"success false", `{"success":false,"message":"boom"}`},
		{"not json", `{{{`},
		{"empty", ``},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := ParseAssignmentResponse([]byte(tc.body)); err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}

func TestAssignmentNilSafe(t *testing.T) {
	var asn *Assignment
	if asn.HasJoinTarget() {
		t.Error("nil.HasJoinTarget = true, want false")
	}
	if got := asn.JoinTargetKey(); got != "" {
		t.Errorf("nil.JoinTargetKey = %q, want empty", got)
	}
}
