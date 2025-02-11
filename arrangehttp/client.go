package arrangehttp

import (
	"errors"
	"net/http"

	"github.com/xmidt-org/arrange"
	"go.uber.org/fx"
)

var (
	// ErrClientNameRequired indicates that ProvideClient or ProvideClientCustom was called
	// with an empty client name.
	ErrClientNameRequired = errors.New("A client name is required")
)

// RoundTripperFunc is a function type that implements http.RoundTripper.  Useful
// when building client middleware.
type RoundTripperFunc func(*http.Request) (*http.Response, error)

func (rtf RoundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return rtf(request)
}

// ApplyClientOptions executes options against a client.  The original client is returned, along
// with any error(s) that occurred.  All options are executed, so the returned error may be an
// aggregate error which can be inspected via go.uber.org/multierr.
//
// This function can be used as an fx decorator for a client within the enclosing application.
func ApplyClientOptions(client *http.Client, opts ...ClientOption) (*http.Client, error) {
	err := ClientOptions(opts).ApplyToClient(client)
	return client, err
}

// NewClient is the primary client constructor for arrange.  Use this when you are creating a client
// from a (possibly unmarshaled) ClientConfig.  The options can be annotated to come from a value group,
// which is useful when there are multiple clients in a single fx.App.
func NewClient(cc ClientConfig, opts ...ClientOption) (*http.Client, error) {
	return NewClientCustom(cc, opts...)
}

// NewClientCustom is an *http.Client constructor that allows customization of the concrete
// ClientFactory used to create the *http.Client.  This function is useful when you have a
// custom (possibly unmarshaled) configuration struct that implements ClientFactory.
func NewClientCustom[F ClientFactory](cf F, opts ...ClientOption) (c *http.Client, err error) {
	c, err = cf.NewClient()
	if err == nil {
		c, err = ApplyClientOptions(c, opts...)
	}

	return
}

// ProvideClient assembles a client out of application components in a standard, opinionated way.
// The clientName parameter is used as both the name of the *http.Client component and a prefix
// for that server's dependencies:
//
//   - NewClient is used to create the client as a component named clientName
//   - ClientConfig is an optional dependency with the name clientName+".config"
//   - []ClientOption is an value group dependency with the name clientName+".options"
//
// The external set of options, if supplied, is applied to the client after any injected options.
// This allows for options that come from outside the enclosing fx.App, as might be the case
// for options driven by the command line.
func ProvideClient(clientName string, external ...ClientOption) fx.Option {
	return ProvideClientCustom[ClientConfig](clientName, external...)
}

// ProvideClientCustom is like ProvideClient, but it allows customization of the concrete
// ClientFactory dependency.
func ProvideClientCustom[F ClientFactory](clientName string, external ...ClientOption) fx.Option {
	if len(clientName) == 0 {
		return fx.Error(ErrClientNameRequired)
	}

	return fx.Provide(
		fx.Annotate(
			NewClientCustom[F],
			arrange.Tags().
				OptionalName(clientName+".config").
				Group(clientName+".options").
				ParamTags(),
			arrange.Tags().Name(clientName).ResultTags(),
		),
	)
}
