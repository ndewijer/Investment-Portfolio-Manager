package model

// VersionInfo contains version and feature information for the application.
type VersionInfo struct {
	AppVersion       string          `json:"app_version"`
	DbVersion        string          `json:"db_version"`
	Features         map[string]bool `json:"features"`
	MigrationNeeded  bool            `json:"migration_needed"`
	MigrationMessage *string         `json:"migration_message,omitempty"`
}
