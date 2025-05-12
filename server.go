package main

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"sync"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	HttpsListenPort int    `envconfig:"HTTPS_PORT" default:"443"`
	HttpListenPort  int    `envconfig:"HTTP_PORT" default:"80"`
	TlsCert         string `envconfig:"TLS_CERT" default:"/usr/src/app/pki/tls.crt"`
	TlsKey          string `envconfig:"TLS_KEY" default:"/usr/src/app/pki/tls.key"`
	ServerTlsCert   string `envconfig:"SERVER_TLS_CERT" default:"/usr/src/app/pki/tls.crt"`
	ServerTlsKey    string `envconfig:"SERVER_TLS_KEY" default:"/usr/src/app/pki/tls.key"`
	ClientTlsCa     string `envconfig:"CLIENT_TLS_CA" default:""`
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	remoteAddress := r.RemoteAddr
	fwdAddress := r.Header.Get("X-Forwarded-For")
	resp := "Got %v request for %v from %v"
	if len(fwdAddress) > 0 {
		resp = fmt.Sprintf(resp+", forwarded by %v!\n", r.RequestURI, r.Host, fwdAddress, remoteAddress)
	} else {
		resp = fmt.Sprintf(resp, r.RequestURI, r.Host, remoteAddress)
	}
	fmt.Fprintf(w, resp)
}

func headerdumpHandler(w http.ResponseWriter, r *http.Request) {
	for k, v := range r.Header {
		fmt.Fprintf(w, "%v: %v\n", k, v)
	}
	// https://cs.opensource.google/go/go/+/refs/tags/go1.20.1:src/net/http/request.go;l=157-158;drc=fd0c0db4a411eae0483d1cb141e801af401e43d3
	fmt.Fprintf(w, "Host: [%v]\n", r.Host)
}

func healthCheckHandler(w http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(w, "ok\n")
}

func setupHandlers(mux *http.ServeMux) {
	mux.HandleFunc("/healthz", healthCheckHandler)
	mux.HandleFunc("/", rootHandler)
	mux.HandleFunc("/headers", headerdumpHandler)
}

func parseTlsConfig(c Config) *tls.Config {
	cer, err := tls.LoadX509KeyPair(c.TlsCert, c.TlsKey)
	if err != nil {
		log.Printf("Failed to load keypair [%s, %s]: %s", c.TlsCert, c.TlsKey, err)
		return nil
	}
	return &tls.Config{Certificates: []tls.Certificate{cer}, GetCertificate: returnCert(c)}
}

func returnCert(c Config) func(helloInfo *tls.ClientHelloInfo) (*tls.Certificate, error) {
	return func(helloInfo *tls.ClientHelloInfo) (*tls.Certificate, error) {
		log.Printf("SNI offered with ClientHelloInfo.ServerName=%v", helloInfo.ServerName)
		var cer tls.Certificate
		var err error
		if len(helloInfo.ServerName) > 0 {
			// if SNI is offered AND we have a certificate configured for it, load that.
			log.Printf("Loading pki pair [%s, %s]", c.ServerTlsCert, c.ServerTlsKey)
			cer, err = tls.LoadX509KeyPair(c.ServerTlsCert, c.ServerTlsKey)
			if err != nil {
				log.Printf("Failed to load keypair [%s, %s]: %s", c.ServerTlsCert, c.ServerTlsKey, err)
				return nil, nil
			}
			return &cer, nil
		} else {
			// if no sni is offered we return the fallback/default certificate:
			log.Printf("Loading pki pair [%s, %s]", c.TlsCert, c.TlsKey)
			cer, err = tls.LoadX509KeyPair(c.TlsCert, c.TlsKey)
			if err != nil {
				log.Printf("Failed to load keypair [%s, %s]: %s", c.TlsCert, c.TlsKey, err)
				return nil, nil
			}
			return &cer, nil
		}
		return nil, nil
	}
}

func newHttpServer(c Config) *http.Server {
	return &http.Server{
		Addr: ":" + strconv.Itoa(c.HttpListenPort),
	}
}

func newHttpsServer(c Config) *http.Server {
	if c.ClientTlsCa == "" {
		return &http.Server{
			Addr:      ":" + strconv.Itoa(c.HttpsListenPort),
			TLSConfig: &tls.Config{GetCertificate: returnCert(c)},
		}
	} else {
		log.Print("Requiring Client Certificate.")
		caCert, err := ioutil.ReadFile(c.ClientTlsCa)
		if err != nil {
			log.Fatal(err)
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)
		return &http.Server{
			Addr: ":" + strconv.Itoa(c.HttpsListenPort),
			TLSConfig: &tls.Config{
				GetCertificate: returnCert(c),
				ClientAuth:     tls.RequireAndVerifyClientCert,
				ClientCAs:      caCertPool,
			},
		}
	}
}

func main() {

	mux := http.NewServeMux()
	var config Config
	err := envconfig.Process("", &config)
	if err != nil {
		log.Fatal(err.Error())
	}
	setupHandlers(mux)

	httpsServer := newHttpsServer(config)
	httpsServer.Handler = mux
	httpServer := newHttpServer(config)
	httpServer.Handler = mux

	wg := new(sync.WaitGroup)
	wg.Add(2)

	go func() {
		log.Printf("Starting listening for incoming HTTPS requests on %v", httpsServer.Addr)
		err := httpsServer.ListenAndServeTLS("", "")
		if errors.Is(err, http.ErrServerClosed) {
			log.Printf("https responder closed\n")
		} else if err != nil {
			log.Printf("error listening for https: %s\n", err)
		}
		wg.Done()
	}()
	go func() {
		log.Printf("Starting listening for incoming HTTP requests on %v", httpServer.Addr)
		err := httpServer.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			log.Printf("http responder closed\n")
		} else if err != nil {
			log.Fatalf("error listening for http: %s\n", err)
		}
		wg.Done()
	}()
	wg.Wait()
}
