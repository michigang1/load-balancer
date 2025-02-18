package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/michigang1/load-balancer/httptools"
	"github.com/michigang1/load-balancer/signal"
	"hash/fnv"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
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
	mutex sync.Mutex
)

func scheme() string {
	if *https {
		return "https"
	}
	return "http"
}

func health(dst string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
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
	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()

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
		log.Printf("Succesful response from %s", dst)
		log.Println("fwd", resp.StatusCode, resp.Request.URL)
		rw.WriteHeader(resp.StatusCode)
		defer resp.Body.Close()
		_, err := io.Copy(rw, resp.Body)
		if err != nil {
			log.Printf("Failed to write response: %s", err)
		}
		return nil
	} else {
		log.Printf("Try to get response from %s: %s", dst, err)
		rw.WriteHeader(http.StatusServiceUnavailable)
		return err
	}
}

type LoadBalancer struct {
	serversHealthyStatus map[string]bool
	durationOfChecking   time.Duration
}

// CheckHealthyServers periodically checks the health of each server in the serversPool pool
func (lb *LoadBalancer) CheckHealthyServers() {
	timeCheck := time.Tick(lb.durationOfChecking)
	for i, server := range serversPool {
		go func(server string, i int) {
			for range timeCheck {
				if health(server) {
					lb.serversHealthyStatus[server] = true
				} else {
					lb.serversHealthyStatus[server] = false
				}
			}
		}(server, i)
	}
}

// GetHealthyServers  checks the health of each server in the serversHealthyStatus and updates the healthyServers list accordingly.
func (lb *LoadBalancer) GetHealthyServers() []string {
	var healthyServers []string
	for _, server := range serversPool {
		if lb.serversHealthyStatus[server] {
			healthyServers = append(healthyServers, server)
		}
	}
	return healthyServers
}

// SelectServer selects a server based on the remote address and the list of healthy servers.
func (lb *LoadBalancer) SelectServer(remoteAddr string) string {
	healthyServers := lb.GetHealthyServers()
	if len(healthyServers) == 0 {
		log.Println("No healthy servers")
		return ""
	} else {
		serverIndex := lb.hash(remoteAddr)
		return healthyServers[serverIndex]
	}
}
func (lb *LoadBalancer) hash(input string) int {
	hash := fnv.New32()
	_, err := hash.Write([]byte(input))
	if err != nil {
		log.Printf("Failed to compute hash: %s", err)
		return 0
	}
	sum := int(hash.Sum32())
	index := sum % len(lb.GetHealthyServers())
	log.Println("hash", sum)
	log.Println("selected server", index+1)
	return index
}

func main() {
	flag.Parse()

	balancer := &LoadBalancer{}
	balancer.serversHealthyStatus = make(map[string]bool)
	balancer.durationOfChecking = 10 * time.Second

	go balancer.CheckHealthyServers()
	frontend := httptools.CreateServer(*port, http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		server := balancer.SelectServer(r.RemoteAddr)
		forward(server, rw, r)
	}))

	log.Println("Starting load balancer...")
	log.Printf("Tracing support enabled: %t", *traceEnabled)
	frontend.Start()
	signal.WaitForTerminationSignal()
}
