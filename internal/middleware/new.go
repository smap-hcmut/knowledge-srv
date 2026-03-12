package middleware

import (
	"knowledge-srv/config"

	"github.com/smap-hcmut/shared-libs/go/auth"
	"github.com/smap-hcmut/shared-libs/go/encrypter"
	"github.com/smap-hcmut/shared-libs/go/log"
)

type Middleware struct {
	l            log.Logger
	jwtManager   auth.Manager
	cookieConfig config.CookieConfig
	internalKey  string
	config       *config.Config
	encrypter    encrypter.Encrypter
}

func New(l log.Logger, jwtManager auth.Manager, cookieConfig config.CookieConfig, internalKey string, cfg *config.Config, enc encrypter.Encrypter) Middleware {
	return Middleware{
		l:            l,
		jwtManager:   jwtManager,
		cookieConfig: cookieConfig,
		internalKey:  internalKey,
		config:       cfg,
		encrypter:    enc,
	}
}
