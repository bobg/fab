package loader

import (
	"context"
	"fmt"
	"go/types"
	"reflect"
	"strings"

	"github.com/bobg/fab"
)

var targetMethods = make(map[string]reflect.Method)

// nullTarget is here so we can get reflection info about Target
type nullTarget struct{}

var _ fab.Target = nullTarget{}

func (nullTarget) ID() string                { return "" }
func (nullTarget) Run(context.Context) error { return nil }

func init() {
	var nt fab.Target = nullTarget{}
	targetType := reflect.TypeOf(nt)
	for i := 0; i < targetType.NumMethod(); i++ {
		method := targetType.Method(i)
		targetMethods[method.Name] = method
	}
}

func implementsTarget(typ types.Type) bool {
	methodSet := types.NewMethodSet(typ)
	for name, targetMethod := range targetMethods {
		m := methodSet.Lookup(nil, name) // TODO: understand whether/when the first arg needs to be supplied.
		if m == nil {
			return false
		}
		f, ok := m.Obj().(*types.Func)
		if !ok {
			return false
		}
		sig, ok := f.Type().(*types.Signature)
		if !ok {
			return false
		}

		var comp comparer
		if !comp.signaturesMatch(sig, targetMethod.Func.Type(), true) {
			return false
		}
	}
	return true
}

type comparer struct {
	depth int
}

func (comp *comparer) debugf(msg string, args ...any) {
	if true {
		return
	}

	fmt.Print(strings.Repeat("  ", comp.depth))
	fmt.Printf(msg, args...)
	fmt.Print("\n")
}

func (comp *comparer) signaturesMatch(sig *types.Signature, fn reflect.Type, skipReceiver bool) (result bool) {
	comp.debugf("signaturesMatch(%s, %s)", sig, fn)
	comp.depth++
	defer func() {
		comp.depth--
		comp.debugf("signaturesMatch(%s, %s) -> %v", sig, fn, result)
	}()

	if fn.Kind() != reflect.Func {
		comp.debugf("  fn.Kind() is %s, not Func", fn.Kind())
		return false
	}
	if sig.Variadic() != fn.IsVariadic() {
		comp.debugf("  sig.Variadic is %v, fn.IsVariadic is %v", sig.Variadic(), fn.IsVariadic())
		return false
	}

	hasReceiver := skipReceiver && (sig.Recv() != nil)

	params := sig.Params()
	nParamsWithReceiver := params.Len()
	if hasReceiver {
		nParamsWithReceiver++
	}

	if nParamsWithReceiver != fn.NumIn() {
		comp.debugf("hasReceiver is %v and %d does not match %d", hasReceiver, params.Len(), fn.NumIn())
		return false
	}
	results := sig.Results()
	if results.Len() != fn.NumOut() {
		comp.debugf("results.Len is %d and fn.NumOut is %d", results.Len(), fn.NumOut())
		return false
	}

	for i := 0; i < params.Len(); i++ {
		j := i
		if hasReceiver {
			j++
		}
		sp, tp := params.At(i).Type(), fn.In(j)
		if !comp.typesMatch(sp, tp) {
			return false
		}
	}
	for i := 0; i < results.Len(); i++ {
		sr, tr := results.At(i).Type(), fn.Out(i)
		if !comp.typesMatch(sr, tr) {
			return false
		}
	}

	return true
}

// TODO: Handle parameterized types.
func (comp *comparer) typesMatch(t types.Type, r reflect.Type) (result bool) {
	comp.debugf("typesMatch(%s, %s)", t, r)
	comp.depth++
	defer func() {
		comp.depth--
		comp.debugf("typesMatch(%s, %s) -> %v", t, r, result)
	}()

	switch t := t.(type) {
	case *types.Array:
		if r.Kind() != reflect.Array {
			return false
		}
		if t.Len() != int64(r.Len()) {
			return false
		}
		return comp.typesMatch(t.Elem(), r.Elem())

	case *types.Basic:
		switch t.Kind() {
		case types.Bool:
			return r.Kind() == reflect.Bool
		case types.Int:
			return r.Kind() == reflect.Int
		case types.Int8:
			return r.Kind() == reflect.Int8
		case types.Int16:
			return r.Kind() == reflect.Int16
		case types.Int32:
			return r.Kind() == reflect.Int32
		case types.Int64:
			return r.Kind() == reflect.Int64
		case types.Uint:
			return r.Kind() == reflect.Uint
		case types.Uint8:
			return r.Kind() == reflect.Uint8
		case types.Uint16:
			return r.Kind() == reflect.Uint16
		case types.Uint32:
			return r.Kind() == reflect.Uint32
		case types.Uint64:
			return r.Kind() == reflect.Uint64
		case types.Uintptr:
			return r.Kind() == reflect.Uintptr
		case types.Float32:
			return r.Kind() == reflect.Float32
		case types.Float64:
			return r.Kind() == reflect.Float64
		case types.Complex64:
			return r.Kind() == reflect.Complex64
		case types.Complex128:
			return r.Kind() == reflect.Complex128
		case types.String:
			return r.Kind() == reflect.String
		case types.UnsafePointer:
			return r.Kind() == reflect.UnsafePointer
		}
		return false

	case *types.Chan:
		if r.Kind() != reflect.Chan {
			return false
		}
		switch t.Dir() {
		case types.SendRecv:
			if r.ChanDir() != reflect.BothDir {
				return false
			}
		case types.SendOnly:
			if r.ChanDir() != reflect.SendDir {
				return false
			}
		case types.RecvOnly:
			if r.ChanDir() != reflect.RecvDir {
				return false
			}
		}
		return comp.typesMatch(t.Elem(), r.Elem())

	case *types.Interface:
		if r.Kind() != reflect.Interface {
			comp.debugf("r.Kind is %s, not Interface", r.Kind())
			return false
		}
		methodSet := types.NewMethodSet(t)
		if methodSet.Len() != r.NumMethod() {
			comp.debugf("methodSet.Len() is %d, r.NumMethod() is %d", methodSet.Len(), r.NumMethod())
			return false
		}
		for i := 0; i < methodSet.Len(); i++ {
			f, ok := methodSet.At(i).Obj().(*types.Func)
			if !ok {
				comp.debugf("methodSet.At(%d).Obj() is a %T, not a Func", i, methodSet.At(i).Obj())
				return false
			}
			method, ok := r.MethodByName(f.Name())
			if !ok {
				comp.debugf("r has no method %s", f.Name())
				return false
			}

			comp.debugf("f = %s, method.Type = %s", f, method.Type)

			sig, ok := f.Type().(*types.Signature)
			if !ok {
				comp.debugf("f.Type() is a %T, not a Signature", f.Type())
				return false
			}
			if !comp.signaturesMatch(sig, method.Type, false) {
				comp.debugf("sig %s does not match method type %s", sig, method.Type)
				return false
			}
		}
		return true

	case *types.Map:
		if r.Kind() != reflect.Map {
			return false
		}
		if !comp.typesMatch(t.Key(), r.Key()) {
			return false
		}
		return comp.typesMatch(t.Elem(), r.Elem())

	case *types.Named:
		if t.Obj().Name() != r.Name() {
			return false
		}
		return comp.typesMatch(t.Underlying(), r)

	case *types.Pointer:
		if r.Kind() != reflect.Ptr {
			return false
		}
		return comp.typesMatch(t.Elem(), r.Elem())

	case *types.Signature:
		return comp.signaturesMatch(t, r, true)

	case *types.Slice:
		if r.Kind() != reflect.Slice {
			return false
		}
		return comp.typesMatch(t.Elem(), r.Elem())

	case *types.Struct:
		if r.Kind() != reflect.Struct {
			return false
		}
		if t.NumFields() != r.NumField() {
			return false
		}
		for i := 0; i < t.NumFields(); i++ {
			v, f := t.Field(i), r.Field(i)
			if v.Name() != f.Name {
				return false
			}
			if t.Tag(i) != string(f.Tag) {
				return false
			}
			if !comp.typesMatch(v.Type(), f.Type) {
				return false
			}
		}
		return true

		// case *types.Tuple:
		// case *types.TypeParam:
		// case *types.Union:
	}

	return false
}
