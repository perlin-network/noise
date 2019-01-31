package noise

import "go.dedis.ch/kyber/v3/suites"

func init() {
	suites.RequireConstantTime()
}
