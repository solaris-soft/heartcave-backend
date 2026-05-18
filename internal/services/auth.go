package services

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const AccessTokenExpiry = time.Minute * 15

const RefreshTokenExpiry = 30 * 24 * time.Hour

type AuthService struct {
	secretKey string
}

func NewAuthService(secretKey string) AuthService {
	if len(secretKey) < 32 {
		log.Fatal("Secret key must be secure (at least 32 characters)")
	}
	return AuthService{secretKey}
}

type HeartCaveClaims struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
}

func (a AuthService) MakeAccessToken(userID uuid.UUID, role string) (string, error) {
	claims := HeartCaveClaims{
		Role: role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			Issuer:    "heartcave",
			Audience:  jwt.ClaimStrings{"public-api"},
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(AccessTokenExpiry)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString([]byte(a.secretKey))
}

func (a AuthService) ValidatePassword(password string) bool {
	return len(password) >= 8
}

func (a AuthService) ValidateAccessToken(tokenString string) (uuid.UUID, string, error) {
	claims := &HeartCaveClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(a.secretKey), nil
	})
	if err != nil {
		return uuid.Nil, "", err
	}

	if !token.Valid {
		return uuid.Nil, "", jwt.ErrTokenInvalidClaims
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return uuid.Nil, "", err
	}

	return userID, claims.Role, nil
}

func (a AuthService) NewRefreshToken() (string, error) {
	bytes := make([]byte, 32)

	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func (a AuthService) HashRefreshToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func (a AuthService) HashPassword(password string) (string, error) {
	return argon2id.CreateHash(password, argon2id.DefaultParams)
}

func (a AuthService) CompareHashPassword(password, hash string) (bool, error) {
	return argon2id.ComparePasswordAndHash(password, hash)
}

func (a AuthService) GetBearerToken(headers http.Header) (string, error) {
	authHeader := headers.Get("Authorization")
	if authHeader == "" {
		return "", errors.New("no authorization header provided")
	}
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", errors.New("malformed authorization header")
	}
	return parts[1], nil
}
