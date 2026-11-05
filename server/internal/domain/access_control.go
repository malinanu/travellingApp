package domain

import "time"

// CommandCenterPermission represents a granular permission / action within a feature.
type CommandCenterPermission struct {
	ID          string `json:"id"`
	FeatureID   string `json:"featureId"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Risk        string `json:"risk"` // low | medium | high | critical
}

// CommandCenterFeature represents a major Command Center capability with sub-permissions.
type CommandCenterFeature struct {
	ID             string                    `json:"id"`
	Name           string                    `json:"name"`
	Description    string                    `json:"description"`
	Category       string                    `json:"category"` // operations | security | monetization | editorial | intelligence
	Icon           string                    `json:"icon"`
	SubPermissions []CommandCenterPermission `json:"subPermissions"`
}

// AdminAccessUser represents a Command Center admin operator with their assigned permissions.
type AdminAccessUser struct {
	ID               string     `json:"id"`
	Name             string     `json:"name"`
	Email            string     `json:"email"`
	Role             Role       `json:"role"`
	IsSuperAdmin     bool       `json:"isSuperAdmin"`
	AdminPermissions []string   `json:"adminPermissions"`
	ActiveSessions   int        `json:"activeSessions"`
	LastActiveAt     *time.Time `json:"lastActiveAt,omitempty"`
	CreatedAt        time.Time  `json:"createdAt"`
}
