//nolint:wrapcheck,err113
package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const (
	maxBcryptPasswordLength int = 72
	hashCost                int = 12
)

func HashPassword(password string) (string, error) {
	if password == "" {
		return "", errors.New("password cannot be empty")
	}

	bytePwd := []byte(password)

	if len(bytePwd) > maxBcryptPasswordLength {
		return "", fmt.Errorf(
			"password is too long, received %d bytes but maximum allowed is %d bytes",
			len(bytePwd),
			maxBcryptPasswordLength,
		)
	}

	hashedPwd, err := bcrypt.GenerateFromPassword([]byte(password), hashCost)
	if err != nil {
		return "", err
	}

	return string(hashedPwd), nil
}

func CheckPasswordHash(password, hash string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

func MakeJWT(
	userID uuid.UUID,
	tokenSecret string,
	expiresIn time.Duration,
) (string, error) {
	if userID == uuid.Nil {
		return "", errors.New("UUID cannot be nil")
	}

	if tokenSecret == "" {
		return "", errors.New("secret cannot be empty")
	}

	//nolint:exhaustruct
	claims := jwt.RegisteredClaims{
		Issuer:    "chirpy",
		IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
		ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(expiresIn)),
		Subject:   userID.String(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signedToken, err := token.SignedString([]byte(tokenSecret))
	if err != nil {
		return "", err
	}

	return signedToken, nil
}

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {
	token, err := jwt.ParseWithClaims(
		tokenString,
		//nolint:exhaustruct
		&jwt.RegisteredClaims{},
		func(_ *jwt.Token) (any, error) {
			return []byte(tokenSecret), nil
		},
	)
	if err != nil {
		return uuid.Nil, err
	}

	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok {
		return uuid.Nil, fmt.Errorf(
			"wrong type of claims, expected `*jwt.RegisteredClaims`, got `%t`",
			token.Claims,
		)
	}

	return uuid.Parse(claims.Subject)
}

func GetBearerToken(headers http.Header) (string, error) {
	token := headers.Get("Authorization")
	if token == "" {
		return "", errors.New("no bearer token present in headers")
	}

	strippedToken := strings.Replace(token, "Bearer ", "", 1)
	parts := strings.Split(strippedToken, " ")

	if len(parts) == 0 {
		return "", errors.New("no bearer token present in headers")
	}

	return parts[0], nil
}

func MakeRefreshToken() (string, error) {
	//nolint:mnd
	buf := make([]byte, 32)

	if _, err := rand.Read(buf); err != nil {
		return "", err
	}

	return hex.EncodeToString(buf), nil
}

func GetAPIKey(headers http.Header) (string, error) {
	key := headers.Get("Authorization")
	if key == "" {
		return "", errors.New("no API key present in headers")
	}

	strippedKey := strings.Replace(key, "ApiKey ", "", 1)
	parts := strings.Split(strippedKey, " ")

	if len(parts) == 0 {
		return "", errors.New("no API key present in headers")
	}

	return parts[0], nil
}
