package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/krkn-chaos/krkn-operator/pkg/auth"
)

func TestLogoutSessionClearsSecureCookie(t *testing.T) {
	h := &Handler{}
	get := httptest.NewRecorder()
	h.LogoutSession(get, httptest.NewRequest(http.MethodGet, AuthBasePath+"/logout", nil))
	if get.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET logout returned %d", get.Code)
	}

	post := httptest.NewRecorder()
	postRequest := httptest.NewRequest(http.MethodPost, AuthBasePath+"/logout", nil)
	postRequest.Header.Set("X-CSRF-Token", "csrf-value")
	h.LogoutSession(post, postRequest)
	if post.Code != http.StatusNoContent {
		t.Fatalf("POST logout returned %d", post.Code)
	}
	cookies := post.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != auth.SessionCookieName ||
		cookies[0].Value != "" || cookies[0].MaxAge >= 0 || cookies[0].Path != "/" ||
		!cookies[0].HttpOnly || !cookies[0].Secure || cookies[0].SameSite != http.SameSiteLaxMode {
		t.Fatalf("logout did not clear the secure session cookie: %+v", cookies)
	}
	withoutHeader := httptest.NewRecorder()
	h.LogoutSession(withoutHeader, httptest.NewRequest(http.MethodPost, AuthBasePath+"/logout", nil))
	if withoutHeader.Code != http.StatusForbidden {
		t.Fatalf("logout without a session header returned %d", withoutHeader.Code)
	}
}
