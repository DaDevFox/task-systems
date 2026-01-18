package oauth2

import (
	"context"
	"github.com/DaDevFox/task-systems/user-core/backend/internal/domain"
	"github.com/DaDevFox/task-systems/user-core/backend/internal/repository"
	"github.com/sirupsen/logrus"
)

type OAuth2Provider interface {
	Name() string
	GetAuthURL(state string) string
	ExchangeCodeForToken(code string) (*OAuth2Token, error)
	GetUserInfo(token string) (*OAuth2UserInfo, error)
	GetScopes() []string
}

type OAuth2Token struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64
	TokenType    string
}

type OAuth2UserInfo struct {
	ID       string
	Email    string
	Name     string
	Avatar   string
	Provider string
}

type OAuth2Config struct {
	GoogleClientID     string
	GoogleClientSecret string
	RedirectURL        string
}

type OAuth2Flow struct {
	config    OAuth2Config
	providers map[string]OAuth2Provider
	userRepo  repository.UserRepository
	logger    *logrus.Logger
}

func NewOAuth2Flow(config OAuth2Config, userRepo repository.UserRepository, logger *logrus.Logger) *OAuth2Flow {
	return &OAuth2Flow{
		config:    config,
		providers: make(map[string]OAuth2Provider),
		userRepo:  userRepo,
		logger:    logger,
	}
}

func (f *OAuth2Flow) RegisterProvider(name string, provider OAuth2Provider) {
	f.providers[name] = provider
}

func (f *OAuth2Flow) GetAuthURL(provider string, state string) (string, error) {
	p, ok := f.providers[provider]
	if !ok {
		return "", domain.ErrInvalidProvider
	}
	return p.GetAuthURL(state), nil
}

func (f *OAuth2Flow) ExchangeCode(provider string, code string) (*OAuth2Token, error) {
	p, ok := f.providers[provider]
	if !ok {
		return nil, domain.ErrInvalidProvider
	}
	return p.ExchangeCodeForToken(code)
}

func (f *OAuth2Flow) LinkAccount(ctx context.Context, provider string, token string) (*domain.User, string, error) {
	p, ok := f.providers[provider]
	if !ok {
		return nil, "", domain.ErrInvalidProvider
	}
	info, err := p.GetUserInfo(token)
	if err != nil {
		return nil, "", err
	}
	// Implement domain link logic, update user repository
	return nil, info.ID, nil // TODO
}
