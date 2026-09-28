package main

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"time"
)

func runHealthcheck(listenAddr string) int {
	url := healthcheckURL(listenAddr)
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "healthcheck: %v\n", err)
		return 1
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "healthcheck: %s\n", resp.Status)
		return 1
	}
	return 0
}

func healthcheckURL(listenAddr string) string {
	host, port, err := net.SplitHostPort(listenAddr)
	if err != nil {
		return "http://127.0.0.1:8080/healthz"
	}
	switch host {
	case "", "0.0.0.0", "::", "[::]":
		host = "127.0.0.1"
	}
	return "http://" + net.JoinHostPort(host, port) + "/healthz"
}
