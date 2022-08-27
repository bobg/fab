package fab

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

//go:embed *.go go.* sqlite/*.go driver.go.tmpl
var embeds embed.FS

var driverStr string

func init() {
	driverBytes, err := embeds.ReadFile("driver.go.tmpl")
	if err != nil {
		panic(err)
	}
	driverStr = string(driverBytes)
}

func populateFabDir(dir string) error {
	return populateFabSubdir(dir, ".")
}

func populateFabSubdir(destdir, subdir string) error {
	fmt.Printf("xxx populateFabSubdir(%s, %s)\n", destdir, subdir)

	if err := os.MkdirAll(destdir, 0755); err != nil {
		return errors.Wrapf(err, "creating %s", destdir)
	}
	entries, err := embeds.ReadDir(subdir)
	if err != nil {
		return errors.Wrap(err, "reading embeds")
	}
	for _, entry := range entries {
		if entry.IsDir() {
			err = populateFabSubdir(filepath.Join(destdir, entry.Name()), filepath.Join(subdir, entry.Name()))
			if err != nil {
				return errors.Wrapf(err, "populating dir %s", entry.Name())
			}
			continue
		}
		if strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		contents, err := embeds.ReadFile(filepath.Join(subdir, entry.Name()))
		if err != nil {
			return errors.Wrapf(err, "reading embedded file %s", entry.Name())
		}
		dest := filepath.Join(destdir, entry.Name())
		err = os.WriteFile(dest, contents, 0644)
		if err != nil {
			return errors.Wrapf(err, "writing %s", dest)
		}
	}
	return nil
}
