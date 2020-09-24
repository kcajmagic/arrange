package arrangehttp

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmidt-org/arrange"
	"github.com/xmidt-org/arrange/arrangetls"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

type TestServerMiddlewareChain []func(http.Handler) http.Handler

func (tsmc TestServerMiddlewareChain) Then(next http.Handler) http.Handler {
	for i := len(tsmc) - 1; i >= 0; i-- {
		next = tsmc[i](next)
	}

	return next
}

type simpleServerFactory struct {
	Address   string
	returnErr error
}

func (ssf simpleServerFactory) NewServer() (*http.Server, error) {
	if ssf.returnErr != nil {
		return nil, ssf.returnErr
	}

	return &http.Server{
		Addr: ssf.Address,
	}, nil
}

func testServerConfigBasic(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		serverConfig = ServerConfig{
			Address:           ":0",
			ReadTimeout:       15 * time.Second,
			ReadHeaderTimeout: 27 * time.Minute,
			WriteTimeout:      38 * time.Second,
			IdleTimeout:       89 * time.Minute,
			MaxHeaderBytes:    478934,
			KeepAlive:         16 * time.Minute,
		}

		router  = mux.NewRouter()
		address = make(chan net.Addr, 1)
	)

	server, err := serverConfig.NewServer()
	require.NoError(err)
	require.NotNil(server)

	assert.Equal(15*time.Second, server.ReadTimeout)
	assert.Equal(27*time.Minute, server.ReadHeaderTimeout)
	assert.Equal(38*time.Second, server.WriteTimeout)
	assert.Equal(89*time.Minute, server.IdleTimeout)
	assert.Equal(478934, server.MaxHeaderBytes)

	// check that this is a functioning server
	lf := NewListenerChain(CaptureListenAddress(address)).
		Factory(DefaultListenerFactory{})
	server.Handler = router
	require.NoError(
		ServerOnStart(server, lf)(context.Background()),
	)

	defer server.Close()

	select {
	case listenAddress := <-address:
		conn, err := net.Dial("tcp", listenAddress.String())
		require.NoError(err)
		defer conn.Close()

		fmt.Fprintf(conn, "GET / HTTP/1.0\r\n\r\n")
		_, err = bufio.NewReader(conn).ReadString('\n')
		require.NoError(err)

	case <-time.After(2 * time.Second):
		assert.Fail("No captured listen address")
	}
}

func testServerConfigTLS(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		serverConfig = ServerConfig{
			Address:           ":0",
			ReadTimeout:       72 * time.Second,
			ReadHeaderTimeout: 109 * time.Minute,
			WriteTimeout:      63 * time.Second,
			IdleTimeout:       9234 * time.Minute,
			MaxHeaderBytes:    3642,
			KeepAlive:         3 * time.Minute,
			TLS: &arrangetls.Config{
				Certificates: arrangetls.ExternalCertificates{
					{
						CertificateFile: CertificateFile,
						KeyFile:         KeyFile,
					},
				},
			},
		}

		router  = mux.NewRouter()
		address = make(chan net.Addr, 1)
	)

	server, err := serverConfig.NewServer()
	require.NoError(err)
	require.NotNil(server)

	assert.Equal(72*time.Second, server.ReadTimeout)
	assert.Equal(109*time.Minute, server.ReadHeaderTimeout)
	assert.Equal(63*time.Second, server.WriteTimeout)
	assert.Equal(9234*time.Minute, server.IdleTimeout)
	assert.Equal(3642, server.MaxHeaderBytes)

	// check that this is a functioning server
	lf := NewListenerChain(CaptureListenAddress(address)).
		Factory(DefaultListenerFactory{})
	server.Handler = router
	require.NoError(
		ServerOnStart(server, lf)(context.Background()),
	)

	defer server.Close()

	select {
	case listenAddress := <-address:
		conn, err := net.Dial("tcp", listenAddress.String())
		require.NoError(err)
		defer conn.Close()

		fmt.Fprintf(conn, "GET / HTTP/1.0\r\n\r\n")
		_, err = bufio.NewReader(conn).ReadString('\n')
		require.NoError(err)

	case <-time.After(2 * time.Second):
		assert.Fail("No captured listen address")
	}
}

func TestServerConfig(t *testing.T) {
	t.Run("Basic", testServerConfigBasic)
	t.Run("TLS", testServerConfigTLS)
}

func testServerOptionsEmpty(t *testing.T) {
	assert := assert.New(t)
	assert.NoError(ServerOptions()(nil))
}

func testServerOptionsSuccess(t *testing.T) {
	for _, count := range []int{0, 1, 2, 5} {
		t.Run(strconv.Itoa(count), func(t *testing.T) {
			var (
				assert = assert.New(t)

				expectedServer = &http.Server{
					Addr: ":123",
				}

				options       []ServerOption
				expectedOrder []int
				actualOrder   []int
			)

			for i := 0; i < count; i++ {
				expectedOrder = append(expectedOrder, i)

				i := i
				options = append(options, func(actualServer *http.Server) error {
					assert.Equal(expectedServer, actualServer)
					actualOrder = append(actualOrder, i)
					return nil
				})
			}

			assert.NoError(
				ServerOptions(options...)(expectedServer),
			)

			assert.Equal(expectedOrder, actualOrder)
		})
	}
}

func testServerOptionsFailure(t *testing.T) {
	var (
		assert = assert.New(t)

		expectedServer = &http.Server{
			Addr: ":456",
		}

		expectedErr = errors.New("expected")
		firstCalled bool

		so = ServerOptions(
			func(actualServer *http.Server) error {
				firstCalled = true
				assert.Equal(expectedServer, actualServer)
				return nil
			},
			func(actualServer *http.Server) error {
				assert.Equal(expectedServer, actualServer)
				return expectedErr
			},
			func(actualServer *http.Server) error {
				assert.Fail("This option should not have been called")
				return errors.New("This option should not have been called")
			},
		)
	)

	assert.Equal(
		expectedErr,
		so(expectedServer),
	)

	assert.True(firstCalled)
}

func TestServerOptions(t *testing.T) {
	t.Run("Empty", testServerOptionsEmpty)
	t.Run("Success", testServerOptionsSuccess)
	t.Run("Failure", testServerOptionsFailure)
}

func testRouterOptionsEmpty(t *testing.T) {
	assert := assert.New(t)
	assert.NoError(RouterOptions()(nil))
}

func testRouterOptionsSuccess(t *testing.T) {
	for _, count := range []int{0, 1, 2, 5} {
		t.Run(strconv.Itoa(count), func(t *testing.T) {
			var (
				assert = assert.New(t)

				expectedRouter = mux.NewRouter()

				options       []RouterOption
				expectedOrder []int
				actualOrder   []int
			)

			for i := 0; i < count; i++ {
				expectedOrder = append(expectedOrder, i)

				i := i
				options = append(options, func(actualRouter *mux.Router) error {
					assert.Equal(expectedRouter, actualRouter)
					actualOrder = append(actualOrder, i)
					return nil
				})
			}

			assert.NoError(
				RouterOptions(options...)(expectedRouter),
			)

			assert.Equal(expectedOrder, actualOrder)
		})
	}
}

func testRouterOptionsFailure(t *testing.T) {
	var (
		assert = assert.New(t)

		expectedRouter = mux.NewRouter()

		expectedErr = errors.New("expected")
		firstCalled bool

		ro = RouterOptions(
			func(actualRouter *mux.Router) error {
				firstCalled = true
				assert.Equal(expectedRouter, actualRouter)
				return nil
			},
			func(actualRouter *mux.Router) error {
				assert.Equal(expectedRouter, actualRouter)
				return expectedErr
			},
			func(actualRouter *mux.Router) error {
				assert.Fail("This option should not have been called")
				return errors.New("This option should not have been called")
			},
		)
	)

	assert.Equal(
		expectedErr,
		ro(expectedRouter),
	)

	assert.True(firstCalled)
}

func TestRouterOptions(t *testing.T) {
	t.Run("Empty", testRouterOptionsEmpty)
	t.Run("Success", testRouterOptionsSuccess)
	t.Run("Failure", testRouterOptionsFailure)
}

func testServerInjectError(t *testing.T) {
	var (
		assert = assert.New(t)
		v      = viper.New()
	)

	app := fx.New(
		arrange.TestLogger(t),
		arrange.ForViper(v),
		Server().
			Inject(struct {
				DoesNotEmbedFxIn string
			}{}).
			Provide(),
		fx.Invoke(
			func(*mux.Router) {
				// doesn't matter
			},
		),
	)

	assert.Error(app.Err())
}

func testServerUnmarshalError(t *testing.T) {
	const yaml = `
maxHeaderBytes: "this is not a valid int"
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
		Server().
			Provide(),
		fx.Invoke(
			func(*mux.Router) {
				// doesn't matter
			},
		),
	)

	assert.Error(app.Err())
}

func testServerFactoryError(t *testing.T) {
	var (
		assert = assert.New(t)
		v      = viper.New()
	)

	app := fx.New(
		arrange.TestLogger(t),
		arrange.ForViper(v),
		Server().
			ServerFactory(simpleServerFactory{
				returnErr: errors.New("expected NewServer error"),
			}).
			Provide(),
		fx.Invoke(
			func(*mux.Router) {
				// doesn't matter
			},
		),
	)

	assert.Error(app.Err())
}

func testServerOptionError(t *testing.T) {
	var (
		assert = assert.New(t)
		v      = viper.New()

		injectedServerOptionCalled bool
		injectedRouterOptionCalled bool
		externalServerOptionCalled bool
		externalRouterOptionCalled bool
	)

	app := fx.New(
		arrange.TestLogger(t),
		arrange.ForViper(v),
		fx.Provide(
			func() ServerOption {
				return func(s *http.Server) error {
					assert.NotNil(s)
					injectedServerOptionCalled = true
					return errors.New("expected ServerOption error")
				}
			},
			func() RouterOption {
				return func(r *mux.Router) error {
					assert.NotNil(r)
					injectedRouterOptionCalled = true
					return errors.New("expected RouterOption error")
				}
			},
		),
		Server().
			With(func(s *http.Server) error {
				assert.NotNil(s)
				externalServerOptionCalled = true
				return errors.New("expected ServerOption error")
			}).
			WithRouter(func(r *mux.Router) error {
				assert.NotNil(r)
				externalRouterOptionCalled = true
				return errors.New("expected RouterOption error")
			}).
			Inject(struct {
				fx.In
				O1 ServerOption
				O2 RouterOption
			}{}).
			Provide(),
		fx.Invoke(
			func(*mux.Router) {
				// doesn't matter
			},
		),
	)

	assert.Error(app.Err())
	assert.True(injectedServerOptionCalled)
	assert.True(injectedRouterOptionCalled)
	assert.True(externalServerOptionCalled)
	assert.True(externalRouterOptionCalled)
}

func testServerDefaultListenerFactory(t *testing.T) {
	var (
		v       = viper.New()
		address = make(chan net.Addr, 1)
	)

	app := fxtest.New(
		t,
		arrange.TestLogger(t),
		arrange.ForViper(v),
		Server().
			// this ServerFactory does not implement ListenerFactory, thus
			// forcing the builder to use the default
			ServerFactory(simpleServerFactory{}).
			CaptureListenAddress(address).
			Provide(),
		fx.Invoke(
			func(*mux.Router) {},
		),
	)

	app.RequireStart()
	defer app.Stop(context.Background())

	MustGetListenAddress(address, time.After(time.Second))
	app.RequireStop()
}

func testServerMiddleware(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		v       = viper.New()
		address = make(chan net.Addr, 1)
	)

	app := fxtest.New(
		t,
		arrange.TestLogger(t),
		arrange.ForViper(v),
		fx.Provide(
			func() mux.MiddlewareFunc {
				return func(next http.Handler) http.Handler {
					return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
						response.Header().Set("Injected-Middleware", "true")
						next.ServeHTTP(response, request)
					})
				}
			},
			func() TestServerMiddlewareChain {
				return TestServerMiddlewareChain{
					func(next http.Handler) http.Handler {
						return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
							response.Header().Set("Injected-Chain-Middleware", "true")
							next.ServeHTTP(response, request)
						})
					},
				}
			},
		),
		Server().
			Middleware(
				func(next http.Handler) http.Handler {
					return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
						response.Header().Set("Local-Middleware", "true")
						next.ServeHTTP(response, request)
					})
				},
			).
			MiddlewareChain(
				TestServerMiddlewareChain{
					func(next http.Handler) http.Handler {
						return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
							response.Header().Set("Chain-Middleware", "true")
							next.ServeHTTP(response, request)
						})
					},
				},
			).
			Inject(struct {
				fx.In
				M  mux.MiddlewareFunc
				MC TestServerMiddlewareChain
			}{}).
			CaptureListenAddress(address).
			Provide(),
		fx.Invoke(
			func(r *mux.Router) {
				r.HandleFunc("/test", func(response http.ResponseWriter, request *http.Request) {
					response.WriteHeader(267)
				})
			},
		),
	)

	require.NoError(app.Err())
	app.RequireStart()
	defer app.Stop(context.Background())

	serverURL := "http://" + MustGetListenAddress(address, time.After(time.Second)).String()
	response, err := http.Get(serverURL + "/test")
	require.NoError(err)
	require.NotNil(response)
	assert.Equal(267, response.StatusCode)
	assert.Equal("true", response.Header.Get("Local-Middleware"))
	assert.Equal("true", response.Header.Get("Chain-Middleware"))
	assert.Equal("true", response.Header.Get("Injected-Middleware"))
	assert.Equal("true", response.Header.Get("Injected-Chain-Middleware"))

	app.RequireStop()
}

func testServerOptions(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		v       = viper.New()
		address = make(chan net.Addr, 1)

		injectedServerOptionCalled  bool
		injectedServerOptionsCalled bool
		injectedRouterOptionCalled  bool
		injectedRouterOptionsCalled bool

		externalServerOptionCalled bool
		externalRouterOptionCalled bool
	)

	app := fxtest.New(
		t,
		arrange.TestLogger(t),
		arrange.ForViper(v),
		fx.Provide(
			fx.Annotated{
				Name: "serverOption",
				Target: func() ServerOption {
					return func(s *http.Server) error {
						assert.NotNil(s)
						injectedServerOptionCalled = true
						return nil
					}
				},
			},
			fx.Annotated{
				Name: "serverOptions",
				Target: func() ServerOption {
					return ServerOptions(
						func(s *http.Server) error {
							assert.NotNil(s)
							injectedServerOptionsCalled = true
							return nil
						},
					)
				},
			},
			fx.Annotated{
				Name: "routerOption",
				Target: func() RouterOption {
					return func(r *mux.Router) error {
						assert.NotNil(r)
						injectedRouterOptionCalled = true
						return nil
					}
				},
			},
			fx.Annotated{
				Name: "routerOptions",
				Target: func() RouterOption {
					return RouterOptions(
						func(r *mux.Router) error {
							assert.NotNil(r)
							injectedRouterOptionsCalled = true
							return nil
						},
					)
				},
			},
		),
		Server().
			Inject(struct {
				fx.In
				O1 ServerOption `name:"serverOption"`
				O2 ServerOption `name:"serverOptions"`
				O3 RouterOption `name:"routerOption"`
				O4 RouterOption `name:"routerOptions"`
			}{}).
			With(
				func(s *http.Server) error {
					assert.NotNil(s)
					externalServerOptionCalled = true
					return nil
				},
			).
			WithRouter(
				func(r *mux.Router) error {
					assert.NotNil(r)
					externalRouterOptionCalled = true
					return nil
				},
			).
			CaptureListenAddress(address).
			Provide(),
		fx.Invoke(
			func(r *mux.Router) {
				r.HandleFunc("/test", func(response http.ResponseWriter, request *http.Request) {
					response.WriteHeader(287)
				})
			},
		),
	)

	require.NoError(app.Err())
	app.RequireStart()
	defer app.Stop(context.Background())

	assert.True(injectedServerOptionCalled)
	assert.True(injectedServerOptionsCalled)
	assert.True(injectedRouterOptionCalled)
	assert.True(injectedRouterOptionsCalled)
	assert.True(externalServerOptionCalled)
	assert.True(externalRouterOptionCalled)

	serverURL := "http://" + MustGetListenAddress(address, time.After(time.Second)).String()
	response, err := http.Get(serverURL + "/test")
	require.NoError(err)
	require.NotNil(response)
	assert.Equal(287, response.StatusCode)

	app.RequireStop()
}

func testServerListener(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		v       = viper.New()
		address = make(chan net.Addr, 1)

		injectedListenerConstructorCalled bool
		injectedListenerChainCalled       bool

		externalListenerConstructorCalled bool
		externalListenerChainCalled       bool
	)

	app := fxtest.New(
		t,
		arrange.TestLogger(t),
		arrange.ForViper(v),
		fx.Provide(
			func() ListenerConstructor {
				return func(next net.Listener) net.Listener {
					assert.NotNil(next)
					injectedListenerConstructorCalled = true
					return next
				}
			},
			func() ListenerChain {
				return NewListenerChain(
					func(next net.Listener) net.Listener {
						assert.NotNil(next)
						injectedListenerChainCalled = true
						return next
					},
				)
			},
		),
		Server().
			Inject(struct {
				fx.In
				LC1 ListenerConstructor
				LC2 ListenerChain
			}{}).
			CaptureListenAddress(address).
			ListenerConstructors(func(next net.Listener) net.Listener {
				assert.NotNil(next)
				externalListenerConstructorCalled = true
				return next
			}).
			ListenerChain(
				NewListenerChain(
					func(next net.Listener) net.Listener {
						assert.NotNil(next)
						externalListenerChainCalled = true
						return next
					},
				),
			).
			Provide(),
		fx.Invoke(
			func(r *mux.Router) {
				r.HandleFunc("/test", func(response http.ResponseWriter, request *http.Request) {
					response.WriteHeader(216)
				})
			},
		),
	)

	require.NoError(app.Err())
	app.RequireStart()
	defer app.Stop(context.Background())

	assert.True(injectedListenerConstructorCalled)
	assert.True(injectedListenerChainCalled)
	assert.True(externalListenerConstructorCalled)
	assert.True(externalListenerChainCalled)

	serverURL := "http://" + MustGetListenAddress(address, time.After(time.Second)).String()
	response, err := http.Get(serverURL + "/test")
	require.NoError(err)
	require.NotNil(response)
	assert.Equal(216, response.StatusCode)

	app.RequireStop()
}

func TestServer(t *testing.T) {
	t.Run("InjectError", testServerInjectError)
	t.Run("UnmarshalError", testServerUnmarshalError)
	t.Run("FactoryError", testServerFactoryError)
	t.Run("OptionError", testServerOptionError)
	t.Run("DefaultListenerFactory", testServerDefaultListenerFactory)
	t.Run("Middleware", testServerMiddleware)
	t.Run("Options", testServerOptions)
	t.Run("Listener", testServerListener)
}
