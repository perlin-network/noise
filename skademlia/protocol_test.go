// Copyright (c) 2019 Perlin
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
package skademlia

import (
	"math/rand"
	"net"
	"strconv"
	"testing"
	"testing/quick"

	"github.com/stretchr/testify/assert"
)

var addressMatchesTests = []struct {
	bind    string
	subject string
	equal   bool
}{
	{"127.0.0.1:100", "127.0.0.1:100", true},
	{":100", "127.0.0.1:100", true},
	{"127.0.0.1:100", ":100", true},
	{"[::]:100", "127.0.0.1:100", true},
	{"127.0.0.1:100", "127.0.0.2:100", false},
	{"127.0.0.1:100", "127.0.0.1:101", false},
}

func TestAddressMatches(t *testing.T) {
	for _, hostMatchTest := range addressMatchesTests {
		assert.True(t, addressMatches(hostMatchTest.bind, hostMatchTest.subject) == hostMatchTest.equal, hostMatchTest.bind+" compared to "+hostMatchTest.subject)
	}
}

func TestQuickCheckAddressMatchesIPV4(t *testing.T) {
	f := func(n int64, port uint16) bool {
		return addressMatches(randomIpV4(n)+":"+strconv.Itoa(int(port)), randomIpV4(n)+":"+strconv.Itoa(int(port)))
	}
	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestQuickCheckAddressMatchesIPV6(t *testing.T) {
	f := func(n int64, port uint16) bool {
		return addressMatches(randomIpV6(n)+":"+strconv.Itoa(int(port)), randomIpV6(n)+":"+strconv.Itoa(int(port)))

	}
	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func randomIpV4(n int64) string {
	return net.IP(randomBytes(n, net.IPv4len)).String()
}

func randomIpV6(n int64) string {
	return net.IP(randomBytes(n, net.IPv6len)).String()
}

func randomBytes(n int64, len int) []byte {
	rand.Seed(n)
	bytes := make([]byte, len)
	for i := 0; i < len; i++ {
		bytes[i] = byte(rand.Intn(255))
	}
	return bytes
}
