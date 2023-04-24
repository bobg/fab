package fab

import (
	"context"
	"fmt"
	"go/types"
	"reflect"

	"github.com/bobg/errors"
)

var targetMethods = make(map[string]reflect.Method)

// nullTarget is here so we can get reflection info about Target
type nullTarget struct{}

var _ Target = nullTarget{}

func (nullTarget) Run(context.Context) error { return nil }
func (nullTarget) Desc() string              { return "(null)" }

func init() {
	var nt Target = nullTarget{}
	targetType := reflect.TypeOf(nt)
	for i := 0; i < targetType.NumMethod(); i++ {
		method := targetType.Method(i)
		targetMethods[method.Name] = method
	}
}

// checkImplementsTarget checks whether the given type implements the Target interface.
func checkImplementsTarget(typ types.Type) error {
	methodSet := types.NewMethodSet(typ)
	for name, targetMethod := range targetMethods {
		m := methodSet.Lookup(nil, name) // TODO: understand whether/when the first arg needs to be supplied.
		if m == nil {
			return fmt.Errorf("%s not found in method set", name)
		}
		f, ok := m.Obj().(*types.Func)
		if !ok {
			return fmt.Errorf("%s is a %T, not a func", name, m.Obj())
		}
		sig, ok := f.Type().(*types.Signature)
		if !ok {
			return fmt.Errorf("the type of func %s is %T, not a signature", name, f.Type())
		}

		if err := checkSignaturesMatch(sig, targetMethod.Func.Type(), true); err != nil {
			return errors.Wrapf(err, "checking method %s", name)
		}
	}
	return nil
}

func checkSignaturesMatch(sig *types.Signature, fn reflect.Type, skipReceiver bool) (err error) {
	if fn.Kind() != reflect.Func {
		return fmt.Errorf("kind is %v, not func", fn.Kind())
	}
	if sig.Variadic() != fn.IsVariadic() {
		return fmt.Errorf("variadic mismatch")
	}

	hasReceiver := skipReceiver && (sig.Recv() != nil)

	params := sig.Params()
	nParamsWithReceiver := params.Len()
	if hasReceiver {
		nParamsWithReceiver++
	}

	if nParamsWithReceiver != fn.NumIn() {
		return fmt.Errorf("parameter count %d != %d (with hasReceiver %v)", params.Len(), fn.NumIn(), hasReceiver)
	}
	results := sig.Results()
	if results.Len() != fn.NumOut() {
		return fmt.Errorf("result count %d != %d", results.Len(), fn.NumOut())
	}

	for i := 0; i < params.Len(); i++ {
		j := i
		if hasReceiver {
			j++
		}
		sp, tp := params.At(i).Type(), fn.In(j)
		if err = checkTypesMatch(sp, tp); err != nil {
			return errors.Wrapf(err, "checking param %d", i)
		}
	}
	for i := 0; i < results.Len(); i++ {
		sr, tr := results.At(i).Type(), fn.Out(i)
		if err = checkTypesMatch(sr, tr); err != nil {
			return errors.Wrapf(err, "checking result %d", i)
		}
	}

	return nil
}

// TODO: Handle parameterized types.
func checkTypesMatch(t types.Type, r reflect.Type) (err error) {
	switch t := t.(type) {
	case *types.Array:
		if r.Kind() != reflect.Array {
			return fmt.Errorf("kinds Array and %v do not match", r.Kind())
		}
		if t.Len() != int64(r.Len()) {
			return fmt.Errorf("array lengths %d and %d do not match", t.Len(), r.Len())
		}
		return checkTypesMatch(t.Elem(), r.Elem())

	case *types.Basic:
		switch t.Kind() {
		case types.Bool:
			if r.Kind() == reflect.Bool {
				return nil
			}
		case types.Int:
			if r.Kind() == reflect.Int {
				return nil
			}
		case types.Int8:
			if r.Kind() == reflect.Int8 {
				return nil
			}
		case types.Int16:
			if r.Kind() == reflect.Int16 {
				return nil
			}
		case types.Int32:
			if r.Kind() == reflect.Int32 {
				return nil
			}
		case types.Int64:
			if r.Kind() == reflect.Int64 {
				return nil
			}
		case types.Uint:
			if r.Kind() == reflect.Uint {
				return nil
			}
		case types.Uint8:
			if r.Kind() == reflect.Uint8 {
				return nil
			}
		case types.Uint16:
			if r.Kind() == reflect.Uint16 {
				return nil
			}
		case types.Uint32:
			if r.Kind() == reflect.Uint32 {
				return nil
			}
		case types.Uint64:
			if r.Kind() == reflect.Uint64 {
				return nil
			}
		case types.Uintptr:
			if r.Kind() == reflect.Uintptr {
				return nil
			}
		case types.Float32:
			if r.Kind() == reflect.Float32 {
				return nil
			}
		case types.Float64:
			if r.Kind() == reflect.Float64 {
				return nil
			}
		case types.Complex64:
			if r.Kind() == reflect.Complex64 {
				return nil
			}
		case types.Complex128:
			if r.Kind() == reflect.Complex128 {
				return nil
			}
		case types.String:
			if r.Kind() == reflect.String {
				return nil
			}
		case types.UnsafePointer:
			if r.Kind() == reflect.UnsafePointer {
				return nil
			}
		}
		return fmt.Errorf("kinds %v and %v do not match", t.Kind(), r.Kind())

	case *types.Chan:
		if r.Kind() != reflect.Chan {
			return fmt.Errorf("kinds Chan and %v do not match", r.Kind())
		}
		switch t.Dir() {
		case types.SendRecv:
			if r.ChanDir() != reflect.BothDir {
				return fmt.Errorf("channel direction SendRecv and %v do not match", r.ChanDir())
			}
		case types.SendOnly:
			if r.ChanDir() != reflect.SendDir {
				return fmt.Errorf("channel direction SendOnly and %v do not match", r.ChanDir())
			}
		case types.RecvOnly:
			if r.ChanDir() != reflect.RecvDir {
				return fmt.Errorf("channel direction RecvOnly and %v do not match", r.ChanDir())
			}
		}
		return checkTypesMatch(t.Elem(), r.Elem())

	case *types.Interface:
		if r.Kind() != reflect.Interface {
			return fmt.Errorf("kinds Interface and %v do not match", r.Kind())
		}
		methodSet := types.NewMethodSet(t)
		if methodSet.Len() != r.NumMethod() {
			return fmt.Errorf("method set lengths %d != %d", methodSet.Len(), r.NumMethod())
		}
		for i := 0; i < methodSet.Len(); i++ {
			f, ok := methodSet.At(i).Obj().(*types.Func)
			if !ok {
				return fmt.Errorf("method set member %d is a %T, not a func", i, methodSet.At(i).Obj())
			}
			method, ok := r.MethodByName(f.Name())
			if !ok {
				return fmt.Errorf("method %s is not in both types", f.Name())
			}

			sig, ok := f.Type().(*types.Signature)
			if !ok {
				return fmt.Errorf("%s is a %T, not a signature", f.Name(), f.Type())
			}
			if err = checkSignaturesMatch(sig, method.Type, false); err != nil {
				return errors.Wrapf(err, "checking %s", f.Name())
			}
		}
		return nil

	case *types.Map:
		if r.Kind() != reflect.Map {
			return fmt.Errorf("kinds Map and %v do not match", r.Kind())
		}
		if err = checkTypesMatch(t.Key(), r.Key()); err != nil {
			return err
		}
		return checkTypesMatch(t.Elem(), r.Elem())

	case *types.Named:
		if t.Obj().Name() != r.Name() {
			return fmt.Errorf("names %s and %s do not match", t.Obj().Name(), r.Name())
		}
		return checkTypesMatch(t.Underlying(), r)

	case *types.Pointer:
		if r.Kind() != reflect.Ptr {
			return fmt.Errorf("kinds Pointer and %v do not match", r.Kind())
		}
		return checkTypesMatch(t.Elem(), r.Elem())

	case *types.Signature:
		return checkSignaturesMatch(t, r, true)

	case *types.Slice:
		if r.Kind() != reflect.Slice {
			return fmt.Errorf("kinds Slice and %v do not match", r.Kind())
		}
		return checkTypesMatch(t.Elem(), r.Elem())

	case *types.Struct:
		if r.Kind() != reflect.Struct {
			return fmt.Errorf("kinds Struct and %v do not match", r.Kind())
		}
		if t.NumFields() != r.NumField() {
			return fmt.Errorf("number of fields %d != %d", t.NumFields(), r.NumField())
		}
		for i := 0; i < t.NumFields(); i++ {
			v, f := t.Field(i), r.Field(i)
			if v.Name() != f.Name {
				return fmt.Errorf("field name %s != %s", v.Name(), f.Name)
			}
			if t.Tag(i) != string(f.Tag) {
				return fmt.Errorf("struct field tag mismatch in field %d", i)
			}
			if err = checkTypesMatch(v.Type(), f.Type); err != nil {
				return errors.Wrapf(err, "checking field %d", i)
			}
		}
		return nil

		// case *types.Tuple:
		// case *types.TypeParam:
		// case *types.Union:
	}

	return fmt.Errorf("unhandled case")
}
