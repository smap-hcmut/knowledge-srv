package middleware

import (
	"knowledge-srv/config"
	"knowledge-srv/pkg/encrypter"
	"knowledge-srv/pkg/log"
	"knowledge-srv/pkg/scope"
)

type Middleware struct {
	l            log.Logger
	jwtManager   scope.Manager
	cookieConfig config.CookieConfig
	internalKey  string
	config       *config.Config
	encrypter    encrypter.Encrypter
}

func New(l log.Logger, jwtManager scope.Manager, cookieConfig config.CookieConfig, internalKey string, cfg *config.Config, enc encrypter.Encrypter) Middleware {
	return Middleware{
		l:            l,
		jwtManager:   jwtManager,
		cookieConfig: cookieConfig,
		internalKey:  internalKey,
		config:       cfg,
		encrypter:    enc,
	}
}
