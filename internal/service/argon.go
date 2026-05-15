package service

import (
	"errors"
	"strconv"
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/golang-jwt/jwt/v5"
)

// Argon2Hasher hashes and verifies passwords using bcrypt.
type Argon2Hasher struct{}

func (a Argon2Hasher) Hash(password string) (string, error) {
	return argon2id.CreateHash(password, argon2id.DefaultParams)
}

func (Argon2Hasher) VerifyPassword(password, hash string) error {
	match, err := argon2id.ComparePasswordAndHash(password, hash)
	if err != nil {
		return err
	}
	if !match {
		return errors.New("invalid password")
	}

	return nil
}

// JWTIssuer issues short JSON Web Tokens for API authentication.
type JWTIssuer struct {
	Secret string
}

func (j JWTIssuer) IssueCustomerToken(customerID int64) (string, error) {
	return j.issue(strconv.FormatInt(customerID, 10))
}

func (j JWTIssuer) IssueAdminToken() (string, error) {
	return j.issue("admin")
}

func (j JWTIssuer) issue(subject string) (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Subject:   subject,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(7 * 24 * time.Hour)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(j.Secret))
}
