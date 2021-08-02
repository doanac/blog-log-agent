package client

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/ThalesIgnite/crypto11"
	toml "github.com/pelletier/go-toml"
)

func tomlGet(tree *toml.Tree, key string) string {
	val := tree.GetDefault(key, "").(string)
	if len(val) == 0 {
		fmt.Println("ERROR: Missing", key, "in sota.toml")
		os.Exit(1)
	}
	return val
}

func tomlAssertVal(tree *toml.Tree, key string, allowed []string) string {
	val := tomlGet(tree, key)
	for _, v := range allowed {
		if val == v {
			return val
		}
	}
	fmt.Println("ERROR: Invalid value", val, "in sota.toml for", key)
	return val
}

// sota.toml has slot id's as "01". We need to turn that into []byte{1}
func idToBytes(id string) []byte {
	bytes := []byte(id)
	start := -1
	for idx, char := range bytes {
		bytes[idx] = char - byte('0')
		if bytes[idx] != 0 && start == -1 {
			start = idx
		}
	}
	//strip off leading 0's
	return bytes[start:]
}

func createClientPkcs11(sota *toml.Tree) *http.Client {
	module := tomlGet(sota, "p11.module")
	pin := tomlGet(sota, "p11.pass")
	pkeyId := tomlGet(sota, "p11.tls_pkey_id")
	certId := tomlGet(sota, "p11.tls_clientcert_id")
	caFile := tomlGet(sota, "import.tls_cacert_path")

	cfg := crypto11.Config{
		Path:        module,
		TokenLabel:  "aktualizr",
		Pin:         pin,
		MaxSessions: 2,
	}

	ctx, err := crypto11.Configure(&cfg)
	if err != nil {
		log.Fatal(err)
	}

	privKey, err := ctx.FindKeyPair(idToBytes(pkeyId), nil)
	if err != nil {
		log.Fatal(err)
	}
	cert, err := ctx.FindCertificate(idToBytes(certId), nil, nil)
	if err != nil {
		log.Fatal(err)
	}
	if cert == nil || privKey == nil {
		log.Fatal("Unable to load pkcs11 client cert and/or private key")
	}

	caCert, err := ioutil.ReadFile(caFile)
	if err != nil {
		log.Fatal(err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{
			tls.Certificate{
				Certificate: [][]byte{cert.Raw},
				PrivateKey:  privKey,
			},
		},
		RootCAs: caCertPool,
	}
	transport := &http.Transport{TLSClientConfig: tlsConfig}
	client := &http.Client{Timeout: time.Second * 30, Transport: transport}
	return client
}

func createClientLocal(sota *toml.Tree) *http.Client {
	certFile := tomlGet(sota, "import.tls_clientcert_path")
	keyFile := tomlGet(sota, "import.tls_pkey_path")
	caFile := tomlGet(sota, "import.tls_cacert_path")

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		log.Fatal(err)
	}

	caCert, err := ioutil.ReadFile(caFile)
	if err != nil {
		log.Fatal(err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caCertPool,
	}
	transport := &http.Transport{TLSClientConfig: tlsConfig}
	return &http.Client{Timeout: time.Second * 30, Transport: transport}
}

func New(sota *toml.Tree) *http.Client {
	_ = tomlAssertVal(sota, "tls.ca_source", []string{"file"})
	source := tomlAssertVal(sota, "tls.pkey_source", []string{"file", "pkcs11"})
	_ = tomlAssertVal(sota, "tls.cert_source", []string{source})
	if source == "file" {
		return createClientLocal(sota)
	}
	return createClientPkcs11(sota)
}
