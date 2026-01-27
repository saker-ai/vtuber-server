package webassets

import (
	"embed"
	"fmt"
	"io/fs"
	"path"
)

//go:embed vtuber live2d-models backgrounds model_dict.json
var embeddedDist embed.FS

// Subdir executes the subdir function.
func Subdir(dir string) (fs.FS, error) {
	cleanDir := path.Clean(dir)
	if cleanDir == "." || cleanDir == "" {
		return embeddedDist, nil
	}

	sub, err := fs.Sub(embeddedDist, cleanDir)
	if err != nil {
		return nil, fmt.Errorf("open embedded dist subdir %q: %w", cleanDir, err)
	}
	return sub, nil
}
