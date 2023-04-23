package proto

import (
	"reflect"
	"testing"
)

func TestDeps(t *testing.T) {
	want := []string{
		"testdata/foo.proto",
		"testdata/x/bar.proto",
		"testdata/x/plugh.proto",
	}

	got, err := Deps("testdata/foo.proto", []string{"testdata"})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
