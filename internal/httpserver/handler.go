package httpserver

import (
	"context"
	"knowledge-srv/internal/model"

	"github.com/smap-hcmut/shared-libs/go/middleware"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func (srv HTTPServer) mapHandlers() error {
	mw := middleware.New(middleware.Config{
		JWTManager:       srv.jwtManager,
		CookieName:       srv.cookieConfig.Name,
		ProductionDomain: srv.cookieConfig.Domain,
		InternalKey:      srv.config.InternalConfig.InternalKey,
	})

	srv.registerMiddlewares()
	srv.registerSystemRoutes()

	if err := srv.registerDomainRoutes(mw); err != nil {
		return err
	}

	return nil
}

func (srv HTTPServer) registerMiddlewares() {
	srv.gin.Use(middleware.Tracing())
	srv.gin.Use(middleware.Recovery(srv.l, srv.discord))

	corsConfig := middleware.DefaultCORSConfig(srv.environment)
	srv.gin.Use(middleware.CORS(corsConfig))

	// Log CORS mode for visibility
	ctx := context.Background()
	if srv.environment == string(model.EnvironmentProduction) {
		srv.l.Infof(ctx, "CORS mode: production")
	} else {
		srv.l.Infof(ctx, "CORS mode: %s", srv.environment)
	}

	// Add locale middleware to extract and set locale from request header
	srv.gin.Use(middleware.Locale())
}

func (srv HTTPServer) registerSystemRoutes() {
	srv.gin.GET("/health", srv.healthCheck)
	srv.gin.GET("/ready", srv.readyCheck)
	srv.gin.GET("/live", srv.liveCheck)

	// Swagger UI and docs
	srv.gin.GET("/swagger/*any", ginSwagger.WrapHandler(
		swaggerFiles.Handler,
		ginSwagger.URL("doc.json"), // Use relative path
		ginSwagger.DefaultModelsExpandDepth(-1),
	))
}

// registerDomainRoutes initializes and registers all domain routes
func (srv HTTPServer) registerDomainRoutes(mw *middleware.Middleware) error {
	ctx := context.Background()

	// Base route group (api/v1/knowledge)
	api := srv.gin.Group(model.APIV1Prefix + "/knowledge")

	// Setup core domains first
	if err := srv.setupCoreDomains(ctx); err != nil {
		return err
	}

	// Setup indexing domain
	if err := srv.setupIndexingDomain(ctx, api, mw); err != nil {
		return err
	}

	// Setup search domain
	if err := srv.setupSearchDomain(ctx, api, mw); err != nil {
		return err
	}

	// Setup chat domain (depends on searchUC from search domain)
	if err := srv.setupChatDomain(ctx, api, mw); err != nil {
		return err
	}

	// Setup report domain (depends on searchUC, geminiClient, minioClient)
	if err := srv.setupReportDomain(ctx, api, mw); err != nil {
		return err
	}

	return nil
}
