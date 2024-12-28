package fab

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/otiai10/copy"
)

func TestTopDir(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		dir     string // A pathname relative to _testdata/topdir.
		want    string
		wantErr bool
	}{{
		name: "case1",
		dir:  "case1",
		want: "case1",
	}, {
		name: "case2",
		dir:  "case1/foo", // sic
		want: "case1",
	}, {
		name: "case3",
		dir:  "case3/foo",
		want: "case3",
	}, {
		name: "case4",
		dir:  "case3/foo/bar", // sic
		want: "case3",
	}, {
		name:    "case5",
		dir:     "case3", // no topdir here or above
		wantErr: true,
	}, {
		name:    "case6",
		dir:     "case6/foo/bar", // does not exist
		wantErr: true,
	}, {
		name: "case7",
		dir:  "case7/subdir1",
		want: "case7",
	}}

	tmpdir, err := os.MkdirTemp("", "fab")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	if err = copy.Copy("_testdata/topdir", tmpdir); err != nil {
		t.Fatal(err)
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fullpath := filepath.Join(tmpdir, tc.dir)
			got, err := TopDir(fullpath)
			if err != nil {
				if !tc.wantErr {
					t.Fatal(err)
				}
				return
			}
			if tc.wantErr {
				t.Fatal("got no error but wanted one")
			}

			rel, err := filepath.Rel(tmpdir, got)
			if err != nil {
				t.Fatal(err)
			}

			if rel != tc.want {
				t.Errorf("got %s, want %s", got, tc.want)
			}
		})
	}
}
