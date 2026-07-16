package webui

import (
	"embed"
	"net/http"
)

//go:embed static/index.html static/assets/app.css static/assets/app.js
var staticFiles embed.FS

type Handler struct{}

func NewHandler() http.Handler {
	return Handler{}
}

func (Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	assetPath, contentType, ok := assetForPath(r.URL.Path)
	if !ok {
		http.NotFound(w, r)
		return
	}

	content, err := staticFiles.ReadFile(assetPath)
	if err != nil {
		http.Error(w, "web asset unavailable", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(content)
}

func assetForPath(path string) (string, string, bool) {
	switch path {
	case "/":
		return "static/index.html", "text/html; charset=utf-8", true
	case "/assets/app.css":
		return "static/assets/app.css", "text/css; charset=utf-8", true
	case "/assets/app.js":
		return "static/assets/app.js", "text/javascript; charset=utf-8", true
	default:
		return "", "", false
	}
}
