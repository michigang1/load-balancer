package main

import (
	"context"
	"flag"
	"fmt"
	"hash/fnv"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/roman-mazur/design-practice-2-template/httptools"
	"github.com/roman-mazur/design-practice-2-template/signal"
)

var (
	port       = flag.Int("port", 8090, "load balancer port")
	timeoutSec = flag.Int("timeout-sec", 3, "request timeout time in seconds")
	https      = flag.Bool("https", false, "whether backends support HTTPs")

	traceEnabled = flag.Bool("trace", false, "whether to include tracing information into responses")
)

var (
	timeout     = time.Duration(*timeoutSec) * time.Second
	serversPool = []string{
		"server1:8080",
		"server2:8080",
		"server3:8080",
	}
	healthyServers = make([]string, 3)
)

func scheme() string {
	if *https {
		return "https"
	}
	return "http"
}

func health(dst string) bool {
	ctx, _ := context.WithTimeout(context.Background(), timeout)
	req, _ := http.NewRequestWithContext(ctx, "GET",
		fmt.Sprintf("%s://%s/health", scheme(), dst), nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	if resp.StatusCode != http.StatusOK {
		return false
	}
	return true
}

func forward(dst string, rw http.ResponseWriter, r *http.Request) error {
	ctx, _ := context.WithTimeout(r.Context(), timeout)
	fwdRequest := r.Clone(ctx)
	fwdRequest.RequestURI = ""
	fwdRequest.URL.Host = dst
	fwdRequest.URL.Scheme = scheme()
	fwdRequest.Host = dst

	resp, err := http.DefaultClient.Do(fwdRequest)
	if err == nil {
		for k, values := range resp.Header {
			for _, value := range values {
				rw.Header().Add(k, value)
			}
		}
		if *traceEnabled {
			rw.Header().Set("lb-from", dst)
		}
		log.Println("fwd", resp.StatusCode, resp.Request.URL)
		rw.WriteHeader(resp.StatusCode)
		defer resp.Body.Close()
		_, err := io.Copy(rw, resp.Body)
		if err != nil {
			log.Printf("Failed to write response: %s", err)
		}
		return nil
	} else {
		log.Printf("Failed to get response from %s: %s", dst, err)
		rw.WriteHeader(http.StatusServiceUnavailable)
		return err
	}
}

type LoadBalancer struct {
	serversPool    []string
	healthyServers []string
	traceEnabled   *bool
	port           *int
}

// NewLoadBalancer creates a new instance of LoadBalancer.
func NewLoadBalancer(serversPool []string, traceEnabled *bool, port *int) *LoadBalancer {
	return &LoadBalancer{
		serversPool:    serversPool,
		healthyServers: []string{},
		traceEnabled:   traceEnabled,
		port:           port,
	}
}

// healthCheck periodically checks the health of each server in the serversPool and updates the healthyServers list accordingly.
func (lb *LoadBalancer) healthCheck() {
	for i, server := range lb.serversPool {
		go func(server string, i int) {
			for range time.Tick(10 * time.Second) {
				isHealthy := health(server)
				if !isHealthy {
					lb.serversPool[i] = ""
				} else {
					lb.serversPool[i] = server
				}

				lb.healthyServers = lb.healthyServers[:0]

				for _, value := range lb.serversPool {
					if value != "" {
						lb.healthyServers = append(lb.healthyServers, value)
					}
				}

				log.Println(server, isHealthy)
			}
		}(server, i)
	}
}

// selectServer selects a server based on the remote address and the list of healthy servers.
func (lb *LoadBalancer) selectServer(remoteAddr string) string {
	hash := fnv.New32a()
	_, _ = hash.Write([]byte(remoteAddr))
	sum := int(hash.Sum32())
	index := sum % len(lb.healthyServers)
	log.Println("hash", sum)
	log.Println("selected server", index)
	return lb.healthyServers[index]
}

// Run starts the load balancer and handles incoming requests.
func (lb *LoadBalancer) Run() {
	lb.healthCheck()

	frontend := httptools.CreateServer(*lb.port, http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		remoteAddr := r.RemoteAddr
		server := lb.selectServer(remoteAddr)
		forward(server, rw, r)
	}))

	log.Println("Starting load balancer...")
	log.Printf("Tracing support enabled: %t", *lb.traceEnabled)
	frontend.Start()
	signal.WaitForTerminationSignal()
}

func main() {
	flag.Parse()

	lb := NewLoadBalancer(serversPool, traceEnabled, port)
	lb.Run()
}
