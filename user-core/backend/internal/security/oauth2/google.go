package oauth2

import (
	"context"
	"net/url"
	"strings"
)

type GoogleProvider struct {
	clientID     string
	clientSecret string
	redirectURL  string
}

func NewGoogleProvider(clientID, clientSecret, redirectURL string) *GoogleProvider {
	return &GoogleProvider{
		clientID:     clientID,
		clientSecret: clientSecret,
		redirectURL:  redirectURL,
	}
}

func (g *GoogleProvider) Name() string {
	return "google"
}

func (g *GoogleProvider) GetScopes() []string {
	return []string{"openid", "email", "profile"}
}

func (g *GoogleProvider) GetAuthURL(state string) string {
	v := url.Values{}
	v.Set("client_id", g.clientID)
	v.Set("redirect_uri", g.redirectURL)
	v.Set("response_type", "code")
	v.Set("scope", strings.Join(g.GetScopes(), " "))
	v.Set("state", state)
	v.Set("access_type", "offline")
	return "https://accounts.google.com/o/oauth2/v2/auth?" + v.Encode()
}

func (g *GoogleProvider) ExchangeCodeForToken(code string) (*OAuth2Token, error) {
	// TODO: implement HTTP POST to Google's token endpoint
	return nil, nil
}

func (g *GoogleProvider) GetUserInfo(token string) (*OAuth2UserInfo, error) {
	// TODO: implement HTTP GET to Google's userinfo endpoint
	return nil, nil
}
