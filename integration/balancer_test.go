package integration

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	. "gopkg.in/check.v1"
)

const baseAddress = "http://balancer:8090"

var client = http.Client{
	Timeout: 3 * time.Second,
}

func Test(t *testing.T) { TestingT(t) }

type BalancerIntegrationSuite struct{}

var _ = Suite(&BalancerIntegrationSuite{})

func (b *BalancerIntegrationSuite) TestBalancer(c *C) {
	if _, exists := os.LookupEnv("INTEGRATION_TEST"); !exists {
		c.Skip("Integration test is not enabled")
	}

	if !checkBalancer() {
		c.Skip("Balancer is not available")
	}

	mockServer := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(http.StatusOK)
		rw.Header().Set("lb-from", "mock-server")
	}))

	resp, err := client.Get(fmt.Sprintf("%s/api/v1/some-data", baseAddress))
	c.Assert(err, IsNil)
	c.Assert(http.StatusOK, Equals, resp.StatusCode)
	c.Assert("mock-server", Equals, resp.Header.Get("lb-from"))

	mockServer.Close()
}
func checkBalancer() bool {
	resp, err := client.Get(baseAddress)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

func (s *BalancerIntegrationSuite) BenchmarkBalancer(c *C) {
	if _, exists := os.LookupEnv("INTEGRATION_TEST"); !exists {
		c.Skip("Integration test is not enabled")
	}

	for i := 0; i < c.N; i++ {
		_, err := client.Get(fmt.Sprintf("%s/api/v1/some-data", baseAddress))
		if err != nil {
			c.Error(err)
		}
	}
}
