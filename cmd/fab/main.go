package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bobg/errors"

	"github.com/bobg/fab"
	"github.com/bobg/fab/golang"
	"github.com/bobg/fab/proto"
	"github.com/bobg/fab/ts"
)

func main() {
	if err := run(); err != nil {
		fmt.Printf("Error: %s\n", err)
		os.Exit(1)
	}
}

func run() error {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		home, err := os.UserHomeDir()
		if err != nil {
			return errors.Wrap(err, "getting home dir")
		}
		cacheDir = filepath.Join(home, ".cache")
	}

	var (
		fabdir  string
		verbose bool
		list    bool
		force   bool
		dryrun  bool
	)
	flag.StringVar(&fabdir, "fab", filepath.Join(cacheDir, "fab"), "directory containing fab DB and compiled drivers")
	flag.BoolVar(&verbose, "v", false, "run verbosely")
	flag.BoolVar(&list, "list", false, "list available targets")
	flag.BoolVar(&force, "f", false, "force compilation of -bin executable")
	flag.BoolVar(&dryrun, "n", false, "dry run mode")
	flag.Parse()

	db, err := fab.OpenHashDB(fabdir)
	if err != nil {
		return errors.Wrap(err, "opening hash DB")
	}

	con := fab.NewController("", db)
	golang.RegisterDefaults(con)
	proto.RegisterDefaults(con)
	ts.RegisterDefaults(con)

	con.DryRun = dryrun
	con.Force = force
	con.Verbose = verbose

	if err := con.ReadYAMLFile(""); err != nil {
		return errors.Wrap(err, "reading YAML file")
	}

	if list {
		con.ListTargets(os.Stdout)
		return nil
	}

	return con.RunArgs(context.Background(), flag.Args())
}
