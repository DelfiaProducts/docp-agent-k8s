package mocks

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"
)

// JWTHeader representa a estrutura do cabeçalho de um JWT
type JWTHeader struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
}

// JWTPayload representa a estrutura do payload (claims) de um JWT
type JWTPayload struct {
	Sub   string `json:"sub"`
	Email string `json:"email,omitempty"`
	IAT   int64  `json:"iat"`
	EXP   int64  `json:"exp"`
	ISS   string `json:"iss,omitempty"`
	AUD   string `json:"aud,omitempty"`
	JTI   string `json:"jti,omitempty"`
}

func GetMockJwt(email string) string {
	// 1. Criar o cabeçalho do JWT
	header := JWTHeader{
		Alg: "HS256",
		Typ: "JWT",
	}

	// Converter o cabeçalho para JSON
	headerBytes, err := json.Marshal(header)
	if err != nil {
		fmt.Printf("Erro ao serializar o cabeçalho: %v\n", err)
		return ""
	}
	// Codificar o cabeçalho JSON em Base64 URL-safe
	encodedHeader := base64.RawURLEncoding.EncodeToString(headerBytes)

	// 2. Criar o payload do JWT (claims)
	now := time.Now()
	// Expirar em 1 hora
	expirationTime := now.Add(1 * time.Hour)

	payload := JWTPayload{
		Sub:   "1234567890",
		Email: email,
		IAT:   now.Unix(),
		EXP:   expirationTime.Unix(),
		ISS:   "mock-auth-service.orya.com",
		AUD:   "auth-api.orya.com",
		JTI:   "unique-mock-jti-123",
	}

	// Converter o payload para JSON
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		fmt.Printf("Erro ao serializar o payload: %v\n", err)
		return ""
	}
	// Codificar o payload JSON em Base64 URL-safe
	encodedPayload := base64.RawURLEncoding.EncodeToString(payloadBytes)

	// 3. Criar uma "assinatura" mock (para um JWT real, isso seria uma hash criptográfica)
	// Para um mock, pode ser qualquer string codificada em Base64 URL-safe.
	// Vamos simular uma assinatura "vazia" ou um placeholder.
	mockSignatureContent := "this_is_a_mock_signature_and_not_cryptographically_secure"
	encodedSignature := base64.RawURLEncoding.EncodeToString([]byte(mockSignatureContent))

	// 4. Montar o JWT
	mockJWT := fmt.Sprintf("%s.%s.%s", encodedHeader, encodedPayload, encodedSignature)

	return mockJWT

}
