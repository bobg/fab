package fab

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"

	"github.com/bradleyjkemp/cupaloy/v2"
)

func TestJoinPath(t *testing.T) {
	cases := []struct {
		inp  []string
		want string
	}{{
		want: "TOP",
	}, {
		inp:  []string{"a"},
		want: "TOP/a",
	}, {
		inp:  []string{"a/b"},
		want: "TOP/a/b",
	}, {
		inp:  []string{"a/b", "c/d"},
		want: "TOP/a/b/c/d",
	}, {
		inp:  []string{"a/b", "/c/d"},
		want: "/c/d",
	}}

	con := &Controller{topdir: "TOP"}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("case_%02d", i+1), func(t *testing.T) {
			got := con.JoinPath(tc.inp...)
			if got != tc.want {
				t.Errorf("got %s, want %s", got, tc.want)
			}
		})
	}
}

func TestParseArgs(t *testing.T) {
	con := NewController("")
	t1, err := con.RegisterTarget("t1", "", &countTarget{})
	if err != nil {
		t.Fatal(err)
	}
	t2, err := con.RegisterTarget("t2", "", &countTarget{})
	if err != nil {
		t.Fatal(err)
	}

	got1, err := con.ParseArgs([]string{"t1", "t2"})
	if err != nil {
		t.Fatal(err)
	}
	want1 := []Target{t1, t2}
	if !reflect.DeepEqual(got1, want1) {
		t.Error("mismatch")
	}

	got2, err := con.ParseArgs([]string{"t1", "-foo", "bar"})
	if err != nil {
		t.Fatal(err)
	}
	want2 := []Target{ArgTarget(t1, "-foo", "bar")}
	if !reflect.DeepEqual(got2, want2) {
		t.Error("mismatch")
	}
}

func TestListTargets(t *testing.T) {
	con := NewController("")
	_, err := con.RegisterTarget("t1", "This is t1.", &countTarget{})
	if err != nil {
		t.Fatal(err)
	}
	_, err = con.RegisterTarget("t2", "And this is t2.", &countTarget{})
	if err != nil {
		t.Fatal(err)
	}

	buf := new(bytes.Buffer)
	con.ListTargets(buf)

	snaps := cupaloy.New(cupaloy.SnapshotSubdirectory("_testdata"))
	snaps.SnapshotT(t, buf.String())
}
