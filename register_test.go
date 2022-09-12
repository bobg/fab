package fab

import (
	"reflect"
	"testing"
)

func TestRegister(t *testing.T) {
	var (
		target     Target
		hashTarget HashTarget
	)
	Register("target", "target doc", target)
	Register("hash_target", "hash_target doc", hashTarget)

	gotNames := RegistryNames()
	wantNames := []string{"hash_target", "target"}
	if !reflect.DeepEqual(gotNames, wantNames) {
		t.Errorf("got %v, want %v", gotNames, wantNames)
	}

	_, gotDoc := RegistryTarget("target")
	if gotDoc != "target doc" {
		t.Errorf(`got "%s", want "target doc"`, gotDoc)
	}
	_, gotDoc = RegistryTarget("hash_target")
	if gotDoc != "hash_target doc" {
		t.Errorf(`got "%s", want "hash_target doc"`, gotDoc)
	}
	gotTarget, _ := RegistryTarget("foobie_bletch")
	if gotTarget != nil {
		t.Errorf(`got non-nil target for "foobie_bletch", want nil`)
	}
}
