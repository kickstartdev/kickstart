package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/kickstartdev/kickstart/internal/debug"
)

// Set via ldflags at build time, falls back to env var
var GitHubClientID string

type DeviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	Interval        int    `json:"interval"`
}

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
	Error       string `json:"error"`
}

func getClientID() string {
	if GitHubClientID != "" {
		return GitHubClientID
	}
	godotenv.Load()
	if id := os.Getenv("GH_CLIENT_ID"); id != "" {
		return id
	}
	return os.Getenv("GITHUB_CLIENT_ID")
}

func RequestDeviceCode() (*DeviceCodeResponse, error) {
	clientID := getClientID()
	debug.Log("GITHUB_CLIENT_ID=%q", clientID)

	if clientID == "" {
		return nil, fmt.Errorf("GITHUB_CLIENT_ID is not set")
	}

	body, _ := json.Marshal(map[string]string{
		"client_id": clientID,
		"scope":     "repo read:org workflow",
	})

	req, _ := http.NewRequest("POST", "https://github.com/login/device/code", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	debug.Log("requesting device code from GitHub...")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		debug.Log("HTTP error: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	debug.Log("response status=%d body=%s", resp.StatusCode, string(respBody))

	if resp.StatusCode != 200 {
		var errResp struct {
			Error       string `json:"error"`
			Description string `json:"error_description"`
		}
		json.Unmarshal(respBody, &errResp)
		if errResp.Description != "" {
			return nil, fmt.Errorf("%s", strings.ToLower(errResp.Description))
		}
		return nil, fmt.Errorf("GitHub returned status %d", resp.StatusCode)
	}

	var result DeviceCodeResponse
	json.Unmarshal(respBody, &result)

	if result.DeviceCode == "" {
		return nil, fmt.Errorf("empty device code in response: %s", string(respBody))
	}

	debug.Log("got device code, user_code=%s uri=%s", result.UserCode, result.VerificationURI)
	return &result, nil
}

func PollForToken(deviceCode string, interval int) (string, error) {
	clientID := getClientID()
	debug.Log("polling for token, interval=%d", interval)

	for {
		time.Sleep(time.Duration(interval) * time.Second)

		body, _ := json.Marshal(map[string]string{
			"client_id":   clientID,
			"device_code": deviceCode,
			"grant_type":  "urn:ietf:params:oauth:grant-type:device_code",
		})

		req, _ := http.NewRequest("POST", "https://github.com/login/oauth/access_token", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			debug.Log("poll HTTP error: %v", err)
			return "", err
		}

		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		debug.Log("poll response status=%d body=%s", resp.StatusCode, string(respBody))

		var result TokenResponse
		json.Unmarshal(respBody, &result)

		if result.AccessToken != "" {
			debug.Log("got access token")
			return result.AccessToken, nil
		}

		if result.Error == "expired_token" {
			return "", fmt.Errorf("code expired, try again")
		}

		if result.Error == "slow_down" {
			interval += 5
		}

		debug.Log("poll status: %s", result.Error)
	}
}

func GetUsername(token string) (string, error) {
	req, _ := http.NewRequest("GET", "https://api.github.com/user", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		debug.Log("username HTTP error: %v", err)
		return "", err
	}
	defer resp.Body.Close()

	var user struct {
		Login string `json:"login"`
	}

	json.NewDecoder(resp.Body).Decode(&user)
	debug.Log("got username=%s", user.Login)
	return user.Login, nil
}
