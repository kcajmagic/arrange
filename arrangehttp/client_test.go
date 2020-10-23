package arrangehttp

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmidt-org/arrange"
	"github.com/xmidt-org/arrange/arrangetls"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

type badClientFactory struct{}

func (bcf badClientFactory) NewClient() (*http.Client, error) {
	return nil, errors.New("expected NewClient error")
}

func testTransportConfigBasic(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		tc = TransportConfig{
			TLSHandshakeTimeout:   15 * time.Second,
			DisableKeepAlives:     true,
			DisableCompression:    true,
			MaxIdleConns:          17,
			MaxIdleConnsPerHost:   5,
			MaxConnsPerHost:       92,
			IdleConnTimeout:       2 * time.Minute,
			ResponseHeaderTimeout: 13 * time.Millisecond,
			ExpectContinueTimeout: 29 * time.Second,
			ProxyConnectHeader: http.Header{
				"Something": []string{"Of Value"},
			},
			MaxResponseHeaderBytes: 347234,
			WriteBufferSize:        234867,
			ReadBufferSize:         93247,
			ForceAttemptHTTP2:      true,
		}
	)

	transport, err := tc.NewTransport(nil)
	require.NoError(err)
	require.NotNil(transport)

	assert.Nil(transport.TLSClientConfig)
	assert.Equal(15*time.Second, transport.TLSHandshakeTimeout)
	assert.True(transport.DisableKeepAlives)
	assert.True(transport.DisableCompression)
	assert.Equal(17, transport.MaxIdleConns)
	assert.Equal(5, transport.MaxIdleConnsPerHost)
	assert.Equal(92, transport.MaxConnsPerHost)
	assert.Equal(2*time.Minute, transport.IdleConnTimeout)
	assert.Equal(13*time.Millisecond, transport.ResponseHeaderTimeout)
	assert.Equal(29*time.Second, transport.ExpectContinueTimeout)
	assert.Equal(
		http.Header{"Something": []string{"Of Value"}},
		transport.ProxyConnectHeader,
	)
	assert.Equal(int64(347234), transport.MaxResponseHeaderBytes)
	assert.Equal(234867, transport.WriteBufferSize)
	assert.Equal(93247, transport.ReadBufferSize)
	assert.True(transport.ForceAttemptHTTP2)
}

func testTransportConfigTLS(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		tc TransportConfig

		config = arrangetls.Config{
			InsecureSkipVerify: true,
		}
	)

	transport, err := tc.NewTransport(&config)
	require.NoError(err)
	require.NotNil(transport)
	assert.NotNil(transport.TLSClientConfig)
}

func testTransportConfigError(t *testing.T) {
	var (
		assert = assert.New(t)

		tc TransportConfig

		config = arrangetls.Config{
			Certificates: arrangetls.ExternalCertificates{
				{
					CertificateFile: "missing",
					KeyFile:         "missing",
				},
			},
		}
	)

	transport, err := tc.NewTransport(&config)
	assert.Error(err)
	assert.NotNil(transport)
}

func TestTransportConfig(t *testing.T) {
	t.Run("Basic", testTransportConfigBasic)
	t.Run("TLS", testTransportConfigTLS)
	t.Run("Error", testTransportConfigError)
}

func testClientConfigBasic(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		cc = ClientConfig{
			Timeout: 15 * time.Second,
		}
	)

	client, err := cc.NewClient()
	require.NoError(err)
	require.NotNil(client)

	assert.Equal(15*time.Second, client.Timeout)
}

func testClientConfigError(t *testing.T) {
	var (
		assert = assert.New(t)

		cc = ClientConfig{
			TLS: &arrangetls.Config{
				Certificates: arrangetls.ExternalCertificates{
					{
						CertificateFile: "missing",
						KeyFile:         "missing",
					},
				},
			},
		}
	)

	_, err := cc.NewClient()
	assert.Error(err)
}

func TestClientConfig(t *testing.T) {
	t.Run("Basic", testClientConfigBasic)
	t.Run("Error", testClientConfigError)
}

func testClientInjectError(t *testing.T) {
	var (
		assert = assert.New(t)
		v      = viper.New()
	)

	app := fx.New(
		arrange.TestLogger(t),
		arrange.ForViper(v),
		Client().
			Inject(struct {
				DoesNotEmbedFxIn string
			}{}).
			Provide(),
		fx.Invoke(
			func(*http.Client) {},
		),
	)

	assert.Error(app.Err())
}

func testClientUnmarshalError(t *testing.T) {
	const yaml = `
timeout: "this is not a valid time.Duration"
`

	var (
		assert  = assert.New(t)
		require = require.New(t)
		v       = viper.New()
	)

	v.SetConfigType("yaml")
	require.NoError(v.ReadConfig(strings.NewReader(yaml)))

	app := fx.New(
		arrange.TestLogger(t),
		arrange.ForViper(v),
		Client().
			Provide(),
		fx.Invoke(
			func(*http.Client) {},
		),
	)

	assert.Error(app.Err())
}

func testClientFactoryError(t *testing.T) {
	var (
		assert = assert.New(t)
		v      = viper.New()
	)

	app := fx.New(
		arrange.TestLogger(t),
		arrange.ForViper(v),
		Client().
			ClientFactory(badClientFactory{}).
			Provide(),
		fx.Invoke(
			func(*http.Client) {},
		),
	)

	assert.Error(app.Err())
}

func testClientOptionError(t *testing.T) {
	var (
		assert = assert.New(t)
		v      = viper.New()

		injectedClientOptionCalled bool
		externalClientOptionCalled bool
	)

	app := fx.New(
		arrange.TestLogger(t),
		arrange.ForViper(v),
		fx.Provide(
			func() ClientOption {
				return func(c *http.Client) error {
					assert.NotNil(c)
					injectedClientOptionCalled = true
					return errors.New("expected ClientOption error")
				}
			},
		),
		Client().
			With(func(c *http.Client) error {
				assert.NotNil(c)
				externalClientOptionCalled = true
				return errors.New("expected ClientOption error")
			}).
			Inject(struct {
				fx.In
				O1 ClientOption
			}{}).
			Provide(),
		fx.Invoke(
			func(*http.Client) {},
		),
	)

	assert.Error(app.Err())
	assert.True(injectedClientOptionCalled)
	assert.True(externalClientOptionCalled)
}

func testClientMiddleware(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		v      = viper.New()
		client *http.Client
	)

	app := fxtest.New(
		t,
		arrange.TestLogger(t),
		arrange.ForViper(v),
		fx.Provide(
			func() RoundTripperConstructor {
				return func(next http.RoundTripper) http.RoundTripper {
					return RoundTripperFunc(func(request *http.Request) (*http.Response, error) {
						request.Header.Set("Injected-Middleware", "true")
						return next.RoundTrip(request)
					})
				}
			},
			func() RoundTripperChain {
				return NewRoundTripperChain(
					func(next http.RoundTripper) http.RoundTripper {
						return RoundTripperFunc(func(request *http.Request) (*http.Response, error) {
							request.Header.Set("Injected-Middleware-Chain", "true")
							return next.RoundTrip(request)
						})
					},
				)
			},
		),
		Client().
			Inject(struct {
				fx.In
				M1 RoundTripperConstructor
				M2 RoundTripperChain
			}{}).
			Middleware(
				func(next http.RoundTripper) http.RoundTripper {
					return RoundTripperFunc(func(request *http.Request) (*http.Response, error) {
						request.Header.Set("External-Middleware", "true")
						return next.RoundTrip(request)
					})
				},
			).
			MiddlewareChain(
				NewRoundTripperChain(
					func(next http.RoundTripper) http.RoundTripper {
						return RoundTripperFunc(func(request *http.Request) (*http.Response, error) {
							request.Header.Set("External-Middleware-Chain", "true")
							return next.RoundTrip(request)
						})
					},
				),
			).
			Provide(),
		fx.Populate(&client),
	)

	require.NoError(app.Err())
	app.RequireStart()
	defer app.Stop(context.Background())

	server := httptest.NewServer(
		http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			assert.Equal("/test", request.RequestURI)
			assert.Equal("true", request.Header.Get("Injected-Middleware"))
			assert.Equal("true", request.Header.Get("Injected-Middleware-Chain"))
			assert.Equal("true", request.Header.Get("External-Middleware"))
			assert.Equal("true", request.Header.Get("External-Middleware-Chain"))
			response.WriteHeader(211)
		}),
	)

	defer server.Close()

	request, err := http.NewRequest("GET", server.URL+"/test", nil)
	require.NoError(err)

	response, err := client.Do(request)
	require.NoError(err)
	require.NotNil(response)
	assert.Equal(211, response.StatusCode)

	app.RequireStop()
}

func testClientHeader(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		client *http.Client
	)

	app := fxtest.New(
		t,
		arrange.TestLogger(t),
		Client().
			ClientFactory(ClientConfig{
				Header: http.Header{
					"test1": {"true"},
					"test2": {"1", "2"},
				},
			}).
			Provide(),
		fx.Populate(&client),
	)

	require.NoError(app.Err())
	app.RequireStart()
	defer app.Stop(context.Background())

	server := httptest.NewServer(
		http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			assert.Equal("/test", request.RequestURI)
			assert.Equal([]string{"true"}, request.Header["Test1"])
			assert.Equal([]string{"1", "2"}, request.Header["Test2"])
			response.WriteHeader(258)
		}),
	)

	defer server.Close()

	request, err := http.NewRequest("GET", server.URL+"/test", nil)
	require.NoError(err)

	response, err := client.Do(request)
	require.NoError(err)
	require.NotNil(response)
	assert.Equal(258, response.StatusCode)

	app.RequireStop()
}

func testClientNoUnmarshaler(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		client *http.Client
	)

	app := fxtest.New(
		t,
		arrange.TestLogger(t),
		// no ForViper call
		Client().
			ClientFactory(ClientConfig{
				Timeout: 17 * time.Hour,
			}).
			Provide(),
		fx.Populate(&client),
	)

	require.NoError(app.Err())
	app.RequireStart()
	defer app.Stop(context.Background())

	require.NotNil(client)
	assert.Equal(17*time.Hour, client.Timeout)

	app.RequireStop()
}

func TestClient(t *testing.T) {
	t.Run("InjectError", testClientInjectError)
	t.Run("UnmarshalError", testClientUnmarshalError)
	t.Run("FactoryError", testClientFactoryError)
	t.Run("OptionError", testClientOptionError)
	t.Run("Middleware", testClientMiddleware)
	t.Run("Header", testClientHeader)
	t.Run("NoUnmarshaler", testClientNoUnmarshaler)
}
