package fab

import (
	"fmt"
	"testing"
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
