package fab

import (
	"context"
	"go/types"
	"reflect"
)

var targetMethods = make(map[string]reflect.Method)

// nullTarget is here so we can get reflection info about Target
type nullTarget struct{}

var _ Target = nullTarget{}

func (nullTarget) ID() string                { return "" }
func (nullTarget) Run(context.Context) error { return nil }

func init() {
	var nt Target = nullTarget{}
	targetType := reflect.TypeOf(nt)
	for i := 0; i < targetType.NumMethod(); i++ {
		method := targetType.Method(i)
		targetMethods[method.Name] = method
	}
}

// implementsTarget tells whether the given type implements the Target interface.
func implementsTarget(typ types.Type) bool {
	methodSet := types.NewMethodSet(typ)
	for name, targetMethod := range targetMethods {
		m := methodSet.Lookup(nil, name) // TODO: understand whether/when the first arg needs to be supplied.
		if m == nil {
			// return fmt.Errorf("%s not found in method set", name)
			return false
		}
		f, ok := m.Obj().(*types.Func)
		if !ok {
			// return fmt.Errorf("%s is a %T, not a func", name, m.Obj())
			return false
		}
		sig, ok := f.Type().(*types.Signature)
		if !ok {
			// return fmt.Errorf("the type of func %s is %T, not a signature", name, f.Type())
			return false
		}

		if !signaturesMatch(sig, targetMethod.Func.Type(), true) {
			// return errors.Wrapf(err, "checking method %s", name)
			return false
		}
	}
	return true
}

func signaturesMatch(sig *types.Signature, fn reflect.Type, skipReceiver bool) bool {
	if fn.Kind() != reflect.Func {
		// return fmt.Errorf("kind is %v, not func", fn.Kind())
		return false
	}
	if sig.Variadic() != fn.IsVariadic() {
		// return fmt.Errorf("variadic mismatch")
		return false
	}

	hasReceiver := skipReceiver && (sig.Recv() != nil)

	params := sig.Params()
	nParamsWithReceiver := params.Len()
	if hasReceiver {
		nParamsWithReceiver++
	}

	if nParamsWithReceiver != fn.NumIn() {
		// return fmt.Errorf("parameter count %d != %d (with hasReceiver %v)", params.Len(), fn.NumIn(), hasReceiver)
		return false
	}
	results := sig.Results()
	if results.Len() != fn.NumOut() {
		// return fmt.Errorf("result count %d != %d", results.Len(), fn.NumOut())
		return false
	}

	for i := 0; i < params.Len(); i++ {
		j := i
		if hasReceiver {
			j++
		}
		sp, tp := params.At(i).Type(), fn.In(j)
		if !typesMatch(sp, tp) {
			// return errors.Wrapf(err, "checking param %d", i)
			return false
		}
	}
	for i := 0; i < results.Len(); i++ {
		sr, tr := results.At(i).Type(), fn.Out(i)
		if !typesMatch(sr, tr) {
			// return errors.Wrapf(err, "checking result %d", i)
			return false
		}
	}

	return true
}

// TODO: Handle parameterized types.
func typesMatch(t types.Type, r reflect.Type) bool {
	switch t := t.(type) {
	case *types.Array:
		if r.Kind() != reflect.Array {
			// return fmt.Errorf("kinds Array and %v do not match", r.Kind())
			return false
		}
		if t.Len() != int64(r.Len()) {
			// return fmt.Errorf("array lengths %d and %d do not match", t.Len(), r.Len())
			return false
		}
		return typesMatch(t.Elem(), r.Elem())

	case *types.Basic:
		switch t.Kind() {
		case types.Bool:
			if r.Kind() == reflect.Bool {
				return true
			}
		case types.Int:
			if r.Kind() == reflect.Int {
				return true
			}
		case types.Int8:
			if r.Kind() == reflect.Int8 {
				return true
			}
		case types.Int16:
			if r.Kind() == reflect.Int16 {
				return true
			}
		case types.Int32:
			if r.Kind() == reflect.Int32 {
				return true
			}
		case types.Int64:
			if r.Kind() == reflect.Int64 {
				return true
			}
		case types.Uint:
			if r.Kind() == reflect.Uint {
				return true
			}
		case types.Uint8:
			if r.Kind() == reflect.Uint8 {
				return true
			}
		case types.Uint16:
			if r.Kind() == reflect.Uint16 {
				return true
			}
		case types.Uint32:
			if r.Kind() == reflect.Uint32 {
				return true
			}
		case types.Uint64:
			if r.Kind() == reflect.Uint64 {
				return true
			}
		case types.Uintptr:
			if r.Kind() == reflect.Uintptr {
				return true
			}
		case types.Float32:
			if r.Kind() == reflect.Float32 {
				return true
			}
		case types.Float64:
			if r.Kind() == reflect.Float64 {
				return true
			}
		case types.Complex64:
			if r.Kind() == reflect.Complex64 {
				return true
			}
		case types.Complex128:
			if r.Kind() == reflect.Complex128 {
				return true
			}
		case types.String:
			if r.Kind() == reflect.String {
				return true
			}
		case types.UnsafePointer:
			if r.Kind() == reflect.UnsafePointer {
				return true
			}
		}
		// return fmt.Errorf("kinds %v and %v do not match", t.Kind(), r.Kind())
		return false

	case *types.Chan:
		if r.Kind() != reflect.Chan {
			// return fmt.Errorf("kinds Chan and %v do not match", r.Kind())
			return false
		}
		switch t.Dir() {
		case types.SendRecv:
			if r.ChanDir() != reflect.BothDir {
				// return fmt.Errorf("channel direction SendRecv and %v do not match", r.ChanDir())
				return false
			}
		case types.SendOnly:
			if r.ChanDir() != reflect.SendDir {
				// return fmt.Errorf("channel direction SendOnly and %v do not match", r.ChanDir())
				return false
			}
		case types.RecvOnly:
			if r.ChanDir() != reflect.RecvDir {
				// return fmt.Errorf("channel direction RecvOnly and %v do not match", r.ChanDir())
				return false
			}
		}
		return typesMatch(t.Elem(), r.Elem())

	case *types.Interface:
		if r.Kind() != reflect.Interface {
			// return fmt.Errorf("kinds Interface and %v do not match", r.Kind())
			return false
		}
		methodSet := types.NewMethodSet(t)
		if methodSet.Len() != r.NumMethod() {
			// return fmt.Errorf("method set lengths %d != %d", methodSet.Len(), r.NumMethod())
			return false
		}
		for i := 0; i < methodSet.Len(); i++ {
			f, ok := methodSet.At(i).Obj().(*types.Func)
			if !ok {
				// return fmt.Errorf("method set member %d is a %T, not a func", i, methodSet.At(i).Obj())
				return false
			}
			method, ok := r.MethodByName(f.Name())
			if !ok {
				// return fmt.Errorf("method %s is not in both types", f.Name())
				return false
			}

			sig, ok := f.Type().(*types.Signature)
			if !ok {
				// return fmt.Errorf("%s is a %T, not a signature", f.Name(), f.Type())
				return false
			}
			if !signaturesMatch(sig, method.Type, false) {
				// return errors.Wrapf(err, "checking %s", f.Name())
				return false
			}
		}
		return true

	case *types.Map:
		if r.Kind() != reflect.Map {
			// return fmt.Errorf("kinds Map and %v do not match", r.Kind())
			return false
		}
		if !typesMatch(t.Key(), r.Key()) {
			return false
		}
		return typesMatch(t.Elem(), r.Elem())

	case *types.Named:
		if t.Obj().Name() != r.Name() {
			// return fmt.Errorf("names %s and %s do not match", t.Obj().Name(), r.Name())
			return false
		}
		return typesMatch(t.Underlying(), r)

	case *types.Pointer:
		if r.Kind() != reflect.Ptr {
			// return fmt.Errorf("kinds Pointer and %v do not match", r.Kind())
			return false
		}
		return typesMatch(t.Elem(), r.Elem())

	case *types.Signature:
		return signaturesMatch(t, r, true)

	case *types.Slice:
		if r.Kind() != reflect.Slice {
			// return fmt.Errorf("kinds Slice and %v do not match", r.Kind())
			return false
		}
		return typesMatch(t.Elem(), r.Elem())

	case *types.Struct:
		if r.Kind() != reflect.Struct {
			// return fmt.Errorf("kinds Struct and %v do not match", r.Kind())
			return false
		}
		if t.NumFields() != r.NumField() {
			// return fmt.Errorf("number of fields %d != %d", t.NumFields(), r.NumField())
			return false
		}
		for i := 0; i < t.NumFields(); i++ {
			v, f := t.Field(i), r.Field(i)
			if v.Name() != f.Name {
				// return fmt.Errorf("field name %s != %s", v.Name(), f.Name)
				return false
			}
			if t.Tag(i) != string(f.Tag) {
				// return fmt.Errorf("struct field tag mismatch in field %d", i)
				return false
			}
			if !typesMatch(v.Type(), f.Type) {
				// return errors.Wrapf(err, "checking field %d", i)
				return false
			}
		}
		return true

		// case *types.Tuple:
		// case *types.TypeParam:
		// case *types.Union:
	}

	// return fmt.Errorf("unhandled case")
	return false
}
