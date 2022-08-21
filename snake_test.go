package fab

import (
	"fmt"
	"testing"
)

func TestToSnake(t *testing.T) {
	cases := []struct {
		inp, want string
	}{{
		inp: "x", want: "x",
	}, {
		inp: "X", want: "x",
	}, {
		inp: "abCd", want: "ab_cd",
	}}
	for i, tc := range cases {
		t.Run(fmt.Sprintf("case_%02d", i+1), func(t *testing.T) {
			got := toSnake(tc.inp)
			if got != tc.want {
				t.Errorf("got %s, want %s", got, tc.want)
			}
		})
	}
}
