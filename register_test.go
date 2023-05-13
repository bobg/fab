package fab

import (
	"reflect"
	"testing"
)

func TestRegister(t *testing.T) {
	t.Parallel()

	con := NewController("")

	target, err := con.RegisterTarget("target", "target doc", &countTarget{})
	if err != nil {
		t.Fatal(err)
	}

	if n := con.Describe(target); n != "target" {
		t.Errorf("got name %s, want target", n)
	}

	gotNames := con.RegistryNames()
	wantNames := []string{"target"}
	if !reflect.DeepEqual(gotNames, wantNames) {
		t.Errorf("got %v, want %v", gotNames, wantNames)
	}

	_, gotDoc := con.RegistryTarget("target")
	if gotDoc != "target doc" {
		t.Errorf(`got "%s", want "target doc"`, gotDoc)
	}

	gotTarget, _ := con.RegistryTarget("foobie_bletch")
	if gotTarget != nil {
		t.Errorf(`got non-nil target for "foobie_bletch", want nil`)
	}
}

func TestDescribe(t *testing.T) {
	t.Parallel()

	con := NewController("")

	targ1 := &countTarget{}
	if _, err := con.RegisterTarget("targ1", "", targ1); err != nil {
		t.Fatal(err)
	}

	got := con.Describe(targ1)
	if got != "targ1" {
		t.Errorf("got %s, want targ1", got)
	}

	targ2 := &countTarget{}
	got = con.Describe(targ2)
	if got != "unnamed count" {
		t.Errorf("got %s, want countTarget", got)
	}
}
