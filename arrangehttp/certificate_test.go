package arrangehttp

import (
	"crypto/tls"
	"io/ioutil"
	"os"
	"testing"
)

// serverPrivateKey is a pregenerated RSA key for testing TLS connections
var serverPrivateKey []byte = []byte(`-----BEGIN RSA PRIVATE KEY-----
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
-----END RSA PRIVATE KEY-----`)

// serverCertificate is a pregenerated self-signed certificate for testing TLS connections
var serverCertificate []byte = []byte(`-----BEGIN CERTIFICATE-----
MIIE7DCCAtQCCQD+Cqp/QSxE4TANBgkqhkiG9w0BAQUFADA4MQswCQYDVQQGEwJV
UzELMAkGA1UECAwCQ0ExDTALBgNVBAoMBFRlc3QxDTALBgNVBAMMBFRlc3QwHhcN
MTkxMTE0MjAyMDIyWhcNMTkxMjE0MjAyMDIyWjA4MQswCQYDVQQGEwJVUzELMAkG
A1UECAwCQ0ExDTALBgNVBAoMBFRlc3QxDTALBgNVBAMMBFRlc3QwggIiMA0GCSqG
SIb3DQEBAQUAA4ICDwAwggIKAoICAQDbiGImOMkWMZbVh55Nc/mpMb7ucG9X94tP
ZhKi1rdd3DSlYwOaG1xF97pSoY2icxZ8t6CHflpN7qkdYaO9RLegM4iblNZsmlrX
122UEGIUiybiyVNfFbBugG4pFfjBq4MWrTolHur0BcOaxatwv4ZUdtz9OxRn7SsI
BCrAZccdc13EauHyNEmz7y9CNPUsQ8A/7Ce8M0DD/ov30Lb9raE8+RqHnUSnw+MV
cbxNuQpkqpBoXkTHHLn4/neFgARb6Kz3c00fwEKE1uv+iv9J01lgZzEniuEKQT57
eo/F2iSF3hDno9ZT2poLfHyngSdjBp21xJXWssJ0zF8+2KeBsXTTDcRUu4lAD3w+
ozBkuhv1ccK+iIheo1QWt/fBy1+JwxxUCYEZCpY/8MuLF88AZSSKTLEkY27vzt9t
AvGUQoGFcG3edPkr8icVkFU0TAlork/3e375pclbSG1axu/SMrLdYM5Qgz6JNaZe
ZpyQ+HzWyPbegUwiT/bo/nr14DaFRWsCVuHeYxMU4G83hAXySSzf6zp+TJH8d0sl
dChjTk7Vu6EoyCU0ZsfWicFTCV5pY7iRqgXdtwcMha4hYn+4SZM5CK1Tj5WHyK/m
pCiYZJ7TmjkemFlTdMlH5oJUso74b13lpsNDXIwpvSUtutO2bm/0lTXBbrBfQ0Ag
zg6RxJL08QIDAQABMA0GCSqGSIb3DQEBBQUAA4ICAQCwR7ujLTOk56u6RkbZQYAo
0PVvCYwi/KXvm5VDJzdmd2xDBp14Fk+M8LlO/2V56eeBp65U4+0oYoIDbFUJrtUi
NWCL22u5v8m3hAe5Go2a07SbqdcVkIrElK0APEqduUgrNh2Z2TVi2jWD9KykMNCZ
apZAIwfwaWJK2ppx9zX9Tk97HfjiojB+DMMOXYYZ2vQn3vEKup7eXBCMMGWubz7J
DsEuieHZ3ayEoI2sLv3mgJh0Nayl/zblDS+LqwbzdUsALAxgRFfhxEWBdlxR3/VW
qNa9Hxij6eRFbVXPnsdds+/Mal9YiMaEWlPLZEUYrVo9L0xL84M1t+yah6yJOMUh
Y5CAKNSNuwLd9tVmK/iPdmyXo2Nn95KFekeLXQAjX3oz7p+y4cAAcjCn7LczYgx5
hl2s/uvAjK9S3qmA1SX4cjpgbZ0lmDksbeS5q+NiKHk8lM2IsgqrObukPgwYvO/L
+q5H1NqDC/3+i5A9dY3XVaJ7R38kkVUytTQWvi63egvIx0eyMJY2BOoVrBPFh/g6
As1OW7O+KpHkBilY1e/bFUOBdEGaQFAc3lagzZxYvI2JAWyYkmVlXhamefEQ24fs
lXYEav2EZPgq1TS0ACvfgzbFhdPHUD7jY54qJMaRoiMuci0XlA1PJ9ovCFTcfLQh
gDSJZ0/AO8WcvMrPaDDv0Q==
-----END CERTIFICATE-----`)

// addServerCertificate configures a tls.Config to use the prebaked server cert and key.
// If the given tls.Config is nil, a new one is created.
func addServerCertificate(t *testing.T, in *tls.Config) *tls.Config {
	cert, err := tls.X509KeyPair(serverCertificate, serverPrivateKey)
	if err != nil {
		t.Fatalf("Unable to create X509 key pair: %s", err)
	}

	var out *tls.Config
	if in == nil {
		out = new(tls.Config)
	} else {
		out = in.Clone()
	}

	out.Certificates = append(out.Certificates, cert)
	return out
}

// createServerFiles creates a certificate file and a key file as temporary files.
// The prebaked key and certificate are used.
func createServerFiles(t *testing.T) (certificateFilePath, keyFilePath string) {
	certificateFile, err := ioutil.TempFile("", "server.*.cert")
	if err != nil {
		t.Fatalf("Unable to create server certificate file: %s", err)
	}

	certificateFilePath = certificateFile.Name()
	_, err = certificateFile.Write(serverCertificate)
	certificateFile.Close()
	if err != nil {
		os.Remove(certificateFilePath)
		t.Fatalf("Unable to write server certificate file '%s': %s", certificateFilePath, err)
	}

	keyFile, err := ioutil.TempFile("", "server.*.key")
	if err != nil {
		t.Fatalf("Unable to create server key file: %s", err)
	}

	keyFilePath = keyFile.Name()
	_, err = keyFile.Write(serverPrivateKey)
	keyFile.Close()
	if err != nil {
		os.Remove(certificateFilePath)
		os.Remove(keyFilePath)
		t.Fatalf("Unable to write server key file '%s': %s", keyFilePath, err)
	}

	return
}
