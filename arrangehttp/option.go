package arrangehttp

import "go.uber.org/multierr"

// Option represents something that can modify a target object.
type Option[T any] interface {
	Apply(*T) error
}

// OptionFunc is a closure type that can act as an Option.
type OptionFunc[T any] func(*T) error

func (of OptionFunc[T]) Apply(t *T) error {
	return of(t)
}

// Options is an aggregate Option that allows several options to
// be grouped together.
type Options[T any] []Option[T]

func (o Options[T]) Apply(t *T) (err error) {
	for _, opt := range o {
		err = multierr.Append(err, opt.Apply(t))
	}

	return
}

// OptionClosure represents the closure types that are convertible
// into Option objects.
type OptionClosure[T any] interface {
	~func(*T) | ~func(*T) error
}

// AsOption converts a closure into an Option for a given target type.
func AsOption[T any, F OptionClosure[T]](f F) Option[T] {
	fv := any(f)
	if of, ok := fv.(func(*T) error); ok {
		return OptionFunc[T](of)
	}

	return OptionFunc[T](func(t *T) error {
		fv.(func(*T))(t)
		return nil
	})
}

// ApplyOptions applies several options to a target.  This function
// can be used to decorate targets via fx.Decorate.
func ApplyOptions[T any](t *T, opts ...Option[T]) (err error) {
	for _, o := range opts {
		err = multierr.Append(err, o.Apply(t))
	}

	return
}

// InvalidOption returns an Option that returns the given error.
// Useful instead of nil or a panic to indicate that something in the setup
// of an Option went wrong.
func InvalidOption[T any](err error) Option[T] {
	return OptionFunc[T](func(_ *T) error {
		return err
	})
}
