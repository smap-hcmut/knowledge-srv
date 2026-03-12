package model

const (
	RoleAdmin   = "ADMIN"
	RoleAnalyst = "ANALYST"
	RoleViewer  = "VIEWER"
)

type Scope struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"` // ADMIN, ANALYST, or VIEWER
	JTI      string `json:"jti"`
}

// IsAdmin checks if the scope has admin role
func (s Scope) IsAdmin() bool {
	return s.Role == RoleAdmin
}

// IsAnalyst checks if the scope has analyst role
func (s Scope) IsAnalyst() bool {
	return s.Role == RoleAnalyst
}

// IsViewer checks if the scope has viewer role
func (s Scope) IsViewer() bool {
	return s.Role == RoleViewer
}

func ToScope(sc interface{}) Scope {
	if s, ok := sc.(interface {
		GetUserID() string
		GetUsername() string
		GetRole() string
		GetJTI() string
	}); ok {
		return Scope{
			UserID:   s.GetUserID(),
			Username: s.GetUsername(),
			Role:     s.GetRole(),
			JTI:      s.GetJTI(),
		}
	}

	// Fallback for direct struct conversion
	if s, ok := sc.(struct {
		UserID   string
		Username string
		Role     string
		JTI      string
	}); ok {
		return Scope{
			UserID:   s.UserID,
			Username: s.Username,
			Role:     s.Role,
			JTI:      s.JTI,
		}
	}

	return Scope{}
}
