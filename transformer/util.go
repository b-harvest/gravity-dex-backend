package transformer

import "time"

func newImmediateTicker(d time.Duration) *time.Ticker {
	t := time.NewTicker(d)
	oc := t.C
	nc := make(chan time.Time, 1)
	go func() {
		nc <- time.Now()
		for tm := range oc {
			nc <- tm
		}
	}()
	t.C = nc
	return t
}
