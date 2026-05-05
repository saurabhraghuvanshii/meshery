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

	const baseCallbackURL = "https://playground.meshery.io"

	tests := []struct {
		name           string
		refQueryParam  string
		callbackURL    string
		expected       string
		expectedDecode string
	}{
		{
			name:           "synthesizes from path when no query param",
			callbackURL:    baseCallbackURL + "/extension/meshmap",
			expected:       base64.RawURLEncoding.EncodeToString([]byte("/extension/meshmap")),
			expectedDecode: "/extension/meshmap",
		},
		{
			name:           "preserves query string in synthesized path",
			callbackURL:    baseCallbackURL + "/extension/meshmap?tab=designs",
			expected:       base64.RawURLEncoding.EncodeToString([]byte("/extension/meshmap?tab=designs")),
			expectedDecode: "/extension/meshmap?tab=designs",
		},
		{
			name:          "explicit ref query param wins",
			refQueryParam: "L2V4dGVuc2lvbi9rYW52YXM",
			callbackURL:   baseCallbackURL + "/somewhere/else",
			expected:      "L2V4dGVuc2lvbi9rYW52YXM",
		},
		{
			name:           "root path round-trips",
			callbackURL:    baseCallbackURL + "/",
			expected:       base64.RawURLEncoding.EncodeToString([]byte("/")),
			expectedDecode: "/",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			actual := computePostLoginRefValue(tc.refQueryParam, tc.callbackURL, baseCallbackURL)
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
			name:     "cookie wins over query param",
			cookie:   &http.Cookie{Name: cookieName, Value: cookieValue},
			query:    "?ref=" + queryValue,
			expected: cookieValue,
		},
		{
			name:     "falls back to query param when cookie missing",
			query:    "?ref=" + queryValue,
			expected: queryValue,
		},
		{
			name:     "falls back to query param when cookie is empty",
			cookie:   &http.Cookie{Name: cookieName, Value: ""},
			query:    "?ref=" + queryValue,
			expected: queryValue,
		},
		{
			name:     "returns empty when neither is set",
			expected: "",
		},
		{
			name: "cookie wins even when query param missing",
			cookie: &http.Cookie{
				Name:  cookieName,
				Value: cookieValue,
			},
			expected: cookieValue,
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
