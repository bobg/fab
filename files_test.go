package fab

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/bobg/go-generics/v2/set"
)

func TestFileChaining(t *testing.T) {
	t.Parallel()

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

	newController := func() *Controller {
		con := NewController("")
		// These registrations make things clearer in verbose mode.
		if _, err = con.RegisterTarget("AB", "", aToB); err != nil {
			t.Fatal(err)
		}
		if _, err = con.RegisterTarget("BC", "", bToC); err != nil {
			t.Fatal(err)
		}
		return con
	}
	con := newController()

	ctx := context.Background()

	db := memdb(set.New[string]())
	ctx = WithHashDB(ctx, db)
	ctx = WithVerbose(ctx, testing.Verbose())

	if err = con.Run(ctx, bToC); err != nil {
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

	con = newController()

	if err = con.Run(ctx, aToB); err != nil {
		t.Fatal(err)
	}
	info, err = os.Stat(bFile)
	if err != nil {
		t.Fatal(err)
	}
	if !info.ModTime().Equal(bTime) {
		t.Errorf("bTime has changed to %s, should still be %s", info.ModTime(), bTime)
	}

	con = newController()

	if err = con.Run(ctx, bToC); err != nil {
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

	con = newController()

	if err = con.Run(ctx, bToC); err != nil {
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

func TestFileHashes(t *testing.T) {
	t.Parallel()

	got, err := fileHashes([]string{
		"_testdata/filehashes/file2",
		"_testdata/filehashes/dir",
		"_testdata/filehashes/file1",
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"_testdata/filehashes/dir/file3",
		"60f0bc98ed8c6cd61b9124a0d03932ef9a35d483076860882f18a976",
		"_testdata/filehashes/file1",
		"55ad928246b8a22184d245a5966ea69fb4aa57103e835f994bd84457",
		"_testdata/filehashes/file2",
		"16cdf838123b47d4244b7d31efc0b8a17ba299bab0f1ba3d61f33b3c",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
