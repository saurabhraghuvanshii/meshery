package models

import (
	"encoding/base64"
	"net/http"
	"net/url"
	"strings"

	"github.com/meshery/meshery/server/core"
)

func resolvePostLoginRedirect(rawRef, fallback string) string {
	if rawRef == "" {
		return fallback
	}

	if decoded, err := core.DecodeRefURL(rawRef); err == nil && isSafeRedirect(decoded) {
		return decoded
	}

	if isSafeRedirect(rawRef) {
		return rawRef
	}

	return fallback
}

// selectPostLoginRefValue returns the raw (encoded or plaintext) value to
// feed into resolvePostLoginRedirect when the auth flow returns to
// TokenHandler.
//
// Meshery is the sole source of truth for its own post-login destination:
// the value is captured into a cookie at InitiateLogin time and read back
// here. We deliberately do NOT fall back to the ?ref= query param. The
// remote provider may echo a synthesized ref back to us (for example when a
// custom-domain login bounce drops our original ref and the main domain
// then auto-captures its own /login URL), and trusting that value is what
// produced the playground.meshery.io 404 in the first place. When the
// cookie is missing or empty resolvePostLoginRedirect already falls back to
// "/", which is the right behavior for callers that never went through
// InitiateLogin (mesheryctl, etc.).
func selectPostLoginRefValue(r *http.Request, cookieName string) string {
	if ck, err := r.Cookie(cookieName); err == nil {
		return ck.Value
	}
	return ""
}

// computePostLoginRefValue returns the value to store in the post-login
// redirect cookie at InitiateLogin time. An explicit ?ref= query param wins
// (callers expressing intent override our default), otherwise we synthesize
// the originally-requested in-app path from callbackURL by stripping the
// baseCallbackURL prefix. The value is left to resolvePostLoginRedirect to
// validate at read time, so this stays a pure string transform.
//
// We normalize the prefix to handle MESHERY_SERVER_CALLBACK_URL configs
// that ship with a trailing slash (the form documented in our examples).
// Without normalization, "https://host/" + "https://host/extension" produced
// "extension" (no leading slash), which isSafeRedirect rejects as relative
// and resolvePostLoginRedirect then drops to "/", silently breaking deep
// links.
func computePostLoginRefValue(refQueryParam, callbackURL, baseCallbackURL string) string {
	if refQueryParam != "" {
		return refQueryParam
	}
	rel := strings.TrimPrefix(callbackURL, strings.TrimSuffix(baseCallbackURL, "/"))
	if rel == "" || !strings.HasPrefix(rel, "/") {
		rel = "/" + rel
	}
	return base64.RawURLEncoding.EncodeToString([]byte(rel))
}

// authInitiationPaths are server routes whose job is to *start* authentication.
// Post-login redirects must never land on one of these, otherwise the browser
// immediately re-enters the OAuth dance and the original target is lost. The
// intermittent Kanvas-never-loads behavior was reproduced as exactly this:
// TokenHandler succeeded and then redirected to /user/login?provider=Layer5,
// which restarted InitiateLogin mid-mount.
var authInitiationPaths = []string{
	"/user/login",
	"/auth/login",
	"/api/user/token",
	"/provider",
}

// isSafeRedirect validates that a decoded ref URL is a relative in-app path
// to prevent open redirects. It rejects absolute URLs (with scheme/host),
// protocol-relative URLs (starting with //), and auth-initiation paths that
// would cause a post-login redirect loop.
func isSafeRedirect(rawURL string) bool {
	if rawURL == "" || strings.HasPrefix(rawURL, "//") {
		return false
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return false
	}

	if parsed.Scheme != "" || parsed.Host != "" {
		return false
	}

	if !strings.HasPrefix(rawURL, "/") {
		return false
	}

	for _, p := range authInitiationPaths {
		if parsed.Path == p || strings.HasPrefix(parsed.Path, p+"/") {
			return false
		}
	}

	return true
}
