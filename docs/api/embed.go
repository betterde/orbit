package api

import (
	"embed"
	"github.com/betterde/orbit/internal/journal"
	"io/fs"
	"net/http"
)

//go:embed orbit/user.swagger.json
var FS embed.FS

func Serve() http.FileSystem {
	dist, err := fs.Sub(FS, "orbit")
	if err != nil {
		journal.Logger.Panicw("Error mounting front-end static resources!", err)
	}

	return http.FS(dist)
}
