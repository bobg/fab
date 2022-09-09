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
	fab.Register("test", &fab.Command{
		Shell:  "go test -race -cover ./...",
		Stdout: os.Stdout,
	})
	fab.Register("lint", &fab.Command{
		Shell:  "staticcheck ./...",
		Stdout: os.Stdout,
	})
	fab.Register("vet", &fab.Command{
		Shell:  "go vet ./...",
		Stdout: os.Stdout,
	})
	fab.Register("check", fab.All(Vet, Lint, Test))
}
