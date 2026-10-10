package handler

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
)

// sessions keeps a script's _SESSION in a cookie the server signs, so clients can read it but not forge it.
// The value is base64url(JSON) "." base64url(HMAC-SHA256(key, JSON)).
type sessions struct {
	name string // the SessionCookieName
	key  []byte
}

var encoding = base64.RawURLEncoding

// read returns the session in r's cookie, or an empty one when it is missing or its signature is wrong
func (s sessions) read(r *http.Request) map[string]string {
	session := make(map[string]string)
	cookie, err := r.Cookie(s.name)
	if err != nil {
		return session
	}
	payload, signature, ok := strings.Cut(cookie.Value, ".")
	if !ok {
		return session
	}
	data, err1 := encoding.DecodeString(payload)
	mac, err2 := encoding.DecodeString(signature)
	if err1 != nil || err2 != nil || !hmac.Equal(mac, s.sign(data)) {
		return session
	}
	if err := json.Unmarshal(data, &session); err != nil {
		return make(map[string]string)
	}
	return session
}

// cookie stores session, or expires the cookie when session is empty
func (s sessions) cookie(session map[string]string, secure bool) *http.Cookie {
	cookie := &http.Cookie{Name: s.name, Path: "/", HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode}
	if len(session) == 0 {
		cookie.MaxAge = -1
		return cookie
	}
	data, _ := json.Marshal(session) // a map of strings always marshals
	cookie.Value = encoding.EncodeToString(data) + "." + encoding.EncodeToString(s.sign(data))
	return cookie
}

func (s sessions) sign(data []byte) []byte {
	mac := hmac.New(sha256.New, s.key)
	mac.Write(data)
	return mac.Sum(nil)
}
