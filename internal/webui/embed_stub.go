//go:build !webui

package webui

import "io/fs"

// FS is nil when the binary is built without -tags webui.
// The web server flag will be unavailable in this mode.
var FS fs.FS
