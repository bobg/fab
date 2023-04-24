package fab

import (
	"reflect"
	"testing"
)

func TestRegister(t *testing.T) {
	resetRegistry()

	target, err := RegisterTarget("target", "target doc", &countTarget{})
	if err != nil {
		t.Fatal(err)
	}

	if n := Describe(target); n != "target" {
		t.Errorf("got name %s, want target", n)
	}

	gotNames := RegistryNames()
	wantNames := []string{"target"}
	if !reflect.DeepEqual(gotNames, wantNames) {
		t.Errorf("got %v, want %v", gotNames, wantNames)
	}

	_, gotDoc := RegistryTarget("target")
	if gotDoc != "target doc" {
		t.Errorf(`got "%s", want "target doc"`, gotDoc)
	}

	gotTarget, _ := RegistryTarget("foobie_bletch")
	if gotTarget != nil {
		t.Errorf(`got non-nil target for "foobie_bletch", want nil`)
	}
}
