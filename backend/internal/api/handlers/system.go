package handlers

import (
	"net/http"

	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/api/response"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/apperrors"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/logging"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/service"
)

var sysLog = logging.NewLogger("system")

// SystemHandler handles system-related HTTP requests
type SystemHandler struct {
	systemService *service.SystemService
}

// NewSystemHandler creates a new SystemHandler
func NewSystemHandler(systemService *service.SystemService) *SystemHandler {
	return &SystemHandler{
		systemService: systemService,
	}
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status   string `json:"status"`
	Database string `json:"database"`
	Error    string `json:"error,omitempty"`
}

// Health checks the health of the system and database connectivity
func (h *SystemHandler) Health(w http.ResponseWriter, r *http.Request) {
	sysLog.DebugContext(r.Context(), "health check request")

	// Check database health
	if err := h.systemService.CheckHealth(); err != nil {
		sysLog.ErrorContext(r.Context(), "health check failed", "error", err)
		health := HealthResponse{
			Status:   "unhealthy",
			Database: "disconnected",
		}
		response.RespondError(w, http.StatusServiceUnavailable, "unhealthy", health)
		return
	}

	// System is healthy
	health := HealthResponse{
		Status:   "healthy",
		Database: "connected",
	}
	response.RespondJSON(w, http.StatusOK, health)
}

// VersionInfoResponse represents the version check response containing application
// and database version information, feature availability, and migration status.
type VersionInfoResponse struct {
	AppVersion       string          `json:"app_version"`
	DbVersion        string          `json:"db_version"`
	Features         map[string]bool `json:"features"`
	MigrationNeeded  bool            `json:"migration_needed"`
	MigrationMessage *string         `json:"migration_message"`
}

// Version handles GET requests to retrieve version information and feature availability.
// Returns the application version, database version, available features, and any pending migrations.
//
// Endpoint: GET /api/system/version
// Response: 200 OK with VersionInfoResponse
// Error: 500 Internal Server Error if version check fails
func (h *SystemHandler) Version(w http.ResponseWriter, r *http.Request) {
	sysLog.DebugContext(r.Context(), "version check request")

	version, err := h.systemService.CheckVersion()
	if err != nil {
		sysLog.ErrorContext(r.Context(), "failed to get version info", "error", err)
		response.RespondInternalError(w, r, apperrors.ErrFailedToGetVersionInfo.Error())
		return
	}

	versionResponse := VersionInfoResponse{
		AppVersion:       version.AppVersion,
		DbVersion:        version.DbVersion,
		Features:         version.Features,
		MigrationNeeded:  version.MigrationNeeded,
		MigrationMessage: version.MigrationMessage,
	}

	response.RespondJSON(w, http.StatusOK, versionResponse)
}
