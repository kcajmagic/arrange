package arrangehttp

import (
	"context"
	"log"
	"net"
	"net/http"
	"reflect"
	"strings"

	"go.uber.org/multierr"
)

// InvalidServerOptionTypeError is returned by a ServerOption produced by AsServerOption
// to indicate that a type could not be converted.
type InvalidServerOptionTypeError struct {
	Type reflect.Type
}

// Error describes the type that could not be converted.
func (isote *InvalidServerOptionTypeError) Error() string {
	var o strings.Builder
	o.WriteString(isote.Type.String())
	o.WriteString(" cannot be converted to a ServerOption")

	return o.String()
}

// ServerOption is a general-purpose modifier for an http.Server.  Typically, these will
// created as value groups within an enclosing fx application.
type ServerOption interface {
	Apply(*http.Server) error
}

// ServerOptionFunc is a convenient function type that implements ServerOption.
type ServerOptionFunc func(*http.Server) error

// Apply invokes the function itself.
func (sof ServerOptionFunc) Apply(s *http.Server) error { return sof(s) }

// ServerOptions is an aggregate ServerOption that acts as a single option.
type ServerOptions []ServerOption

// Apply invokes each option in order.  Options are always invoked, even when
// one or more errors occur.  The returned error may be an aggregate error
// and can always be inspected via go.uber.org/multierr.
func (so ServerOptions) Apply(s *http.Server) (err error) {
	for _, o := range so {
		err = multierr.Append(err, o.Apply(s))
	}

	return
}

// Add appends options to this slice.  Each value is converted to a ServerOption
// via AsServerOption.
func (so *ServerOptions) Add(opts ...any) {
	if len(opts) == 0 {
		return
	}

	if cap(*so) < (len(*so) + len(opts)) {
		bigger := make(ServerOptions, 0, len(*so)+len(opts))
		bigger = append(bigger, *so...)
		*so = bigger
	}

	for _, o := range opts {
		*so = append(*so, AsServerOption(o))
	}
}

// AsServerOption converts a value into a ServerOption.  This function never returns nil
// and does not panic if v cannot be converted.
//
// Any of the following kinds of values can be converted:
//   - any type that implements ServerOption
//   - any type that supplies an Apply(*http.Server) method that returns no error
//   - an underlying type of func(*http.Server)
//   - an underlying type of func(*http.Server) error
//
// Any other kind of value will result in a ServerOption that returns an error indicating
// that the type cannot be converted.
func AsServerOption(v any) ServerOption {
	type serverOptionNoError interface {
		Apply(*http.Server)
	}

	if so, ok := v.(ServerOption); ok {
		return so
	} else if so, ok := v.(serverOptionNoError); ok {
		return ServerOptionFunc(func(s *http.Server) error {
			so.Apply(s)
			return nil
		})
	} else if f, ok := v.(func(*http.Server) error); ok {
		return ServerOptionFunc(f)
	} else if f, ok := v.(func(*http.Server)); ok {
		return ServerOptionFunc(func(s *http.Server) error {
			f(s)
			return nil
		})
	}

	return ServerOptionFunc(func(_ *http.Server) error {
		return &InvalidServerOptionTypeError{
			Type: reflect.TypeOf(v),
		}
	})
}

// BaseContext returns a server option that sets or replaces the http.Server.BaseContext function
func BaseContext(fn func(net.Listener) context.Context) ServerOption {
	return AsServerOption(func(s *http.Server) {
		s.BaseContext = fn
	})
}

// ConnContext returns a server option that sets or replaces the http.Server.ConnContext function
func ConnContext(fn func(context.Context, net.Conn) context.Context) ServerOption {
	return AsServerOption(func(s *http.Server) {
		s.ConnContext = fn
	})
}

// ErrorLog returns a server option that sets or replaces the http.Server.ErrorLog
func ErrorLog(l *log.Logger) ServerOption {
	return AsServerOption(func(s *http.Server) {
		s.ErrorLog = l
	})
}
