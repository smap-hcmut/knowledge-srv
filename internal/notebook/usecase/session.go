package usecase

import (
	"context"

	"knowledge-srv/internal/model"
	"knowledge-srv/pkg/maestro"
)

// StartSessionLoop ensures a session is active and starts a health check loop
func (uc *implUseCase) StartSessionLoop(ctx context.Context, sc model.Scope) error {
	uc.mu.Lock()
	defer uc.mu.Unlock()

	if uc.activeSession != "" {
		uc.l.Info(ctx, "Session already active")
		return nil
	}

	session, err := uc.maestro.CreateSession(ctx, maestro.CreateSessionReq{})
	if err != nil {
		uc.l.Errorf(ctx, "Failed to create session: %v", err)
		return err
	}

	uc.activeSession = session.SessionID
	uc.l.Infof(ctx, "Session created: %s", uc.activeSession)

	go uc.healthCheckLoop(context.Background())

	return nil
}

// StopSessionLoop stops the active session and terminates the health check loop
func (uc *implUseCase) StopSessionLoop(ctx context.Context, sc model.Scope) error {
	uc.mu.Lock()
	defer uc.mu.Unlock()

	if uc.activeSession == "" {
		uc.l.Info(ctx, "No active session to stop")
		return nil
	}

	err := uc.maestro.DeleteSession(ctx, uc.activeSession)
	if err != nil {
		uc.l.Errorf(ctx, "Failed to delete session: %v", err)
		return err
	}

	uc.l.Infof(ctx, "Session stopped: %s", uc.activeSession)
	uc.activeSession = ""

	return nil
}

// healthCheckLoop periodically checks the health of the active session
func (uc *implUseCase) healthCheckLoop(ctx context.Context) {
	uc.l.Info(ctx, "Starting session health check loop")
	for {
		select {
		case <-ctx.Done():
			uc.l.Info(ctx, "Health check loop terminated")
			return
		default:
			uc.mu.RLock()
			sessionID := uc.activeSession
			uc.mu.RUnlock()

			if sessionID == "" {
				uc.l.Warn(ctx, "No active session for health check")
				return
			}

			_, err := uc.maestro.GetSession(ctx, sessionID)
			if err != nil {
				uc.l.Errorf(ctx, "Session health check failed: %v", err)
				uc.mu.Lock()
				uc.activeSession = ""
				uc.mu.Unlock()
				return
			}

			uc.l.Infof(ctx, "Session %s is healthy", sessionID)
		}
	}
}
