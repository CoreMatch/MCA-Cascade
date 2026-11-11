package mcacascade

import (
	"testing"
	"time"
)

func testConfig() Config {
	return Config{
		BaseURL: "https://arena.example",
		Token:   "tok",
		Timeout: time.Second,
		Targets: map[string]string{
			"10.0.0.5:5124": "sub-a",
			"10.0.0.6:5124": "sub-b",
		},
	}
}

func TestDecideRoute(t *testing.T) {
	asn := &Assignment{Assigned: true, GroupID: 9, MatchType: "1v1", TeamID: "A",
		JoinHost: "10.0.0.5", JoinPort: 5124}
	d := Decide(testConfig(), asn)
	if d.Outcome != OutcomeRoute {
		t.Fatalf("Outcome = %v, want OutcomeRoute", d.Outcome)
	}
	if d.BackendID != "sub-a" {
		t.Errorf("BackendID = %q, want sub-a", d.BackendID)
	}
	if d.JoinTarget != "10.0.0.5:5124" {
		t.Errorf("JoinTarget = %q", d.JoinTarget)
	}
}

func TestDecideRouteCaseInsensitiveHost(t *testing.T) {
	asn := &Assignment{Assigned: true, JoinHost: "10.0.0.6", JoinPort: 5124}
	d := Decide(testConfig(), asn)
	if d.Outcome != OutcomeRoute || d.BackendID != "sub-b" {
		t.Fatalf("got %v/%q, want OutcomeRoute/sub-b", d.Outcome, d.BackendID)
	}
}

func TestDecideNoAssignment(t *testing.T) {
	d := Decide(testConfig(), &Assignment{})
	if d.Outcome != OutcomeNoAssignment {
		t.Fatalf("Outcome = %v, want OutcomeNoAssignment", d.Outcome)
	}
	if d.BackendID != "" {
		t.Errorf("BackendID = %q, want empty", d.BackendID)
	}
}

func TestDecideNilAssignment(t *testing.T) {
	if d := Decide(testConfig(), nil); d.Outcome != OutcomeNoAssignment {
		t.Fatalf("Outcome = %v, want OutcomeNoAssignment", d.Outcome)
	}
}

func TestDecideNoRoom(t *testing.T) {
	asn := &Assignment{Assigned: false, Reason: ReasonNoRoom}
	d := Decide(testConfig(), asn)
	if d.Outcome != OutcomeNoRoom {
		t.Fatalf("Outcome = %v, want OutcomeNoRoom", d.Outcome)
	}
	if d.Reason != ReasonNoRoom {
		t.Errorf("Reason = %q, want %q", d.Reason, ReasonNoRoom)
	}
}

func TestDecideNoJoinTarget(t *testing.T) {
	for _, asn := range []*Assignment{
		{Assigned: true},
		{Assigned: true, JoinHost: "10.0.0.5"},
		{Assigned: true, JoinPort: 5124},
	} {
		if d := Decide(testConfig(), asn); d.Outcome != OutcomeNoJoinTarget {
			t.Errorf("%+v: Outcome = %v, want OutcomeNoJoinTarget", asn, d.Outcome)
		}
	}
}

func TestDecideUnknownTarget(t *testing.T) {
	asn := &Assignment{Assigned: true, JoinHost: "192.168.9.9", JoinPort: 9999}
	d := Decide(testConfig(), asn)
	if d.Outcome != OutcomeUnknownTarget {
		t.Fatalf("Outcome = %v, want OutcomeUnknownTarget", d.Outcome)
	}
	if d.JoinTarget != "192.168.9.9:9999" {
		t.Errorf("JoinTarget = %q (logging must keep the unmapped target)", d.JoinTarget)
	}
}

func TestDecideEmptyTargetsMap(t *testing.T) {
	cfg := testConfig()
	cfg.Targets = nil
	asn := &Assignment{Assigned: true, JoinHost: "10.0.0.5", JoinPort: 5124}
	if d := Decide(cfg, asn); d.Outcome != OutcomeUnknownTarget {
		t.Fatalf("Outcome = %v, want OutcomeUnknownTarget (empty map)", d.Outcome)
	}
}

func TestNormalizeTarget(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"10.0.0.5:5124", "10.0.0.5:5124"},
		{" Host.Example :5124 ", "host.example:5124"},
		{"HOST:1", "host:1"},
		{"host:65535", "host:65535"},
		{"", ""},
		{"host", ""},
		{":5124", ""},
		{"host:", ""},
		{"host:0", ""},
		{"host:65536", ""},
		{"host:51a4", ""},
		{"host:-1", ""},
	}
	for _, tc := range cases {
		if got := NormalizeTarget(tc.in); got != tc.want {
			t.Errorf("NormalizeTarget(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestOutcomeString(t *testing.T) {
	if OutcomeRoute.String() != "routed" || OutcomeNoRoom.String() != "no_room_available" {
		t.Errorf("outcome strings: %v / %v", OutcomeRoute, OutcomeNoRoom)
	}
	if got := Outcome(99).String(); got != "outcome(99)" {
		t.Errorf("unknown outcome string = %q", got)
	}
}
