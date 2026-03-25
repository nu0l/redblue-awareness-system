package httpserver

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"
)

type JWTClaims struct {
	Sub  string `json:"sub"`
	Role string `json:"role"`
	Iat  int64  `json:"iat"`
	Exp  int64  `json:"exp"`
}

type jwtHeader struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
}

var errUnauthorized = errors.New("unauthorized")

// ErrForbidden 表示已登录但角色不满足（例如非 admin 访问管理接口）。
var ErrForbidden = errors.New("forbidden")

func getJWTSecret() []byte {
	sec := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	return []byte(sec)
}

func signJWT(sub string, role string, ttl time.Duration) (string, error) {
	secret := getJWTSecret()
	if len(secret) == 0 {
		return "", errors.New("JWT_SECRET is not set")
	}

	now := time.Now()
	claims := JWTClaims{
		Sub:  sub,
		Role: role,
		Iat:  now.Unix(),
		Exp:  now.Add(ttl).Unix(),
	}
	h := jwtHeader{Alg: "HS256", Typ: "JWT"}

	hb, err := json.Marshal(h)
	if err != nil {
		return "", err
	}
	pb, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	enc := base64.RawURLEncoding
	headerB64 := enc.EncodeToString(hb)
	payloadB64 := enc.EncodeToString(pb)
	toSign := headerB64 + "." + payloadB64

	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(toSign))
	sig := mac.Sum(nil)
	sigB64 := enc.EncodeToString(sig)

	return toSign + "." + sigB64, nil
}

func verifyJWT(token string) (*JWTClaims, error) {
	secret := getJWTSecret()
	if len(secret) == 0 {
		return nil, errors.New("JWT_SECRET is not set")
	}

	token = strings.TrimSpace(token)
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, errUnauthorized
	}
	headerB64, payloadB64, sigB64 := parts[0], parts[1], parts[2]

	enc := base64.RawURLEncoding
	sigBytes, err := enc.DecodeString(sigB64)
	if err != nil {
		return nil, errUnauthorized
	}

	toSign := headerB64 + "." + payloadB64
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(toSign))
	expectedSig := mac.Sum(nil)
	if subtle.ConstantTimeCompare(expectedSig, sigBytes) != 1 {
		return nil, errUnauthorized
	}

	claimsBytes, err := enc.DecodeString(payloadB64)
	if err != nil {
		return nil, errUnauthorized
	}

	var claims JWTClaims
	if err := json.Unmarshal(claimsBytes, &claims); err != nil {
		return nil, errUnauthorized
	}

	now := time.Now().Unix()
	if claims.Exp < now {
		return nil, errUnauthorized
	}
	return &claims, nil
}

func extractBearerToken(r *http.Request) string {
	// Authorization: Bearer <token>
	auth := r.Header.Get("Authorization")
	if auth == "" {
		// also support token query param for WS/screen deployments
		return strings.TrimSpace(r.URL.Query().Get("token"))
	}
	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 {
		return ""
	}
	if !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func requireAuth(r *http.Request) (*JWTClaims, error) {
	token := extractBearerToken(r)
	if token == "" {
		return nil, errUnauthorized
	}
	claims, err := verifyJWT(token)
	if err != nil {
		return nil, errUnauthorized
	}
	return claims, nil
}

func writeAuthError(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
}

func requireRole(r *http.Request, want string) (*JWTClaims, error) {
	claims, err := requireAuth(r)
	if err != nil {
		return nil, err
	}
	if want != "" && claims.Role != want {
		return nil, ErrForbidden
	}
	return claims, nil
}

