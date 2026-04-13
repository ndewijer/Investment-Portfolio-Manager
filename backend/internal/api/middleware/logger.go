package middleware

import (
	"net"
	"net/http"
	"reflect"
	"runtime"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/logging"
)

var log = logging.NewLogger("system")

// handlerSources maps "METHOD /route/pattern" to the handler function name
// (e.g. "handlers.(*PortfolioHandler).PortfolioHistory"). Populated once at
// startup by InitHandlerSources; read-only after that.
var handlerSources map[string]string

// InitHandlerSources walks the router's route tree to build a lookup from
// route pattern to handler function name. Call once after all routes are
// registered and before serving requests.
func InitHandlerSources(router http.Handler) {
	routes, ok := router.(chi.Routes)
	if !ok {
		return
	}
	m := make(map[string]string)
	if err := chi.Walk(routes, func(method, route string, handler http.Handler, _ ...func(http.Handler) http.Handler) error {
		fn := reflect.ValueOf(handler)
		if fn.Kind() == reflect.Func {
			if fi := runtime.FuncForPC(fn.Pointer()); fi != nil {
				name := fi.Name()
				// "github.com/.../handlers.(*PortfolioHandler).PortfolioHistory-fm"
				// -> "handlers.(*PortfolioHandler).PortfolioHistory"
				parts := strings.Split(name, "/")
				short := parts[len(parts)-1]
				short = strings.TrimSuffix(short, "-fm")
				m[method+" "+route] = short
			}
		}
		return nil
	}); err != nil {
		log.Error("failed to walk routes for handler source map", "error", err)
		return
	}
	handlerSources = m
}

// Logger is a middleware that enriches the request context with logging metadata
// and logs each HTTP request with structured fields.
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Enrich context with request info for downstream slog calls.
		reqID := chimw.GetReqID(r.Context())
		ip := clientIP(r)
		ua := r.Header.Get("User-Agent")
		ctx := logging.WithRequestInfo(r.Context(), reqID, ip, ua)
		r = r.WithContext(ctx)

		// Create a response writer wrapper to capture status code.
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Call the next handler.
		next.ServeHTTP(wrapped, r)

		// Sanitize user-supplied values to prevent log injection: strip CR/LF.
		sanitize := strings.NewReplacer("\n", "", "\r", "").Replace

		// Resolve handler function name from the matched route pattern.
		// Falls back to PC-derived source in the DB handler if not found.
		var source string
		if handlerSources != nil {
			routePattern := chi.RouteContext(r.Context()).RoutePattern()
			source = handlerSources[r.Method+" "+routePattern]
		}

		log.InfoContext(ctx, "http request",
			"source", source,
			"method", sanitize(r.Method),
			"path", sanitize(r.URL.Path),
			"status", wrapped.statusCode,
			"duration", time.Since(start).String(),
		)
	})
}

// clientIP extracts the client IP address from a request.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if i := strings.IndexByte(xff, ','); i != -1 {
			xff = xff[:i]
		}
		return strings.TrimSpace(xff)
	}
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// responseWriter wraps http.ResponseWriter to capture the status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
