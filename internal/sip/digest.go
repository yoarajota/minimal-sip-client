package sip

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"strings"
)

// HTTP digest authentication (§22.4). The client computes the Authorization
// header for a challenge it received in a 401/407 response.

type digestParams struct {
	Realm, Nonce, QOP, Opaque, Algorithm string
}

func parseDigest(challenge string) digestParams {
	var p digestParams
	for _, kv := range strings.Split(strings.TrimPrefix(challenge, "Digest "), ",") {
		kv = strings.TrimSpace(kv)
		idx := strings.Index(kv, "=")
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(kv[:idx])
		val := strings.Trim(strings.TrimSpace(kv[idx+1:]), `"`)
		switch key {
		case "realm":
			p.Realm = val
		case "nonce":
			p.Nonce = val
		case "qop":
			p.QOP = val
		case "opaque":
			p.Opaque = val
		case "algorithm":
			p.Algorithm = val
		}
	}
	return p
}

func md5hex(s string) string {
	sum := md5.Sum([]byte(s))
	return hex.EncodeToString(sum[:])
}

// Authorization builds the Authorization header value for a request, per
// RFC 2617 §3.2.2 (digest) as adopted by §22.4. Supports qop="auth" and the
// no-qop legacy form.
func Authorization(challenge, method, uri, user, password string) string {
	p := parseDigest(challenge)
	ha1 := md5hex(user + ":" + p.Realm + ":" + password)
	ha2 := md5hex(method + ":" + uri)
	opaque := ""
	if p.Opaque != "" {
		opaque = fmt.Sprintf(`, opaque="%s"`, p.Opaque)
	}
	if strings.Contains(p.QOP, "auth") {
		nc := "00000001"
		cnonce := randHex(8)
		response := md5hex(strings.Join([]string{ha1, p.Nonce, nc, cnonce, "auth", ha2}, ":"))
		return fmt.Sprintf(`Digest username="%s", realm="%s", nonce="%s", uri="%s", qop=auth, nc=%s, cnonce="%s", response="%s"%s`,
			user, p.Realm, p.Nonce, uri, nc, cnonce, response, opaque)
	}
	response := md5hex(ha1 + ":" + p.Nonce + ":" + ha2)
	return fmt.Sprintf(`Digest username="%s", realm="%s", nonce="%s", uri="%s", response="%s"%s`,
		user, p.Realm, p.Nonce, uri, response, opaque)
}
