package integration

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"testing"
	"time"

	. "gopkg.in/check.v1"
)

const baseAddress = "http://balancer:8090"

var client = http.Client{
	Timeout: 20 * time.Second,
}

func Test(t *testing.T) { TestingT(t) }

type BalancerIntegrationSuite struct{}

var _ = Suite(&BalancerIntegrationSuite{})

func (b *BalancerIntegrationSuite) TestBalancer(c *C) {
	if _, exists := os.LookupEnv("INTEGRATION_TEST"); !exists {
		c.Skip("Integration test is not enabled")
	}

	for i := 0; i < 10; i++ {
		time.Sleep(5 * time.Second)
		resp, err := client.Get(fmt.Sprintf("%s/api/v1/some-data", baseAddress))
		c.Assert(err, IsNil)
		log.Printf("response from [%s]", resp.Header.Get("lb-from"))
	}
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
