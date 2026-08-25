package web

import (
	"embed"
	"net/http"
)

//go:embed static/index.html static/app.css static/app.js
var assets embed.FS

func (s *Server) serveAsset(w http.ResponseWriter, name, contentType string) {
	data, err := assets.ReadFile(name)
	if err != nil {
		http.Error(w, "资源不存在", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func (s *Server) HandleWorkspace(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	s.serveAsset(w, "static/index.html", "text/html; charset=utf-8")
}

func (s *Server) HandleCSS(w http.ResponseWriter, _ *http.Request) {
	s.serveAsset(w, "static/app.css", "text/css; charset=utf-8")
}

func (s *Server) HandleJS(w http.ResponseWriter, _ *http.Request) {
	s.serveAsset(w, "static/app.js", "text/javascript; charset=utf-8")
}
