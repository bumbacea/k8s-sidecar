package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func startMetricsServer(port uint) (io.Closer, error) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, fmt.Errorf("unable to bind to port: %w", err)
	}
	log.Printf("Started listening on port: %s", ln.Addr().String())

	server := &http.Server{Addr: fmt.Sprintf(":%d", port), Handler: mux}
	go func() {
		if err := server.Serve(ln); err != nil {
			log.Printf("failed to serve: %s", err)
		}
	}()
	return server, nil
}
