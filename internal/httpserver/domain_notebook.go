package httpserver

import (
	"context"

	notebookHTTP "knowledge-srv/internal/notebook/delivery/http"
	notebookPostgre "knowledge-srv/internal/notebook/repository/postgre"
	notebookUsecase "knowledge-srv/internal/notebook/usecase"

	"github.com/gin-gonic/gin"
)

// setupNotebookDomain initializes the components related to the notebook domain.
func (srv *HTTPServer) setupNotebookDomain(ctx context.Context, r *gin.RouterGroup) error {
	chatJobRepo := notebookPostgre.NewChatJobRepo(srv.postgresDB)
	campaignRepo := notebookPostgre.NewCampaignRepo(srv.postgresDB)
	sourceRepo := notebookPostgre.NewSourceRepo(srv.postgresDB)
	sessionRepo := notebookPostgre.NewSessionRepo()

	cfg := notebookUsecase.Config{
		NotebookEnabled:    srv.config.Notebook.Enabled,
		JobPollIntervalMs:  srv.config.Maestro.JobPollIntervalMs,
		JobPollMaxAttempts: srv.config.Maestro.JobPollMaxAttempts,
		SyncMaxRetries:     srv.config.Notebook.SyncMaxRetries,
		WebhookCallbackURL: srv.config.Maestro.WebhookCallbackURL,
		WebhookSecret:      srv.config.Maestro.WebhookSecret,
	}

	uc := notebookUsecase.NewUseCase(
		srv.maestroClient,
		campaignRepo,
		sourceRepo,
		sessionRepo,
		chatJobRepo,
		cfg,
		srv.l,
	)

	// Setup HTTP Handlers representing the webhooks
	handler := notebookHTTP.New(srv.l, uc, cfg.WebhookSecret)
	
	// Register Routes under /internal
	handler.RegisterRoutes(r)

	srv.notebookUC = uc
	srv.l.Infof(ctx, "Notebook domain registered")
	return nil
}
