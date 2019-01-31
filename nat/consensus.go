package nat

import (
	"github.com/pkg/errors"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type source interface {
	IP(timeout time.Duration) (net.IP, error)
}

type voter struct {
	source source // provides the IP (see: vote)
	weight uint   // provides the weight of its vote (acts as a multiplier)
}

var _ Provider = (*consensusProvider)(nil)

type consensusProvider struct {
	voters []voter
	cached net.IP
}

func NewConsensusProvider() *consensusProvider {
	consensus := new(consensusProvider)

	consensus.addVoter(newHTTPSource("https://icanhazip.com/"), 3)
	consensus.addVoter(newHTTPSource("https://myexternalip.com/raw"), 3)

	// Plain-text providers.
	consensus.addVoter(newHTTPSource("http://ifconfig.io/ip"), 1)
	consensus.addVoter(newHTTPSource("http://checkip.amazonaws.com/"), 1)
	consensus.addVoter(newHTTPSource("http://ident.me/"), 1)
	consensus.addVoter(newHTTPSource("http://whatismyip.akamai.com/"), 1)
	consensus.addVoter(newHTTPSource("http://tnx.nl/ip"), 1)
	consensus.addVoter(newHTTPSource("http://myip.dnsomatic.com/"), 1)
	consensus.addVoter(newHTTPSource("http://diagnostic.opendns.com/myip"), 1)

	return consensus
}

func (c *consensusProvider) addVoter(source source, weight uint) {
	if source == nil {
		panic("nat[consensus]: no voting source was specified")
	}

	if weight == 0 {
		panic("nat[consensus]: weight for a voting source cannot be 0")
	}

	c.voters = append(c.voters, voter{
		source: source,
		weight: weight,
	})
}

func (c *consensusProvider) ExternalIP() net.IP {
	if c.cached != nil {
		return c.cached
	}

	votes := make(map[string]uint)

	var mutex sync.Mutex
	var wg sync.WaitGroup

	for _, v := range c.voters {
		wg.Add(1)

		go func(v voter) {
			defer wg.Done()

			ip, err := v.source.IP(5 * time.Second)

			if err == nil && ip != nil {
				mutex.Lock()
				votes[ip.String()] += v.weight
				mutex.Unlock()
			}
		}(v)
	}

	wg.Wait()

	if len(votes) == 0 {
		panic("nat[consensus]: no voters could respond with a valid external IP candidate")
	}

	var max uint
	var externalIP string

	for ip, votes := range votes {
		if votes > max {
			max, externalIP = votes, ip
		}
	}

	c.cached = net.ParseIP(externalIP)

	return c.cached
}

func newHTTPSource(url string) httpSource {
	return httpSource{url: url}
}

type httpSource struct {
	url    string
	parser func(buf []byte) (string, error)
}

func (s httpSource) WithParser(parser func(buf []byte) (string, error)) httpSource {
	s.parser = parser
	return s
}

func (s httpSource) IP(timeout time.Duration) (net.IP, error) {
	req, err := http.NewRequest("GET", s.url, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: timeout}

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	buf, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var raw string

	if s.parser != nil {
		raw, err = s.parser(buf)

		if err != nil {
			return nil, err
		}
	} else {
		raw = string(buf)
	}

	externalIP := net.ParseIP(strings.TrimSpace(raw))

	if externalIP == nil {
		return nil, errors.Errorf("got an invalid external IP %q", externalIP)
	}

	return externalIP, nil
}
