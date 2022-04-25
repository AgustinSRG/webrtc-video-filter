// Authentication token generator

package main

import "github.com/golang-jwt/jwt"

func generateToken(secret string, streamId string) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "stream_publish",
		"sid": streamId,
	})

	tokenb64, e := token.SignedString([]byte(secret))

	if e != nil {
		return ""
	}

	return tokenb64
}
