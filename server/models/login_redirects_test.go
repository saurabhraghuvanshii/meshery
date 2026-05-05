package models

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestResolvePostLoginRedirect(t *testing.T) {
	t.Parallel()

	const fallback = "/"

	tests := []struct {
		name     string
		rawRef   string
		expected string
	}{
		{
			name:     "empty ref falls back",
			rawRef:   "",
			expected: fallback,
		},
		{
			name:     "encoded in app path is decoded",
			rawRef:   base64.RawURLEncoding.EncodeToString([]byte("/extension/meshmap")),
			expected: "/extension/meshmap",
		},
		{
			name:     "plain in app path is preserved",
			rawRef:   "/extension/meshmap",
			expected: "/extension/meshmap",
		},
		{
			name:     "encoded absolute url falls back",
			rawRef:   base64.RawURLEncoding.EncodeToString([]byte("https://evil.example/phish")),
			expected: fallback,
		},
		{
			name:     "plain absolute url falls back",
			rawRef:   "https://evil.example/phish",
			expected: fallback,
		},
		{
			name:     "invalid base64 falls back",
			rawRef:   "not-base64",
			expected: fallback,
		},
		// Regression coverage: /user/login and /api/user/token are auth
		// initiation paths. Redirecting to them after a successful token
		// exchange re-enters the OAuth dance and caused Kanvas to hang on
		// the loading splash indefinitely (meshery-server-1345 followed by
		// a second InitiateLogin in the same second).
		{
			name:     "plain /user/login ref falls back",
			rawRef:   "/user/login",
			expected: fallback,
		},
		{
			name:     "/user/login with query falls back",
			rawRef:   "/user/login?provider=Layer5",
			expected: fallback,
		},
		{
			name:     "encoded /user/login ref falls back",
			rawRef:   base64.RawURLEncoding.EncodeToString([]byte("/user/login?provider=Layer5")),
			expected: fallback,
		},
		{
			name:     "plain /api/user/token ref falls back",
			rawRef:   "/api/user/token",
			expected: fallback,
		},
		{
			name:     "/provider ref falls back",
			rawRef:   "/provider?ref=xyz",
			expected: fallback,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			actual := resolvePostLoginRedirect(tc.rawRef, fallback)
			if actual != tc.expected {
				t.Fatalf("expected redirect %q, got %q", tc.expected, actual)
			}
		})
	}
}

func TestComputePostLoginRefValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		refQueryParam   string
		callbackURL     string
		baseCallbackURL string
		expected        string
		expectedDecode  string
	}{
		{
			name:            "synthesizes from path when no query param",
			callbackURL:     "https://playground.meshery.io/extension/meshmap",
			baseCallbackURL: "https://playground.meshery.io",
			expected:        base64.RawURLEncoding.EncodeToString([]byte("/extension/meshmap")),
			expectedDecode:  "/extension/meshmap",
		},
		{
			name:            "preserves query string in synthesized path",
			callbackURL:     "https://playground.meshery.io/extension/meshmap?tab=designs",
			baseCallbackURL: "https://playground.meshery.io",
			expected:        base64.RawURLEncoding.EncodeToString([]byte("/extension/meshmap?tab=designs")),
			expectedDecode:  "/extension/meshmap?tab=designs",
		},
		{
			name:            "explicit ref query param wins",
			refQueryParam:   "L2V4dGVuc2lvbi9rYW52YXM",
			callbackURL:     "https://playground.meshery.io/somewhere/else",
			baseCallbackURL: "https://playground.meshery.io",
			expected:        "L2V4dGVuc2lvbi9rYW52YXM",
		},
		{
			name:            "root path round-trips",
			callbackURL:     "https://playground.meshery.io/",
			baseCallbackURL: "https://playground.meshery.io",
			expected:        base64.RawURLEncoding.EncodeToString([]byte("/")),
			expectedDecode:  "/",
		},
		// Regression: MESHERY_SERVER_CALLBACK_URL is documented with a
		// trailing slash in our deployment examples ("https://custom-host/").
		// Pre-fix, strings.TrimPrefix produced "extension/meshmap" (no
		// leading slash), which isSafeRedirect rejects as relative and
		// resolvePostLoginRedirect then dropped to "/". Trim the trailing
		// slash from the base before stripping so deep links survive.
		{
			name:            "trailing slash on baseCallbackURL is normalized",
			callbackURL:     "https://playground.meshery.io/extension/meshmap",
			baseCallbackURL: "https://playground.meshery.io/",
			expected:        base64.RawURLEncoding.EncodeToString([]byte("/extension/meshmap")),
			expectedDecode:  "/extension/meshmap",
		},
		{
			name:            "trailing slash on baseCallbackURL with root path",
			callbackURL:     "https://playground.meshery.io/",
			baseCallbackURL: "https://playground.meshery.io/",
			expected:        base64.RawURLEncoding.EncodeToString([]byte("/")),
			expectedDecode:  "/",
		},
		// Defense-in-depth: if for any reason the prefix trim leaves a
		// non-slash-prefixed remainder (e.g. a misconfigured callbackURL
		// that doesn't actually start with baseCallbackURL), prepend "/" so
		// the result is still parseable as a relative path. resolvePost-
		// LoginRedirect's safety check is the final gate, but an absolute-
		// URL leak past the encoder is worth preventing at the source.
		{
			name:            "non-prefix callbackURL gets leading slash prepended",
			callbackURL:     "extension/meshmap",
			baseCallbackURL: "https://playground.meshery.io",
			expected:        base64.RawURLEncoding.EncodeToString([]byte("/extension/meshmap")),
			expectedDecode:  "/extension/meshmap",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			actual := computePostLoginRefValue(tc.refQueryParam, tc.callbackURL, tc.baseCallbackURL)
			if actual != tc.expected {
				t.Fatalf("expected %q, got %q", tc.expected, actual)
			}
			if tc.expectedDecode != "" {
				decoded, err := base64.RawURLEncoding.DecodeString(actual)
				if err != nil {
					t.Fatalf("expected value to be valid base64, got decode err: %v", err)
				}
				if string(decoded) != tc.expectedDecode {
					t.Fatalf("decoded value: expected %q, got %q", tc.expectedDecode, string(decoded))
				}
			}
		})
	}
}

func TestSelectPostLoginRefValue(t *testing.T) {
	t.Parallel()

	const cookieName = "playground.meshery.io_ref"
	const cookieValue = "L2V4dGVuc2lvbi9tZXNobWFw" // base64 of /extension/meshmap
	const queryValue = "L2Rhc2hib2FyZA"             // base64 of /dashboard

	tests := []struct {
		name     string
		cookie   *http.Cookie
		query    string
		expected string
	}{
		{
			name:     "cookie value is used when present",
			cookie:   &http.Cookie{Name: cookieName, Value: cookieValue},
			expected: cookieValue,
		},
		// Regression: the cookie is the SOLE source of truth. A ?ref= the
		// remote provider echoes back must never override (or fill in for)
		// the cookie — that's how the playground.meshery.io 404 escaped in
		// the first place. resolvePostLoginRedirect's "/" fallback handles
		// the missing-cookie case without us re-trusting provider state.
		{
			name:     "ignores ?ref= query param even when cookie missing",
			query:    "?ref=" + queryValue,
			expected: "",
		},
		{
			name:     "ignores ?ref= query param when cookie is empty",
			cookie:   &http.Cookie{Name: cookieName, Value: ""},
			query:    "?ref=" + queryValue,
			expected: "",
		},
		{
			name:     "returns empty when cookie is missing",
			expected: "",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(http.MethodGet, "/api/user/token"+tc.query, nil)
			if tc.cookie != nil {
				req.AddCookie(tc.cookie)
			}
			actual := selectPostLoginRefValue(req, cookieName)
			if actual != tc.expected {
				t.Fatalf("expected %q, got %q", tc.expected, actual)
			}
		})
	}
}
