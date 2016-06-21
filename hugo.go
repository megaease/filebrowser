//go:generate go get github.com/jteeuwen/go-bindata
//go:generate go install github.com/jteeuwen/go-bindata/go-bindata
//go:generate go-bindata -debug -pkg hugo -prefix "assets" -o binary.go assets/...

// Package hugo makes the bridge between the static website generator Hugo
// and the webserver Caddy, also providing an administrative user interface.
package hugo

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/hacdias/caddy-filemanager"
	"github.com/mholt/caddy/caddyhttp/httpserver"
)

// Hugo contais the next middleware to be run and the configuration
// of the current one.
type Hugo struct {
	FileManager *filemanager.FileManager
	Next        httpserver.Handler
	Config      *Config
}

// ServeHTTP is the main function of the whole plugin that routes every single
// request to its function.
func (h Hugo) ServeHTTP(w http.ResponseWriter, r *http.Request) (int, error) {
	// Check if the current request if for this plugin
	if httpserver.Path(r.URL.Path).Matches(h.Config.BaseURL) {
		// If the url matches exactly with /{admin}/settings/, redirect
		// to the page of the configuration file
		if r.URL.Path == h.Config.BaseURL+"/settings/" {
			var frontmatter string

			if _, err := os.Stat(h.Config.Root + "config.yaml"); err == nil {
				frontmatter = "yaml"
			}

			if _, err := os.Stat(h.Config.Root + "config.json"); err == nil {
				frontmatter = "json"
			}

			if _, err := os.Stat(h.Config.Root + "config.toml"); err == nil {
				frontmatter = "toml"
			}

			http.Redirect(w, r, h.Config.BaseURL+"/config."+frontmatter, http.StatusTemporaryRedirect)
			return 0, nil
		}

		if strings.HasPrefix(r.URL.Path, h.Config.BaseURL+"/api/git/") && r.Method == http.MethodPost {
			//return HandleGit(w, r, h.Config)
			return 0, nil
		}

		if h.ShouldHandle(r) {
			filename := strings.Replace(r.URL.Path, h.Config.BaseURL, h.Config.Root, 1)
			switch r.Method {
			case http.MethodGet:
				return h.GET(w, r, filename)
			case http.MethodPost:
				return h.POST(w, r, filename)
			default:
				return h.FileManager.ServeHTTP(w, r)
			}
		}

		return h.FileManager.ServeHTTP(w, r)
	}
	return h.Next.ServeHTTP(w, r)
}

var extensions = []string{
	"md", "markdown", "mdown", "mmark",
	"asciidoc", "adoc", "ad",
	"rst",
	"html", "htm",
	"js",
	"toml", "yaml", "json",
}

// ShouldHandle checks if this extension should be handled by this plugin
func (h Hugo) ShouldHandle(r *http.Request) bool {
	// Checks if the method is get or post
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		return false
	}

	// Check if this request is for FileManager assets
	if httpserver.Path(r.URL.Path).Matches(h.Config.BaseURL + filemanager.AssetsURL) {
		return false
	}

	// If this request requires a raw file or a download, return the FileManager
	query := r.URL.Query()
	if val, ok := query["raw"]; ok && val[0] == "true" {
		return false
	}

	if val, ok := query["download"]; ok && val[0] == "true" {
		return false
	}

	// Check by file extension
	extension := strings.TrimPrefix(filepath.Ext(r.URL.Path), ".")

	for _, ext := range extensions {
		if ext == extension {
			return true
		}
	}

	return false
}
