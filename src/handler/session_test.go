package handler

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestSessionRoundTrip(t *testing.T) {
	s := sessions{name: "GopSession", key: []byte("key")}
	want := map[string]string{"user": "bob", "role": "a=b; c"}

	cookie := s.cookie(want, false)
	if !cookie.HttpOnly || cookie.Path != "/" || cookie.SameSite != http.SameSiteLaxMode || cookie.Secure {
		t.Errorf("cookie attributes: %+v", cookie)
	}
	if got := s.read(requestWith(cookie)); !reflect.DeepEqual(got, want) {
		t.Errorf("read back %v, want %v", got, want)
	}
	if !s.cookie(want, true).Secure {
		t.Error("cookie over TLS is not Secure")
	}
}

func TestSessionRejectsTampering(t *testing.T) {
	s := sessions{name: "GopSession", key: []byte("key")}
	valid := s.cookie(map[string]string{"admin": "no"}, false).Value
	forged := sessions{name: "GopSession", key: []byte("other")}.cookie(map[string]string{"admin": "yes"}, false).Value

	for _, value := range []string{
		"eyJhZG1pbiI6InllcyJ9", // the old unsigned format
		forged,
		valid[:len(valid)-2] + "AA",
		"not base64!.x",
		"",
	} {
		cookie := &http.Cookie{Name: "GopSession", Value: value}
		if got := s.read(requestWith(cookie)); len(got) != 0 {
			t.Errorf("%q gave session %v", value, got)
		}
	}
}

func TestEmptySessionExpiresCookie(t *testing.T) {
	cookie := sessions{name: "GopSession", key: []byte("key")}.cookie(map[string]string{}, false)
	if cookie.MaxAge >= 0 || cookie.Value != "" {
		t.Errorf("cookie %+v does not expire", cookie)
	}
}

func requestWith(cookie *http.Cookie) *http.Request {
	r := httptest.NewRequest("GET", "/", nil)
	r.AddCookie(cookie)
	return r
}
