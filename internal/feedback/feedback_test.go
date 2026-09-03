package feedback

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

func TestOnlyTheThreeKnownKindsOfNoteAreAccepted(t *testing.T) {
	assert.True(t, ValidCategory("bug"))
	assert.True(t, ValidCategory("idea"))
	assert.True(t, ValidCategory("question"))
	assert.False(t, ValidCategory("rant"))
	assert.False(t, ValidCategory(""))
}

func TestANoteNeedsASignedInSender(t *testing.T) {
	issuer := auth.NewTokenIssuer("0123456789012345678901234567890123456789012345678901234567890123", "issuer", "audience", 60)
	mux := web.NewRouter()
	mux.Use(auth.Authenticate(issuer))
	NewHandler(NewStore(nil), nil, "").Mount(mux)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/Feedback", strings.NewReader(`{"Category":"idea","Message":"hello"}`))
	mux.ServeHTTP(recorder, request)
	assert.Equal(t, http.StatusUnauthorized, recorder.Code)
}

func TestAnEmptyOrMislabelledNoteIsTurnedBack(t *testing.T) {
	issuer := auth.NewTokenIssuer("0123456789012345678901234567890123456789012345678901234567890123", "issuer", "audience", 60)
	token, _, err := issuer.Issue(domain.User{Id: domain.NewGuid(), Fullname: "Admin", UserName: "admin"}, domain.NewGuid(), "admin")
	require.NoError(t, err)
	mux := web.NewRouter()
	mux.Use(auth.Authenticate(issuer))
	NewHandler(NewStore(nil), nil, "").Mount(mux)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/Feedback", strings.NewReader(`{"Category":"rant","Message":"  "}`))
	request.Header.Set("Authorization", "Bearer "+token)
	mux.ServeHTTP(recorder, request)
	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	var problem web.ValidationProblem
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &problem))
	assert.Contains(t, problem.Errors, "Category")
	assert.Contains(t, problem.Errors, "Message")
}
