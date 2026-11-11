package mcacascade

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func newTestServer(t *testing.T, status int, body string) (*httptest.Server, *[]string) {
	t.Helper()
	var auths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auths = append(auths, r.Header.Get("Authorization"))
		if !strings.HasPrefix(r.URL.Path, "/api/matchmaking/assignment/") {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv, &auths
}

func TestQueryAssigned(t *testing.T) {
	srv, auths := newTestServer(t, http.StatusOK,
		`{"success":true,"data":{"assigned":true,"group_id":3,"match_type":"1v1","team_id":"B","join_target":{"host":"Host.Example","port":5124}}}`)
	c := NewHTTPClient(srv.URL, "ops-token", time.Second)
	asn, err := c.Query(context.Background(), "uuid-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !asn.Assigned {
		t.Fatal("Assigned = false, want true")
	}
	// 解析层保留原样，归一化发生在 JoinTargetKey/映射查找。
	if asn.JoinHost != "Host.Example" || asn.JoinPort != 5124 {
		t.Errorf("join target = %s:%d", asn.JoinHost, asn.JoinPort)
	}
	if got := asn.JoinTargetKey(); got != "host.example:5124" {
		t.Errorf("JoinTargetKey = %q, want host.example:5124", got)
	}
	if len(*auths) != 1 || (*auths)[0] != "Bearer ops-token" {
		t.Errorf("Authorization = %v, want [Bearer ops-token]", *auths)
	}
}

func TestQueryNotAssigned(t *testing.T) {
	srv, _ := newTestServer(t, http.StatusOK, `{"success":true,"data":{"assigned":false}}`)
	c := NewHTTPClient(srv.URL, "tok", time.Second)
	asn, err := c.Query(context.Background(), "uuid-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if asn.Assigned || asn.Reason != "" {
		t.Errorf("got assigned=%v reason=%q, want false/empty", asn.Assigned, asn.Reason)
	}
}

func TestQueryNotFoundIsNoAssignment(t *testing.T) {
	srv, _ := newTestServer(t, http.StatusNotFound, `{"success":false}`)
	c := NewHTTPClient(srv.URL, "tok", time.Second)
	asn, err := c.Query(context.Background(), "uuid-1")
	if err != nil {
		t.Fatalf("404 must be treated as no assignment, got error: %v", err)
	}
	if asn.Assigned {
		t.Error("Assigned = true, want false")
	}
}

func TestQueryUnavailableStatuses(t *testing.T) {
	for _, status := range []int{http.StatusInternalServerError, http.StatusBadGateway, http.StatusUnauthorized, http.StatusForbidden} {
		srv, _ := newTestServer(t, status, `{}`)
		c := NewHTTPClient(srv.URL, "tok", time.Second)
		_, err := c.Query(context.Background(), "uuid-1")
		if !errors.Is(err, ErrUnavailable) {
			t.Errorf("status %d: err = %v, want ErrUnavailable", status, err)
		}
	}
}

func TestQueryUnavailableBadEnvelope(t *testing.T) {
	srv, _ := newTestServer(t, http.StatusOK, `{"success":false,"message":"down"}`)
	c := NewHTTPClient(srv.URL, "tok", time.Second)
	_, err := c.Query(context.Background(), "uuid-1")
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("err = %v, want ErrUnavailable", err)
	}
}

func TestQueryUnreachable(t *testing.T) {
	// 指向一个立即关闭的服务器地址 → 连接失败。
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	addr := srv.URL
	srv.Close()
	c := NewHTTPClient(addr, "tok", 500*time.Millisecond)
	_, err := c.Query(context.Background(), "uuid-1")
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("err = %v, want ErrUnavailable", err)
	}
}

func TestQueryTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(300 * time.Millisecond)
		_, _ = w.Write([]byte(`{"success":true,"data":{"assigned":false}}`))
	}))
	t.Cleanup(srv.Close)
	c := NewHTTPClient(srv.URL, "tok", 50*time.Millisecond)
	_, err := c.Query(context.Background(), "uuid-1")
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("err = %v, want ErrUnavailable (timeout)", err)
	}
}

func TestQueryRejectsBadUUID(t *testing.T) {
	srv, auths := newTestServer(t, http.StatusOK, `{"success":true,"data":{"assigned":false}}`)
	c := NewHTTPClient(srv.URL, "tok", time.Second)
	for _, uuid := range []string{"", strings.Repeat("x", maxUUIDLen+1)} {
		_, err := c.Query(context.Background(), uuid)
		if !errors.Is(err, ErrUnavailable) {
			t.Errorf("uuid %q: err = %v, want ErrUnavailable", uuid, err)
		}
	}
	if len(*auths) != 0 {
		t.Errorf("requests reached server = %d, want 0 (invalid uuids must not be sent)", len(*auths))
	}
}

func TestQueryEscapesUUID(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		_, _ = w.Write([]byte(`{"success":true,"data":{"assigned":false}}`))
	}))
	t.Cleanup(srv.Close)
	c := NewHTTPClient(srv.URL+"/", "tok", time.Second) // 尾部斜杠必须被去掉
	if _, err := c.Query(context.Background(), "a b/c"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotPath != "/api/matchmaking/assignment/a%20b%2Fc" {
		t.Errorf("path = %q, want escaped uuid segment", gotPath)
	}
}

func TestQueryContextCancel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(300 * time.Millisecond)
	}))
	t.Cleanup(srv.Close)
	c := NewHTTPClient(srv.URL, "tok", time.Second)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := c.Query(ctx, "uuid-1")
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("err = %v, want ErrUnavailable", err)
	}
}
