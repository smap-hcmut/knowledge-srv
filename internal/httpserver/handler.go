package httpserver

import (
	"context"

	"knowledge-srv/internal/middleware"
	"knowledge-srv/internal/model"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func (srv HTTPServer) mapHandlers() error {
	mw := middleware.New(srv.l, srv.jwtManager, srv.cookieConfig, srv.config.InternalConfig.InternalKey, srv.config, srv.encrypter)

	srv.registerMiddlewares(mw)
	srv.registerSystemRoutes()

	return nil
}

func (srv HTTPServer) registerMiddlewares(mw middleware.Middleware) {
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
	srv.gin.Use(mw.Locale())
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
