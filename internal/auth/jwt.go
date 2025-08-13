package auth

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)


func MakeJWT (userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error) {

	token := jwt.NewWithClaims(jwt.SigningMethodHS256,jwt.RegisteredClaims{
		Issuer: "chirpy",
		IssuedAt: jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiresIn)),
		Subject: userID.String(),
	})

	signedToken, err := token.SignedString([]byte(tokenSecret))

	if err != nil {
		return "", err
	}

	return signedToken, nil
}


func ValidateJWT (tokenString, tokenSecret string) (uuid.UUID, error) {
	var emptyUUID uuid.UUID

	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate the signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		// Provide the secret key for verification
		return []byte(tokenSecret), nil
	})

	if err != nil {
		return emptyUUID, err // Return the error from parsing or validation
	}

	// Check if the token is valid (includes expiration, etc.)
	if !token.Valid {
		return emptyUUID, fmt.Errorf("invalid token")
	}

	// Type assert the claims to access the Subject
	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok {
		return emptyUUID, fmt.Errorf("could not get claims from token")
	}

	// Parse the Subject string back to a UUID
	parsedUserID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return emptyUUID, fmt.Errorf("could not parse userID from subject: %w", err)
	}

	return parsedUserID, nil
}



func GetBearerToken (headers http.Header) (string, error) {

	if len(headers.Values("Authorization")) <= 0 {
		return "", fmt.Errorf("missing auth token")
	}
	tokenString := headers.Values("Authorization")[0]
	token := strings.TrimSpace(strings.TrimPrefix(tokenString, "Bearer "))

	if token == "" {
		return "", fmt.Errorf("missing auth token")
	}

	return token, nil
}
