package integration

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"gopkg.in/check.v1"
)

const baseAddress = "http://balancer:8090"

var client = http.Client{
	Timeout: 3 * time.Second,
}

type BalancerIntegrationSuite struct{}

func init() {
	check.Suite(&BalancerIntegrationSuite{})
}

func (b *BalancerIntegrationSuite) TestBalancer(c *check.C) {
	if _, exists := os.LookupEnv("INTEGRATION_TEST"); !exists {
		c.Skip("Integration test is not enabled")
	}

	testCases := []struct {
		endpoint string
		expected string
	}{
		{"/api/v1/some-data1", "server1:8080"},
		{"/api/v1/some-data2", "server2:8080"},
		{"/api/v1/some-dat3", "server3:8080"},
		{"/api/v1/some-data4", "server2:8080"},
	}

	for _, tc := range testCases {
		resp, err := client.Get(fmt.Sprintf("%s%s", baseAddress, tc.endpoint))
		if err != nil {
			c.Error(err)
		}
		c.Check(resp.Header.Get("lb-from"), check.Equals, tc.expected)
	}
}

func (s *BalancerIntegrationSuite) BenchmarkBalancer(c *check.C) {
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
