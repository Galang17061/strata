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
	"github.com/Galang17061/strata-api/internal/web"
)

func TestTokensAreLongRandomAndVerifiableOnlyByTheirDigest(t *testing.T) {
	first, firstDigest, err := newToken()
	require.NoError(t, err)
	second, secondDigest, err := newToken()
	require.NoError(t, err)
	assert.Len(t, first, 64)
	assert.Len(t, firstDigest, 64)
	assert.NotEqual(t, first, second)
	assert.NotEqual(t, firstDigest, secondDigest)
	assert.NotEqual(t, first, firstDigest)
	assert.Equal(t, firstDigest, hashToken(first))
}

func TestForgotPasswordInsistsOnAnEmail(t *testing.T) {
	issuer := auth.NewTokenIssuer("0123456789012345678901234567890123456789012345678901234567890123", "issuer", "audience", 60)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/Auth/ForgotPassword", strings.NewReader(`{}`))
	mountedRouter(issuer).ServeHTTP(recorder, request)
	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	var problem web.ValidationProblem
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &problem))
	assert.Equal(t, map[string][]string{"Email": {"The Email field is required."}}, problem.Errors)
}

func TestAcceptInviteListsEverythingItStillNeeds(t *testing.T) {
	issuer := auth.NewTokenIssuer("0123456789012345678901234567890123456789012345678901234567890123", "issuer", "audience", 60)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/User/AcceptInvite", strings.NewReader(`{"Token":"x"}`))
	mountedRouter(issuer).ServeHTTP(recorder, request)
	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	var problem web.ValidationProblem
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &problem))
	assert.Contains(t, problem.Errors, "UserName")
	assert.Contains(t, problem.Errors, "Fullname")
	assert.Contains(t, problem.Errors, "Password")
	assert.Contains(t, problem.Errors, "ReconfirmPassword")
}

func TestInvitingSomeoneNeedsASignedInCaller(t *testing.T) {
	issuer := auth.NewTokenIssuer("0123456789012345678901234567890123456789012345678901234567890123", "issuer", "audience", 60)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/User/Invite", strings.NewReader(`{"Email":"a@b.c"}`))
	mountedRouter(issuer).ServeHTTP(recorder, request)
	assert.Equal(t, http.StatusUnauthorized, recorder.Code)
}
