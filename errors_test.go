package omnisignal_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/plexusone/omnisignal"
)

func TestWrapHTTPError(t *testing.T) {
	baseErr := errors.New("api error")

	tests := []struct {
		name       string
		statusCode int
		wantErr    error
	}{
		{"401 unauthorized", http.StatusUnauthorized, omnisignal.ErrAuthentication},
		{"403 forbidden", http.StatusForbidden, omnisignal.ErrAuthentication},
		{"429 rate limited", http.StatusTooManyRequests, omnisignal.ErrRateLimited},
		{"500 server error", http.StatusInternalServerError, baseErr},
		{"200 ok", http.StatusOK, baseErr},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &http.Response{StatusCode: tt.statusCode}
			wrapped := omnisignal.WrapHTTPError(baseErr, resp, "test")
			if !errors.Is(wrapped, tt.wantErr) {
				t.Errorf("WrapHTTPError with %d: got %v, want %v", tt.statusCode, wrapped, tt.wantErr)
			}
		})
	}
}

func TestWrapHTTPErrorNilResponse(t *testing.T) {
	baseErr := errors.New("api error")
	wrapped := omnisignal.WrapHTTPError(baseErr, nil, "test")
	if !errors.Is(wrapped, baseErr) {
		t.Errorf("WrapHTTPError with nil resp: got %v, want %v", wrapped, baseErr)
	}
}

func TestWrapHTTPErrorNilError(t *testing.T) {
	resp := &http.Response{StatusCode: http.StatusUnauthorized}
	wrapped := omnisignal.WrapHTTPError(nil, resp, "test")
	if wrapped != nil {
		t.Errorf("WrapHTTPError with nil err: got %v, want nil", wrapped)
	}
}

func TestWrapErrorByMessage(t *testing.T) {
	tests := []struct {
		name    string
		errMsg  string
		wantErr error
	}{
		{"401 in message", "HTTP 401: unauthorized", omnisignal.ErrAuthentication},
		{"403 in message", "HTTP 403: forbidden", omnisignal.ErrAuthentication},
		{"unauthorized text", "request unauthorized", omnisignal.ErrAuthentication},
		{"forbidden text", "access forbidden", omnisignal.ErrAuthentication},
		{"authentication text", "authentication failed", omnisignal.ErrAuthentication},
		{"invalid token", "invalid token provided", omnisignal.ErrAuthentication},
		{"429 in message", "HTTP 429: too many requests", omnisignal.ErrRateLimited},
		{"rate limit text", "rate limit exceeded", omnisignal.ErrRateLimited},
		{"too many requests", "too many requests", omnisignal.ErrRateLimited},
		{"throttled", "request throttled", omnisignal.ErrRateLimited},
		{"generic error", "some other error", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseErr := errors.New(tt.errMsg)
			wrapped := omnisignal.WrapErrorByMessage(baseErr, "test")
			if tt.wantErr == nil {
				if errors.Is(wrapped, omnisignal.ErrAuthentication) || errors.Is(wrapped, omnisignal.ErrRateLimited) {
					t.Errorf("expected no sentinel, got %v", wrapped)
				}
			} else if !errors.Is(wrapped, tt.wantErr) {
				t.Errorf("WrapErrorByMessage(%q): got %v, want %v", tt.errMsg, wrapped, tt.wantErr)
			}
		})
	}
}

func TestWrapErrorByMessageNilError(t *testing.T) {
	wrapped := omnisignal.WrapErrorByMessage(nil, "test")
	if wrapped != nil {
		t.Errorf("WrapErrorByMessage with nil err: got %v, want nil", wrapped)
	}
}
