package account

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Galang17061/strata-api/internal/auth"
	"github.com/Galang17061/strata-api/internal/domain"
	"github.com/Galang17061/strata-api/internal/web"
)

func mountedRouter(issuer auth.TokenIssuer) http.Handler {
	mux := web.NewRouter()
	mux.Use(auth.Authenticate(issuer))
	NewHandler(NewService(nil, auth.NewCipher("key"), issuer, nil, "http://localhost:3000")).Mount(mux)
	return mux
}

func TestResetPasswordNeedsASignedInCaller(t *testing.T) {
	issuer := auth.NewTokenIssuer("0123456789012345678901234567890123456789012345678901234567890123", "issuer", "audience", 60)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPut, "/api/User/ChangePasswordAdmin?UserId=1b4e28ba-2fa1-11d2-883f-0016d3cca427", strings.NewReader(`{"PasswordNew":"a","ReconfirmPassword":"a"}`))
	mountedRouter(issuer).ServeHTTP(recorder, request)
	assert.Equal(t, http.StatusUnauthorized, recorder.Code)
	assert.Equal(t, "Bearer", recorder.Header().Get("WWW-Authenticate"))

	recorder = httptest.NewRecorder()
	request.Header.Set("Authorization", "Bearer not-a-token")
	mountedRouter(issuer).ServeHTTP(recorder, request)
	assert.Equal(t, http.StatusUnauthorized, recorder.Code)
	assert.Equal(t, `Bearer error="invalid_token"`, recorder.Header().Get("WWW-Authenticate"))
}

func TestResetPasswordRejectsMissingFieldsBeforeTouchingStorage(t *testing.T) {
	issuer := auth.NewTokenIssuer("0123456789012345678901234567890123456789012345678901234567890123", "issuer", "audience", 60)
	token, _, err := issuer.Issue(domain.User{Id: domain.NewGuid(), Fullname: "Admin", UserName: "admin"}, domain.NewGuid(), "admin")
	require.NoError(t, err)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPut, "/api/user/changepasswordadmin?userId=1b4e28ba-2fa1-11d2-883f-0016d3cca427", strings.NewReader(`{"PasswordNew":"a"}`))
	request.Header.Set("Authorization", "Bearer "+token)
	mountedRouter(issuer).ServeHTTP(recorder, request)
	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Equal(t, "application/problem+json; charset=utf-8", recorder.Header().Get("Content-Type"))
	var problem web.ValidationProblem
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &problem))
	assert.Equal(t, map[string][]string{"ReconfirmPassword": {"The ReconfirmPassword field is required."}}, problem.Errors)
}
