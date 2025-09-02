package utils

import (
	"encoding/json"

	"github.com/DelfiaProducts/docp-agent-k8s/dto"
	"github.com/golang-jwt/jwt/v5"
)

// DecodeJwt execute decode the jwt
func DecodeJwt(token string) (dto.AuthTokenClaims, error) {
	var claims dto.AuthTokenClaims
	tokenJwt, _, err := new(jwt.Parser).ParseUnverified(token, jwt.MapClaims{})
	if err != nil {
		return claims, err
	}

	if tokenClaims, ok := tokenJwt.Claims.(jwt.MapClaims); ok {
		jsonData, err := json.Marshal(tokenClaims)
		if err != nil {
			return claims, err
		}
		if err := json.Unmarshal(jsonData, &claims); err != nil {
			return claims, err
		}
	} else {
		return claims, ErrAuthTokenClaimsInvalid()
	}
	return claims, nil
}
