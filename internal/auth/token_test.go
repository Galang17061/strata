package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Galang17061/strata-api/internal/domain"
)

func TestIssuedTokenCarriesIdentity(t *testing.T) {
	issuer := NewTokenIssuer("secret-key-secret-key-secret-key", "issuer", "audience", 10080)
	user := domain.User{Id: domain.NewGuid(), Fullname: "Jane Doe", UserName: "jane", Email: "jane@example.com"}
	roleId := domain.NewGuid()
	token, validUntil, err := issuer.Issue(user, roleId, "admin")
	require.NoError(t, err)
	assert.False(t, validUntil.IsZero())
	identity, err := issuer.Parse(token)
	require.NoError(t, err)
	assert.Equal(t, user.Id.String(), identity.Id)
	assert.Equal(t, "Jane Doe", identity.Fullname)
	assert.Equal(t, "jane", identity.Username)
	assert.Equal(t, "jane@example.com", identity.Email)
	assert.Equal(t, roleId.String(), identity.RoleId)
	assert.Equal(t, "admin", identity.Role)
}

func TestRequireRejectsMissingOrBrokenTokens(t *testing.T) {
	issuer := NewTokenIssuer("secret-key-secret-key-secret-key", "issuer", "audience", 10)
	handler := Authenticate(issuer)(Require(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(CurrentUserName(r.Context())))
	})))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
	assert.Equal(t, http.StatusUnauthorized, recorder.Code)
	assert.Equal(t, "Bearer", recorder.Header().Get("WWW-Authenticate"))
	assert.Empty(t, recorder.Body.String())

	broken := httptest.NewRequest(http.MethodGet, "/", nil)
	broken.Header.Set("Authorization", "Bearer not-a-token")
	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, broken)
	assert.Equal(t, http.StatusUnauthorized, recorder.Code)
	assert.Contains(t, recorder.Header().Get("WWW-Authenticate"), "invalid_token")

	token, _, err := issuer.Issue(domain.User{Id: domain.NewGuid(), Fullname: "Jane Doe"}, domain.EmptyGuid, "Unknown")
	require.NoError(t, err)
	good := httptest.NewRequest(http.MethodGet, "/", nil)
	good.Header.Set("Authorization", "Bearer "+token)
	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, good)
	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "Jane Doe", recorder.Body.String())
}
