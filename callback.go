package main

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/cenkalti/backoff"
)

type ReqConfig struct {
	Url                string  `envconfig:"REQ_URL" required:"true"`
	Method             string  `envconfig:"REQ_METHOD" default:"GET"`
	Payload            []byte  `envconfig:"REQ_PAYLOAD"`
	RetryTotal         uint    `envconfig:"REQ_RETRY_TOTAL" default:"5"`
	RetryBackoffFactor float64 `envconfig:"REQ_RETRY_BACKOFF_FACTOR" default:"1.1"`
	Timeout            string  `envconfig:"REQ_TIMEOUT" default:"10"`
	Username           string  `envconfig:"REQ_USERNAME"`
	Password           string  `envconfig:"REQ_PASSWORD"`
	BasicAuthEncoding  string  `envconfig:"REQ_BASIC_AUTH_ENCODING" default:"latin1"`
	SkipTlsVerify      bool    `envconfig:"REQ_SKIP_TLS_VERIFY" default:"false"`
}

func runCallback(reqConfig ReqConfig) {
	client := &http.Client{}
	if reqConfig.SkipTlsVerify {
		client.Transport = &http.Transport{TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		}}
	}

	req, err := http.NewRequest(reqConfig.Method, reqConfig.Url, bytes.NewBuffer(reqConfig.Payload))
	if err != nil {
		log.Printf("failed to create request struct: %s", err)
	}
	if reqConfig.Username != "" || reqConfig.Password != "" {
		req.SetBasicAuth(reqConfig.Username, reqConfig.Password)
	}

	expBackoff := backoff.NewExponentialBackOff()
	expBackoff.MaxElapsedTime = 10 * time.Second
	expBackoff.Multiplier = reqConfig.RetryBackoffFactor

	err = backoff.Retry(func() error {
		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("failed retry to make req: %w", err)
		}

		log.Printf("Made request to %s and got status code %d", reqConfig.Url, resp.StatusCode)

		return nil
	}, expBackoff)

	if err != nil {
		log.Printf("permanent failed to make request: %s", err)
	}

}
