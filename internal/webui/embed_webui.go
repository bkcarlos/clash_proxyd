//go:build webui

package webui

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var distFS embed.FS

// FS is the embedded web UI filesystem (sub-rooted at dist/).
// It is non-nil only when the binary is built with -tags webui.
var FS fs.FS

func init() {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		panic("webui: failed to sub dist FS: " + err.Error())
	}
	FS = sub
}
