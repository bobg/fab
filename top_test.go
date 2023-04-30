package fab

import (
	"io/fs"
	"os"
	"testing"
)

func TestTopDir(t *testing.T) {
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
		dir:  "case3",
		want: "case3",
	}, {
		name: "case4",
		dir:  "case3/foo", // sic
		want: "case3",
	}, {
		name: "case5",
		dir:  "case5/foo",
		want: "case5",
	}, {
		name: "case6",
		dir:  "case5/foo/bar", // sic
		want: "case5",
	}}

	// https://pkg.go.dev/os#DirFS assures us that the result of os.DirFS implements StatFS.
	fsys := os.DirFS("_testdata/topdir").(fs.StatFS)

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := topDir(fsys, tc.dir)
			if err != nil {
				if !tc.wantErr {
					t.Fatal(err)
				}
			} else if tc.wantErr {
				t.Fatal("got no error but wanted one")
			}

			if got != tc.want {
				t.Errorf("got %s, want %s", got, tc.want)
			}
		})
	}
}
