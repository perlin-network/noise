package skademlia

import (
	"time"
)

type ClientFilter func([]*ID) []*ID

func Dummy(ids []*ID) []*ID {
	return ids
}

func FilterFailedAgo(ago time.Duration) ClientFilter {
	return func(ids []*ID) []*ID {
		res := make([]*ID, 0, len(ids))
		t := time.Now().Truncate(ago)
		for _, id := range ids {
			if id.lastFailTime.Before(t) {
				res = append(res, id)
			}
		}

		return res
	}
}
