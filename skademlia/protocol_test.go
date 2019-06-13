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
	{"[7a8d:e2e:9b16:e1cc:f9c4:ce95:d96a:a32c]:100", "[7a8d:e2e:9b16:e1cc:f9c4:ce95:d96a:a32c]:100", true},
	{"127.0.0.1:100", "127.0.0.1:100", true},
	{":100", "127.0.0.1:100", true},
	{"127.0.0.1:100", ":100", true},
	{"[::]:100", "127.0.0.1:100", true},
	{"127.0.0.1:100", "127.0.0.2:100", false},
	{"127.0.0.1:100", "127.0.0.1:101", false},
	{"0.0:0", "127.0.0.1:100", false},
	{"invalid_address", "127.0.0.1:100", false},
}

func TestAddressMatches(t *testing.T) {
	for _, hostMatchTest := range addressMatchesTests {
		assert.True(t, addressMatches(hostMatchTest.bind, hostMatchTest.subject) == hostMatchTest.equal, hostMatchTest.bind+" compared to "+hostMatchTest.subject)
	}
}

func TestQuickCheckAddressMatchesIPV4(t *testing.T) {
	f := func(ip [net.IPv4len]byte, port uint16) bool {
		return addressMatches(net.IP(ip[:]).String()+":"+strconv.Itoa(int(port)), net.IP(ip[:]).String()+":"+strconv.Itoa(int(port)))
	}
	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestQuickCheckAddressMatchesIPV6(t *testing.T) {
	f := func(ip [net.IPv6len]byte, port uint16) bool {
		return addressMatches("["+net.IP(ip[:]).String()+"]"+":"+strconv.Itoa(int(port)), "["+net.IP(ip[:]).String()+"]"+":"+strconv.Itoa(int(port)))

	}
	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}
