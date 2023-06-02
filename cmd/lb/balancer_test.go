package main

import (
	. "gopkg.in/check.v1"
	"testing"
	"time"
)

func Test(t *testing.T) { TestingT(t) }

type BalancerSuite struct{}

var _ = Suite(&BalancerSuite{})

func (s *BalancerSuite) TestBalancerAllHealthy(c *C) {
	balancer := &LoadBalancer{}

	balancer.durationOfChecking = 10 * time.Second

	balancer.serversHealthyStatus = map[string]bool{
		"server1:8080": true,
		"server2:8080": true,
		"server3:8080": true,
	}
	server1 := balancer.SelectServer("92.168.0.0:80")
	server2 := balancer.SelectServer("127.0.0.0:8080")
	server3 := balancer.SelectServer("26.143.218.9:80")

	c.Assert("server1:8080", Equals, server1)
	c.Assert("server3:8080", Equals, server2)
	c.Assert("server2:8080", Equals, server3)

}
func (s *BalancerSuite) TestBalancerNotAllHealthy(c *C) {
	balancer := &LoadBalancer{}

	balancer.durationOfChecking = 10 * time.Second

	balancer.serversHealthyStatus = map[string]bool{
		"server1:8080": true,
		"server2:8080": true,
		"server3:8080": false,
	}

	balancer.CheckHealthyServers()

	addr1 := balancer.SelectServer("92.168.0.0:80")
	addr2 := balancer.SelectServer("127.0.0.0:8080")
	addr3 := balancer.SelectServer("26.143.218.9:80")

	addr4 := balancer.SelectServer("95.138.0.1:80")
	addr5 := balancer.SelectServer("123.0.3.0:8080")
	addr6 := balancer.SelectServer("46.143.218.9:80")
	c.Assert("server2:8080", Equals, addr1)
	c.Assert("server2:8080", Equals, addr2)
	c.Assert("server2:8080", Equals, addr3)

	c.Assert("server1:8080", Equals, addr4)
	c.Assert("server1:8080", Equals, addr5)
	c.Assert("server2:8080", Equals, addr6)

}
func (s *BalancerSuite) TestBalancerNoHealthy(c *C) {
	balancer := &LoadBalancer{}

	balancer.durationOfChecking = 10 * time.Second

	balancer.serversHealthyStatus = map[string]bool{
		"server1:8080": false,
		"server2:8080": false,
		"server3:8080": false,
	}

	balancer.CheckHealthyServers()

	addr1 := balancer.SelectServer("92.168.0.0:80")
	addr2 := balancer.SelectServer("127.0.0.0:8080")
	addr3 := balancer.SelectServer("26.143.218.9:80")

	addr4 := balancer.SelectServer("95.138.0.1:80")
	addr5 := balancer.SelectServer("123.0.3.0:8080")
	addr6 := balancer.SelectServer("46.143.218.9:80")
	c.Assert("", Equals, addr1)
	c.Assert("", Equals, addr2)
	c.Assert("", Equals, addr3)

	c.Assert("", Equals, addr4)
	c.Assert("", Equals, addr5)
	c.Assert("", Equals, addr6)

}
