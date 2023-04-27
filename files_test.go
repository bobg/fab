package fab

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/bobg/go-generics/v2/set"
)

func TestFileChaining(t *testing.T) {
	tmpdir, err := os.MkdirTemp("", "fab")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	var (
		aFile = filepath.Join(tmpdir, "a")
		bFile = filepath.Join(tmpdir, "b")
		cFile = filepath.Join(tmpdir, "c")
	)

	if err = os.WriteFile(aFile, []byte("Aardvark"), 0644); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(aFile)
	if err != nil {
		t.Fatal(err)
	}
	aTime := info.ModTime()

	aToB := fileCopyTarget(aFile, bFile)
	bToC := fileCopyTarget(bFile, cFile)

	RegisterTarget("aToB", "", aToB)
	RegisterTarget("bToC", "", bToC)

	ctx := context.Background()

	db := memdb(set.New[string]())
	ctx = WithHashDB(ctx, db)
	ctx = WithVerbose(ctx, testing.Verbose())

	runner := NewRunner()

	if err = runner.Run(ctx, bToC); err != nil {
		t.Fatal(err)
	}

	info, err = os.Stat(bFile)
	if err != nil {
		t.Fatal(err)
	}
	bTime := info.ModTime()
	if !bTime.After(aTime) {
		t.Errorf("aTime %s is later than bTime %s, should be earlier", aTime, bTime)
	}

	info, err = os.Stat(cFile)
	if err != nil {
		t.Fatal(err)
	}
	cTime := info.ModTime()
	if !cTime.After(bTime) {
		t.Errorf("bTime %s is later than cTime %s, should be earlier", bTime, cTime)
	}

	runner = NewRunner()

	if err = runner.Run(ctx, aToB); err != nil {
		t.Fatal(err)
	}
	info, err = os.Stat(bFile)
	if err != nil {
		t.Fatal(err)
	}
	if !info.ModTime().Equal(bTime) {
		t.Errorf("bTime has changed to %s, should still be %s", info.ModTime(), bTime)
	}

	runner = NewRunner()

	if err = runner.Run(ctx, bToC); err != nil {
		t.Fatal(err)
	}
	info, err = os.Stat(cFile)
	if err != nil {
		t.Fatal(err)
	}
	if !info.ModTime().Equal(cTime) {
		t.Errorf("cTime has changed to %s, should still be %s", info.ModTime(), cTime)
	}

	if err = os.WriteFile(aFile, []byte("Anteater"), 0644); err != nil {
		t.Fatal(err)
	}

	info, err = os.Stat(aFile)
	if err != nil {
		t.Fatal(err)
	}
	aTime = info.ModTime()

	runner = NewRunner()

	if err = runner.Run(ctx, bToC); err != nil {
		t.Fatal(err)
	}

	info, err = os.Stat(bFile)
	if err != nil {
		t.Fatal(err)
	}
	bTime = info.ModTime()
	if !bTime.After(aTime) {
		t.Errorf("aTime %s is later than bTime %s, should be earlier", aTime, bTime)
	}

	info, err = os.Stat(cFile)
	if err != nil {
		t.Fatal(err)
	}
	cTime = info.ModTime()
	if !cTime.After(bTime) {
		t.Errorf("bTime %s is later than cTime %s, should be earlier", bTime, cTime)
	}
}

func fileCopyTarget(from, to string) Target {
	return Files(
		Shellf("sleep 1; cp %s %s", from, to),
		[]string{from},
		[]string{to},
	)
}
