package util

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	log "github.com/golang/glog"
	"golang.org/x/oauth2"
)

func GetOauthClient(ctx context.Context, config *oauth2.Config, tokenFile string) (*http.Client, error) {
	// The file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
	tok, err := OauthTokenFromFile(tokenFile)
	if err != nil {
		log.Errorf("getting oauth token from file: %s", err)

		tok, err = GetOauthTokenFromWeb(ctx, config)
		if err != nil {
			return nil, fmt.Errorf("getting oauth token from web: %w", err)
		}

		err = SaveOauthToken(tokenFile, tok)
		if err != nil {
			log.Errorf("saving oauth token: %s", err)
		}
	}
	return config.Client(ctx, tok), nil
}

// GetOauthTokenFromWeb retrieves a token from the web, then returns the retrieved token.
func GetOauthTokenFromWeb(ctx context.Context, config *oauth2.Config) (*oauth2.Token, error) {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		return nil, fmt.Errorf("reading auth code: %w", err)
	}

	tok, err := config.Exchange(ctx, authCode)
	if err != nil {
		return nil, fmt.Errorf("exchanging auth code: %w", err)
	}

	return tok, nil
}

// OauthTokenFromFile retrieves a token from a local file.
func OauthTokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, fmt.Errorf("opening file %s: %w", file, err)
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	if err != nil {
		return nil, fmt.Errorf("decoding token: %w", err)
	}

	return tok, nil
}

// SaveOauthToken saves a token to a file path.
func SaveOauthToken(path string, token *oauth2.Token) error {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("opening file to save %s: %w", path, err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)

	return nil
}
