package arrangehttp

import (
	"errors"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmidt-org/arrange/arrangetls"
)

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

func testClientOptionsEmpty(t *testing.T) {
	assert := assert.New(t)
	assert.NoError(ClientOptions()(nil))
}

func testClientOptionsSuccess(t *testing.T) {
	for _, count := range []int{0, 1, 2, 5} {
		t.Run(strconv.Itoa(count), func(t *testing.T) {
			var (
				assert = assert.New(t)

				expectedClient = &http.Client{
					Timeout: 125 * time.Minute,
				}

				options       []ClientOption
				expectedOrder []int
				actualOrder   []int
			)

			for i := 0; i < count; i++ {
				expectedOrder = append(expectedOrder, i)

				i := i
				options = append(options, func(actualClient *http.Client) error {
					assert.Equal(expectedClient, actualClient)
					actualOrder = append(actualOrder, i)
					return nil
				})
			}

			assert.NoError(
				ClientOptions(options...)(expectedClient),
			)

			assert.Equal(expectedOrder, actualOrder)
		})
	}
}

func testClientOptionsFailure(t *testing.T) {
	var (
		assert = assert.New(t)

		expectedClient = &http.Client{
			Timeout: 45 * time.Second,
		}

		expectedErr = errors.New("expected")
		firstCalled bool

		co = ClientOptions(
			func(actualClient *http.Client) error {
				firstCalled = true
				assert.Equal(expectedClient, actualClient)
				return nil
			},
			func(actualClient *http.Client) error {
				assert.Equal(expectedClient, actualClient)
				return expectedErr
			},
			func(actualClient *http.Client) error {
				assert.Fail("This option should not have been called")
				return errors.New("This option should not have been called")
			},
		)
	)

	assert.Equal(
		expectedErr,
		co(expectedClient),
	)

	assert.True(firstCalled)
}

func TestClientOptions(t *testing.T) {
	t.Run("Empty", testClientOptionsEmpty)
	t.Run("Success", testClientOptionsSuccess)
	t.Run("Failure", testClientOptionsFailure)
}

func TestClient(t *testing.T) {
}
