package credential

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// RefreshOAuth2Token calls the token endpoint with a refresh token to get a new access token.
func RefreshOAuth2Token(tokenUrl, clientId, clientSecret, refreshToken string) (map[string]any, error) {
	data := url.Values{}
	data.Set("client_id", clientId)
	data.Set("client_secret", clientSecret)
	data.Set("refresh_token", refreshToken)
	data.Set("grant_type", "refresh_token")

	resp, err := http.PostForm(tokenUrl, data)
	if err != nil {
		return nil, fmt.Errorf("POST %s: %w", tokenUrl, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("token refresh failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return result, nil
}
