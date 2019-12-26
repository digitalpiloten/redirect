package main

import (
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/mholt/certmagic"
	"github.com/securityclippy/magicstorage"
)

func fallback(w http.ResponseWriter, r *http.Request, reason string) {
	location := "https://redirect.name/"
	if reason != "" {
		location = fmt.Sprintf("%s#reason=%s", location, url.QueryEscape(reason))
	}
	http.Redirect(w, r, location, 302)
}

func getRedirect(txt []string, url string) (*Redirect, error) {
	var catchAlls []*Config
	for _, record := range txt {
		config := Parse(record)
		if config.From == "" {
			catchAlls = append(catchAlls, config)
			continue
		}
		redirect := Translate(url, config)
		if redirect != nil {
			return redirect, nil
		}
	}

	var config *Config
	for _, config = range catchAlls {
		redirect := Translate(url, config)
		if redirect != nil {
			return redirect, nil
		}
	}

	return nil, errors.New("No paths matched")
}

func handler(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.Host, ":")
	host := parts[0]

	hostname := fmt.Sprintf("_redirect.%s", host)
	txt, err := net.LookupTXT(hostname)
	if err != nil {
		fallback(w, r, fmt.Sprintf("Could not resolve hostname (%v)", err))
		return
	}

	redirect, err := getRedirect(txt, r.URL.String())
	if err != nil {
		fallback(w, r, err.Error())
	} else {
		http.Redirect(w, r, redirect.Location, redirect.Status)
	}
}

func main() {
	certmagic.Default.Agreed = true
	// certmagic.Default.CA = certmagic.LetsEncryptStagingCA
	certmagic.Default.OnDemand = new(certmagic.OnDemandConfig)
	certmagic.Default.Storage = magicstorage.NewS3Storage("certpool", "eu-central-1")
	magic := certmagic.NewDefault()

	mux := http.NewServeMux()
	mux.HandleFunc("/", handler)

	httpSrv := &http.Server{
		Handler: magic.HTTPChallengeHandler(mux),
		Addr:    ":80",
		// ReadTimeout:  2 * time.Second,
		// WriteTimeout: 2 * time.Second,
	}

	httpsSrv := &http.Server{
		Handler:   mux,
		Addr:      ":443",
		TLSConfig: magic.TLSConfig(),
		// ReadTimeout:  2 * time.Second,
		// WriteTimeout: 2 * time.Second,
	}

	ln, err := certmagic.Listen([]string{})
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		log.Printf("Listening on http://0.0.0.0:80")
		log.Fatal(httpSrv.ListenAndServe())
	}()

	log.Printf("Listening on https://0.0.0.0:443")
	log.Fatal(httpsSrv.Serve(ln))
}
