package server

import "io/fs"

type Embed struct {
	Templates fs.FS
	Static    fs.FS
	Resources fs.FS
}
