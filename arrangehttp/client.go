package arrangehttp

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"time"

	"github.com/xmidt-org/arrange"
	"github.com/xmidt-org/arrange/arrangetls"
	"github.com/xmidt-org/httpaux"
	"go.uber.org/fx"
	"go.uber.org/multierr"
)

// ClientFactory is the interface implemented by unmarshaled configuration objects
// that produces an http.Client.  The default implementation of this interface is ClientConfig.
type ClientFactory interface {
	NewClient() (*http.Client, error)
}

// TransportConfig holds the unmarshalable configuration options for building an http.Transport
type TransportConfig struct {
	TLSHandshakeTimeout    time.Duration
	DisableKeepAlives      bool
	DisableCompression     bool
	MaxIdleConns           int
	MaxIdleConnsPerHost    int
	MaxConnsPerHost        int
	IdleConnTimeout        time.Duration
	ResponseHeaderTimeout  time.Duration
	ExpectContinueTimeout  time.Duration
	ProxyConnectHeader     http.Header
	MaxResponseHeaderBytes int64
	WriteBufferSize        int
	ReadBufferSize         int
	ForceAttemptHTTP2      bool
}

// NewTransport creates an http.Transport using this unmarshaled configuration
// together with TLS information
func (tc TransportConfig) NewTransport(c *arrangetls.Config) (transport *http.Transport, err error) {
	transport = &http.Transport{
		TLSHandshakeTimeout:    tc.TLSHandshakeTimeout,
		DisableKeepAlives:      tc.DisableKeepAlives,
		DisableCompression:     tc.DisableCompression,
		MaxIdleConns:           tc.MaxIdleConns,
		MaxIdleConnsPerHost:    tc.MaxIdleConnsPerHost,
		MaxConnsPerHost:        tc.MaxConnsPerHost,
		IdleConnTimeout:        tc.IdleConnTimeout,
		ResponseHeaderTimeout:  tc.ResponseHeaderTimeout,
		ExpectContinueTimeout:  tc.ExpectContinueTimeout,
		ProxyConnectHeader:     tc.ProxyConnectHeader,
		MaxResponseHeaderBytes: tc.MaxResponseHeaderBytes,
		WriteBufferSize:        tc.WriteBufferSize,
		ReadBufferSize:         tc.ReadBufferSize,
		ForceAttemptHTTP2:      tc.ForceAttemptHTTP2,
	}

	transport.TLSClientConfig, err = c.New()
	return
}

// ClientConfig holds unmarshaled client configuration options.  It is the
// built-in ClientFactory implementation in this package.
type ClientConfig struct {
	Timeout   time.Duration
	Transport TransportConfig
	Header    http.Header
	TLS       *arrangetls.Config
}

// NewClient produces an http.Client given these unmarshaled configuration options
func (cc ClientConfig) NewClient() (client *http.Client, err error) {
	client = &http.Client{
		Timeout: cc.Timeout,
	}

	header := httpaux.NewHeader(cc.Header)
	transport, err := cc.Transport.NewTransport(cc.TLS)
	if err == nil {
		client.Transport = header.RoundTrip(transport)
	}

	return
}

// ClientIn is the set of dependencies required to build an *http.Client component.
// A parameter of this struct type will always be the first input parameter to
// the dynamic function generated by Client().Unmarshal or Client().UnmarshalKey.
type ClientIn struct {
	fx.In

	// Unmarshaler is the optional arrange Unmarshaler component used to unmarshal
	// a ClientFactory.  If this component is not supplied, the ClientFactory is used as is.
	Unmarshaler arrange.Unmarshaler `optional:"true"`

	// Printer is the optional fx.Printer used to output informational messages about
	// client unmarshaling and configuration.  If unset, arrange.DefaultPrinter() is used.
	Printer fx.Printer `optional:"true"`

	// Lifecycle is used to bind http.Client.CloseIdleConnections to the
	// fx.App OnStop event
	Lifecycle fx.Lifecycle
}

// C is a Fluent Builder for creating an http.Client as an uber/fx component.
// This type should be constructed with the Client function.
type C struct {
	// errs are the collected errors that happened during fluent building
	errs []error

	// options are the explicit options added that are NOT injected
	options []ClientOption

	// dependencies are extra dependencies beyond ClientIn
	dependencies []reflect.Type

	// prototype is the ClientFactory instance that is used for cloning and unmarshaling
	prototype ClientFactory
}

// Client begins a Fluent Builder chain for constructing an http.Client from
// unmarshaled configuration and introducing that http.Client as a component
// for an enclosing uber/fx app.
func Client() *C {
	return new(C).
		ClientFactory(ClientConfig{})
}

// ClientFactory sets the prototype factory that is unmarshaled from Viper.
// This prototype obeys the rules of arrange.NewTarget.  By default, ClientConfig
// is used as the ClientFactory.  This build method allows a caller to use
// custom configuration.
func (c *C) ClientFactory(prototype ClientFactory) *C {
	c.prototype = prototype
	return c
}

// With adds any number of externally supplied options that will be applied
// to the client when the enclosing fx.App asks to instantiate it.
func (c *C) With(o ...ClientOption) *C {
	c.options = append(c.options, o...)
	return c
}

// MiddlewareChain adds a RoundTripperChain as a client option
func (c *C) MiddlewareChain(rtc RoundTripperChain) *C {
	return c.With(func(client *http.Client) error {
		client.Transport = rtc.Then(client.Transport)
		return nil
	})
}

// Middleware adds several RoundTripperConstructors as a client option
func (c *C) Middleware(m ...RoundTripperConstructor) *C {
	return c.MiddlewareChain(
		NewRoundTripperChain(m...),
	)
}

// Inject allows additional components to be supplied to build an http.Client.
//
// Each dependency supplied via this method must be a struct value that embeds
// fx.In.  The embedding may be at any arbitrarily nested level.
//
// When unmarshaling occurs, these structs are injected via the normal usage
// of uber/fx.  Each field of each struct is examined to see if it can be converted
// into something that affects the construction of an http.Client.  Any field that
// cannot be converted is silently ignored, which allows a single struct to
// be used for more than one purpose.
func (c *C) Inject(deps ...interface{}) *C {
	for _, d := range deps {
		if dt, ok := arrange.IsIn(d); ok {
			c.dependencies = append(c.dependencies, dt)
		} else {
			c.errs = append(c.errs,
				fmt.Errorf("%s is not an fx.In struct", dt),
			)
		}
	}

	return c
}

// unmarshalFuncOf determines the function signature for Unmarshal or UnmarshalKey.
// The first input parameter is always a ClientIn struct.  Following that will be any
// fx.In structs, and following that will be any simple dependencies.
func (c *C) unmarshalFuncOf() reflect.Type {
	return reflect.FuncOf(
		// inputs
		append(
			[]reflect.Type{reflect.TypeOf(ClientIn{})},
			c.dependencies...,
		),

		// outputs
		[]reflect.Type{
			reflect.TypeOf((*http.Client)(nil)),
			arrange.ErrorType(),
		},

		false, // not variadic
	)
}

// unmarshal does all the heavy lifting of unmarshaling a ClientFactory and creating an *http.Client.
// If this method does not return an error, it will have bound the client's CloseIdleConnections
// to the shutdown of the enclosing fx.App.
func (c *C) unmarshal(u func(arrange.Unmarshaler, interface{}) error, inputs []reflect.Value) (client *http.Client, err error) {
	if len(c.errs) > 0 {
		err = multierr.Combine(c.errs...)
		return
	}

	var (
		target   = arrange.NewTarget(c.prototype)
		clientIn = inputs[0].Interface().(ClientIn)

		p = arrange.NewModulePrinter(Module, clientIn.Printer)
	)

	if clientIn.Unmarshaler != nil {
		if err = u(clientIn.Unmarshaler, target.UnmarshalTo.Interface()); err != nil {
			return
		}
	} else {
		p.Printf("CLIENT => No Unmarshaler supplied")
	}

	factory := target.Component.Interface().(ClientFactory)
	if client, err = factory.NewClient(); err != nil {
		return
	}

	var optionErrs []error
	for _, dependency := range inputs[1:] {
		arrange.VisitDependencies(
			dependency,
			func(f reflect.StructField, fv reflect.Value) bool {
				if arrange.IsInjected(f, fv) {
					// ignore dependencies that can't be converted
					if co := newClientOption(fv.Interface()); co != nil {
						p.Printf("CLIENT INJECT => %s.%s %s", dependency.Type(), f.Name, f.Tag)
						if err = co(client); err != nil {
							optionErrs = append(optionErrs, err)
						}
					}
				}

				return true
			},
		)
	}

	for _, o := range c.options {
		if err = o(client); err != nil {
			optionErrs = append(optionErrs, err)
		}
	}

	if len(optionErrs) > 0 {
		err = multierr.Combine(optionErrs...)
		client = nil
	} else {
		// if all went well, ensure that the client closes idle connections
		// when the fx.App is shutdown
		clientIn.Lifecycle.Append(fx.Hook{
			OnStop: func(context.Context) error {
				client.CloseIdleConnections()
				return nil
			},
		})
	}

	return
}

// makeUnmarshalFunc dynamically creates the function to be passed as a constructor to the fx.App.
func (c *C) makeUnmarshalFunc(u func(arrange.Unmarshaler, interface{}) error) reflect.Value {
	return reflect.MakeFunc(
		c.unmarshalFuncOf(),
		func(inputs []reflect.Value) []reflect.Value {
			client, err := c.unmarshal(u, inputs)
			return []reflect.Value{
				reflect.ValueOf(client),
				arrange.NewErrorValue(err),
			}
		},
	)
}

// Unmarshal creates a function at runtime that produces an *http.Client using this builder's
// current state.  The builder chain is terminated by this method, and the returned function
// may be passed to fx.Provide or used as the Target field in fx.Annotated.  Any errors that
// occur during construction of the *http.Client will short circuit the enclosing fx.App with an error.
//
// The signature of the returned function will always be "func(ClientIn, /* dependencies */) (*http.Client, error)".
// The dependencies will be each of the structs passed to Inject in order.  If no Inject dependencies
// were supplied, then the returned function will only accept a ClientIn as its sole input parameter.
func (c *C) Unmarshal() interface{} {
	return c.makeUnmarshalFunc(
		func(u arrange.Unmarshaler, v interface{}) error {
			return u.Unmarshal(v)
		},
	).Interface()
}

// Provide is the simplest way to leverage a client builder.  This method invokes Unmarshal
// and wraps it in an fx.Provide that can be directly passed to fx.New.
func (c *C) Provide() fx.Option {
	return fx.Provide(
		c.Unmarshal(),
	)
}

// UnmarshalKey is the same as Unmarshal, save that it unmarshals the ClientFactory from
// a specific configuration key.
func (c *C) UnmarshalKey(key string) interface{} {
	return c.makeUnmarshalFunc(
		func(u arrange.Unmarshaler, v interface{}) error {
			return u.UnmarshalKey(key, v)
		},
	).Interface()
}

// ProvideKey unmarshals from a given configuration key using UnmarshalKey.  It then returns
// an fx.Annotated with the key as the Name an the dynamic function created by UnmarshalKey
// as the Target.
func (c *C) ProvideKey(key string) fx.Option {
	return fx.Provide(
		fx.Annotated{
			Name:   key,
			Target: c.UnmarshalKey(key),
		},
	)
}
