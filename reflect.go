package arrange

import (
	"reflect"

	"go.uber.org/fx"
)

// errorType is the cached reflection lookup for the error type
var errorType reflect.Type = reflect.TypeOf((*error)(nil)).Elem()

// ErrorType returns the reflection type for the error interface
func ErrorType() reflect.Type {
	return errorType
}

// inType is the cached reflection lookup for fx.In
var inType reflect.Type = reflect.TypeOf(fx.In{})

// InType returns the reflection type of fx.In
func InType() reflect.Type {
	return inType
}

// NewErrorValue is a convenience for safely producing a reflect.Value from an error.
// Useful when creating function stubs for reflect.MakeFunc.
func NewErrorValue(err error) reflect.Value {
	errPtr := reflect.New(ErrorType())
	if err != nil {
		errPtr.Elem().Set(reflect.ValueOf(err))
	}

	return errPtr.Elem()
}

// Target describes a sink for an unmarshal operation.
//
// Viper requires a pointer to be passed to its UnmarshalXXX functions.  However,
// this package uses a prototype pattern whereby a caller may specify a pointer
// or a value.  A target bridges that gap by storing the results of reflection
// from NewTarget to give a consistent way of referring to the actual object
// that should be unmarshaled as opposed to the object produced for dependency injection.
type Target struct {
	// Component refers the the actual value that should be returned from an uber/fx constructor.
	// This holds the value of the actual component that participates in dependency injection.
	Component reflect.Value

	// UnmarshalTo is the value that should be unmarshaled.  This value is always a pointer.
	// If Component is a pointer, UnmarshalTo will be the same value.  Otherwise, UnmarshalTo
	// will be a pointer the the Component value.
	UnmarshalTo reflect.Value
}

// NewTarget reflects a prototype object that describes the target
// for an unmarshaling operation.  The various unmarshalers and providers
// in this package that accept prototype objects use this function.
//
// The prototype itself is somewhat flexible:
//
// (1) The prototype may be a struct value.  A new struct is created with fields
// set to the same values as the prototype prior to unmarshaling.  The component
// will be a struct value of the same type, i.e. not a pointer to a struct.
//
//   NewTarget(Config{})
//   NewTarget(Config{Timeout: 15 * time.Second}) // a default value for Timeout
//
// can be used with:
//
//   fx.New(
//     fx.Invoke(
//       func(cfg Config) {},
//     ),
//   )
//
// (2) The prototype may be a non-nil pointer to a struct.  A new struct will be
// allocated with fields set to the same values as the prototype prior to
// unmarshaling.  The component will be pointer to this new struct.
//
//   NewTarget(&Config{})
//   NewTarget(new(Config))
//   NewTarget(&Config{Timeout: 15 * time.Second}) // a default value for Timeout
//
// can be used with:
//
//   fx.New(
//     fx.Invoke(
//       func(cfg *Config) {
//         // always a non-nil pointer, but any fields not unmarshaled
//         // will be set to their corresponding fields in the prototype
//       },
//     ),
//   )
//
// (3) The prototype may be a nil pointer to a struct.  A new struct of the same type
// will be created, but with all fields set to their zero values prior to unmarshaling.
// The component will be a pointer to this new struct.
//
//   NewTarget((*Config)(nil))
//
// can be used with:
//
//   fx.New(
//     fx.Invoke(
//       func(cfg *Config) {
//         // always a non-nil pointer, but any fields not unmarshaled
//         // will be set to their zero values
//       },
//     ),
//   )
//
// If the prototype does not refer to a struct, the results of this function are undefined.
func NewTarget(prototype interface{}) (t Target) {
	pvalue := reflect.ValueOf(prototype)
	if pvalue.Kind() == reflect.Ptr {
		t.UnmarshalTo = reflect.New(pvalue.Type().Elem())
		if !pvalue.IsNil() {
			t.UnmarshalTo.Elem().Set(pvalue.Elem())
		}

		t.Component = t.UnmarshalTo
	} else {
		t.UnmarshalTo = reflect.New(pvalue.Type())
		t.UnmarshalTo.Elem().Set(pvalue)
		t.Component = t.UnmarshalTo.Elem()
	}

	return
}

// VisitResult is the enumerated constant returned by a FieldVisitor
type VisitResult int

const (
	// VisitContinue indicates that field visitation should continue as normal
	VisitContinue VisitResult = iota

	// VisitSkip indicates that the fields of an embedded struct should not be visited.
	// If returned for any other kind of field, this is equivalent to VisitContinue.
	VisitSkip

	// VisitTerminate terminates the tree walk immediately
	VisitTerminate
)

// FieldVisitor is a strategy for visiting each exported field of a struct
type FieldVisitor func(reflect.StructField, reflect.Value) VisitResult

// IsDependency is a filter for struct fields that cannot possibly be injected
// dependencies.  This function returns true if all of the following are true:
//
//   - The reflect.Value is valid (i.e. IsValid() returns true)
//   - The reflect.Value can return an interface (i.e. CanInterface() returns true)
//   - The struct field is not anonymous
//   - The struct field is exported
//   - The struct field is either not the zero value or doesn't have the optional tag set to true
func IsDependency(f reflect.StructField, fv reflect.Value) bool {
	return fv.IsValid() &&
		fv.CanInterface() &&
		!f.Anonymous && // i.e. must not be embedded
		len(f.PkgPath) == 0 && // must be exported
		!(fv.IsZero() && f.Tag.Get("optional") == "true")
}

// VisitFields walks the tree of struct fields.  Each embedded struct is also
// traversed, but named struct fields are not.  Unexported fields are never traversed.
//
// If root is actually a reflect.Value, that value will be used or dereferenced if
// it is a pointer.
//
// If root is a struct or any level of pointer to a struct, it will be dereferenced
// and used as the starting point.
//
// If root is not a struct, or cannot be dereferenced to a struct, this function
// returns an invalid value, i.e. IsValid() will return false.  Also, an invalid
// value is returned if root is a nil pointer.
//
// If any traversal occurred, this function returns the actual reflect.Value representing
// the struct that was the root of the tree traversal.
func VisitFields(root interface{}, v FieldVisitor) reflect.Value {
	var rv reflect.Value
	if rt, ok := root.(reflect.Value); ok {
		rv = rt
	} else {
		rv = reflect.ValueOf(root)
	}

	// dereference as much as needed
	for rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			// can't traverse into a nil
			return reflect.ValueOf(nil)
		}

		rv = rv.Elem()
	}

	if rv.Kind() != reflect.Struct {
		return reflect.ValueOf(nil)
	}

	stack := []reflect.Value{rv}
	for len(stack) > 0 {
		var (
			end = len(stack) - 1
			s   = stack[end]
			st  = s.Type()
		)

		stack = stack[:end]
		for i := 0; i < st.NumField(); i++ {
			f := st.Field(i)
			if len(f.PkgPath) > 0 {
				// NOTE: don't consider unexported fields
				continue
			}

			fv := s.Field(i)
			if r := v(f, fv); r == VisitTerminate {
				return rv
			} else if f.Anonymous && r != VisitSkip {
				stack = append(stack, fv)
			}
		}
	}

	return rv
}

// ValueOf is a convenient utility function for turning v into a reflect.Value.
// If v is already a reflect.Value, it is returned as is.  Otherwise, the result
// of reflect.ValueOf(v) is returned.
func ValueOf(v interface{}) reflect.Value {
	if vv, ok := v.(reflect.Value); ok {
		return vv
	}

	return reflect.ValueOf(v)
}

// TypeOf is a convenient utility function for turning a v into a reflect.Type.
// If v is already a reflect.Type, it is returned as is.  If v is a reflect.Value,
// v.Type() is returned.  Otherwise, the result of reflect.TypeOf(v) is returned.
func TypeOf(v interface{}) reflect.Type {
	if vv, ok := v.(reflect.Value); ok {
		return vv.Type()
	} else if vt, ok := v.(reflect.Type); ok {
		return vt
	}

	return reflect.TypeOf(v)
}

// TryConvert attempts to convert dst into a slice of whatever type src is.  If src is
// itself a slice, then an attempt each made to convert each element of src into the dst type.
//
// The ConvertibleTo method in the reflect package is used to determine if conversion
// is possible.  If it is, then this function always returns a slice of the type
// referred to by dst.  This simplifies consumption of the result, as a caller may
// always safely cast it to a "[]dst" if the second return value is true.
//
// The src parameter may be an actual object or a reflect.Value.  The src may also be a slice
// type instead of a scalar.
//
// The dst parameter may be an actual object, a reflect.Value, or a reflect.Type.
//
// This function is useful in dependency injection situations when the
// allowed type should be looser than what golang allows.  For example, allowing
// a "func(http.Handler) http.Handler" where a "gorilla/mux.MiddlewareFunc" is desired.
//
// This function returns a nil interface{} and false if the conversion was not possible.
func TryConvert(dst, src interface{}) (interface{}, bool) {
	var (
		from = ValueOf(src)
		to   = TypeOf(dst)
	)

	switch {
	case from.Kind() == reflect.Array:
		fallthrough

	case from.Kind() == reflect.Slice:
		if from.Type().Elem().ConvertibleTo(to) {
			s := reflect.MakeSlice(
				reflect.SliceOf(to), // element type
				from.Len(),          // len
				from.Len(),          // cap
			)

			for i := 0; i < from.Len(); i++ {
				s.Index(i).Set(
					from.Index(i).Convert(to),
				)
			}

			return s.Interface(), true
		}

	case from.Type().ConvertibleTo(to):
		s := reflect.MakeSlice(
			reflect.SliceOf(to), // element type
			1,                   // len
			1,                   // cap
		)

		s.Index(0).Set(
			from.Convert(to),
		)

		return s.Interface(), true
	}

	return nil, false
}
