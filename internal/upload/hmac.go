package upload

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"io"
	"net/http"
	"time"
)

// hmacCredentials signs outbound requests using BloodHound's HMAC signature
// scheme. Ported from github.com/SpecterOps/bloodhound-go-sdk/sdk.HMACCredentials
// so this repo doesn't need the SDK's openapi-codegen dependency chain for a
// single request-signing struct.
type hmacCredentials struct {
	tokenKey string
	tokenID  string
}

func newHMACCredentials(tokenKey, tokenID string) *hmacCredentials {
	return &hmacCredentials{tokenKey: tokenKey, tokenID: tokenID}
}

// intercept signs req in place: HMAC-SHA256 chained over method+URI, then the
// hour-truncated RFC3339 timestamp, then the body (in that order, each link
// keyed by the previous digest). Matches the original SDK's signature format
// byte-for-byte so BloodHound's API continues to accept these requests.
func (c *hmacCredentials) intercept(ctx context.Context, req *http.Request) error {
	digester := hmac.New(sha256.New, []byte(c.tokenKey))
	digester.Write([]byte(req.Method + req.URL.RequestURI()))

	digester = hmac.New(sha256.New, digester.Sum(nil))
	datetimeFormatted := time.Now().UTC().Format(time.RFC3339)
	digester.Write([]byte(datetimeFormatted[:13]))

	digester = hmac.New(sha256.New, digester.Sum(nil))
	if req.Body != nil {
		bodyBytes, err := io.ReadAll(req.Body)
		if err != nil {
			return err
		}
		req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		digester.Write(bodyBytes)
	}

	req.Header.Set("User-Agent", "bhe-go-sdk 0001")
	req.Header.Set("Authorization", "bhesignature "+c.tokenID)
	req.Header.Set("RequestDate", datetimeFormatted)
	req.Header.Set("Signature", base64.StdEncoding.EncodeToString(digester.Sum(nil)))

	return nil
}
