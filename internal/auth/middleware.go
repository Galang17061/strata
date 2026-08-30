package auth

import (
	"context"
	"net/http"
	"strings"
)

type contextKey int

const identityKey contextKey = iota

type authState struct {
	identity Identity
	present  bool
	invalid  bool
}

func Authenticate(issuer TokenIssuer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			state := authState{}
			header := r.Header.Get("Authorization")
			if strings.HasPrefix(strings.ToLower(header), "bearer ") {
				raw := strings.TrimSpace(header[7:])
				if raw != "" {
					state.present = true
					identity, err := issuer.Parse(raw)
					if err != nil {
						state.invalid = true
					} else {
						state.identity = identity
					}
				}
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), identityKey, state)))
		})
	}
}

func Require(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		state, _ := r.Context().Value(identityKey).(authState)
		if !state.present || state.invalid || state.identity.Id == "" {
			challenge := "Bearer"
			if state.invalid {
				challenge = `Bearer error="invalid_token"`
			}
			w.Header().Set("WWW-Authenticate", challenge)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func IdentityFrom(ctx context.Context) (Identity, bool) {
	state, ok := ctx.Value(identityKey).(authState)
	if !ok || state.invalid || !state.present {
		return Identity{}, false
	}
	return state.identity, true
}

func CurrentUserName(ctx context.Context) string {
	identity, ok := IdentityFrom(ctx)
	if !ok || identity.Fullname == "" {
		return "Unknown"
	}
	return identity.Fullname
}
