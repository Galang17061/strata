package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/Galang17061/strata-api/internal/domain"
)

const (
	claimId       = "Id"
	claimFullname = "Fullname"
	claimName     = "unique_name"
	claimEmail    = "email"
	claimRoleId   = "RoleId"
	claimRole     = "role"
)

type Identity struct {
	Id       string
	Fullname string
	Username string
	Email    string
	RoleId   string
	Role     string
}

type TokenIssuer struct {
	key      []byte
	issuer   string
	audience string
	lifetime time.Duration
}

func NewTokenIssuer(key, issuer, audience string, expiryMinutes int) TokenIssuer {
	return TokenIssuer{key: []byte(key), issuer: issuer, audience: audience, lifetime: time.Duration(expiryMinutes) * time.Minute}
}

func (t TokenIssuer) Issue(user domain.User, roleId domain.Guid, roleName string) (string, domain.DateTime, error) {
	expires := time.Now().Add(t.lifetime)
	claims := jwt.MapClaims{
		claimId:       user.Id.String(),
		claimFullname: user.Fullname,
		claimName:     user.UserName,
		claimEmail:    user.Email,
		claimRoleId:   roleId.String(),
		claimRole:     roleName,
		"exp":         expires.Unix(),
		"iss":         t.issuer,
		"aud":         t.audience,
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(t.key)
	return signed, domain.DateTime{Time: expires}, err
}

func (t TokenIssuer) Parse(raw string) (Identity, error) {
	claims := jwt.MapClaims{}
	_, err := jwt.ParseWithClaims(raw, claims, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return t.key, nil
	}, jwt.WithLeeway(5*time.Minute), jwt.WithValidMethods([]string{"HS256"}))
	if err != nil {
		return Identity{}, err
	}
	return Identity{
		Id:       claimString(claims, claimId),
		Fullname: claimString(claims, claimFullname),
		Username: claimString(claims, claimName),
		Email:    claimString(claims, claimEmail),
		RoleId:   claimString(claims, claimRoleId),
		Role:     claimString(claims, claimRole),
	}, nil
}

func claimString(claims jwt.MapClaims, name string) string {
	if value, ok := claims[name].(string); ok {
		return value
	}
	return ""
}
