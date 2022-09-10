package build

import (
	"os"

	"github.com/bobg/fab"
)

func init() {
	fab.Register("build", &fab.Command{
		Shell:  "go build ./...",
		Stdout: os.Stdout,
	})

	var (
		test = &fab.Command{
			Shell:  "go test -race -cover ./...",
			Stdout: os.Stdout,
		}
		lint = &fab.Command{
			Shell:  "staticcheck ./...",
			Stdout: os.Stdout,
		}
		vet = &fab.Command{
			Shell:  "go vet ./...",
			Stdout: os.Stdout,
		}
	)

	fab.Register("test", test)
	fab.Register("lint", lint)
	fab.Register("vet", vet)
	fab.Register("check", fab.All(vet, lint, test))
}
