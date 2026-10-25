package httpapi

import (
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"clyra/backend/internal/problem"
)

func (a *API) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/setup/status", a.setupStatus)
	mux.HandleFunc("POST /api/setup", a.setup)
	mux.HandleFunc("POST /api/auth/login", a.login)
	mux.HandleFunc("GET /api/_auth/session", a.getSession)
	mux.HandleFunc("DELETE /api/_auth/session", a.logout)
	for route, handler := range map[string]http.HandlerFunc{
		"PATCH /api/auth/profile": a.updateProfile, "PATCH /api/auth/password": a.updatePassword,
		"GET /api/subscriptions": a.listSubscriptions, "POST /api/subscriptions": a.createSubscription,
		"PATCH /api/subscriptions/{id}": a.patchSubscription, "DELETE /api/subscriptions/{id}": a.deleteSubscription,
		"POST /api/subscriptions/{id}/renew": a.renewSubscription,
		"GET /api/settings":                  a.getSettings, "PATCH /api/settings": a.patchSettings,
		"GET /api/notifications/runs": a.listNotificationRuns, "POST /api/notifications/test": a.testNotification,
		"POST /api/icons/search": a.searchIcons,
	} {
		mux.HandleFunc(route, a.requireAdmin(handler))
	}
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) { writeError(w, problem.New(404, "接口不存在")) })
	mux.Handle("/", a.staticHandler())
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		if strings.HasPrefix(r.URL.Path, "/api/") {
			w.Header().Set("Cache-Control", "no-store")
			if r.Method != "GET" && r.Method != "HEAD" && r.Method != "OPTIONS" && r.Header.Get("Origin") != a.config.AppOrigin {
				writeError(w, problem.New(403, "请求来源无效"))
				return
			}
		}
		defer func() {
			if recovered := recover(); recovered != nil {
				log.Print("request failed unexpectedly")
				writeError(w, problem.New(500, "服务器暂时无法处理请求"))
			}
		}()
		mux.ServeHTTP(w, r)
	})
}
func (a *API) staticHandler() http.Handler {
	files := http.FileServer(http.Dir(a.config.StaticDir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" && r.Method != "HEAD" {
			w.WriteHeader(405)
			return
		}
		if r.URL.Path == "/" || r.URL.Path == "/login" || r.URL.Path == "/setup" {
			w.Header().Set("Cache-Control", "no-cache")
			http.ServeFile(w, r, filepath.Join(a.config.StaticDir, "index.html"))
			return
		}
		// Serve build assets only; unknown routes must not turn API mistakes into an HTML success.
		files.ServeHTTP(w, r)
	})
}
