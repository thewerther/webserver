package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error) {
	timeNow := time.Now().UTC()
	expiresAt := timeNow.Add(expiresIn)
  log.Printf("Expire time for jwt token set to %v\n", expiresAt)
	claims := jwt.RegisteredClaims{
		Issuer:    "chirpy",
		IssuedAt:  jwt.NewNumericDate(timeNow),
		ExpiresAt: jwt.NewNumericDate(expiresAt),
		Subject:   userID.String(),
	}
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := jwtToken.SignedString([]byte(tokenSecret))
	if err != nil {
		return "", err
	}

	return signedToken, nil
}

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(t *jwt.Token) (interface{}, error) {
    if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
      return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
    }
		return []byte(tokenSecret), nil
	})
	if err != nil {
		return uuid.UUID{}, err
	}

  claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok {
    return uuid.UUID{}, errors.New("Unable to extract claims from token!")
  }

  subject, err := claims.GetSubject()
  if err != nil {
    return uuid.UUID{}, err
  }

  id, err := uuid.Parse(subject)
  if err != nil {
    return uuid.UUID{}, err
  }

  return id, nil
}

func GetBearerToken(headers http.Header) (string, error) {
  headerAuth := headers.Get("Authorization")
  if headerAuth == "" {
    return "", errors.New("No \"Authorization\" header in request!")
  }

  authToken := strings.TrimPrefix(headerAuth, "Bearer ")
  // No "Bearer " prefix
  if authToken == headerAuth {
    return "", errors.New("Wrong auth token request header format!")
  }

  return authToken, nil
}

func MakeRefreshToken() (string, error) {
  lenInBytes := 32
  bytes := make([]byte, lenInBytes)

  _, err := rand.Read(bytes)
  if err != nil {
    return "", err
  }

  token := hex.EncodeToString(bytes)

  return token, nil
}
