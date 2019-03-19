package naive

import "time"

func NaiveSubscribe(fetcher Fetcher) Subscription {
	s := &naiveSub{
		fetcher: fetcher,
		updates: make(chan Item),
	}
	go s.loop()
	return s
}

type naiveSub struct {
	fetcher Fetcher
	updates chan Item
	closed  bool
	err     error
}

func (s *naiveSub) Updates() <-chan Item {
	return s.updates
}

func (s *naiveSub) loop() {
	// STARTNAIVE OMIT
	for {
		if s.closed { // HLsync
			close(s.updates)
			return
		}
		items, next, err := s.fetcher.Fetch()
		if err != nil {
			s.err = err                  // HLsync
			time.Sleep(10 * time.Second) // HLsleep
			continue
		}
		for _, item := range items {
			s.updates <- item // HLsend
		}
		if now := time.Now(); next.After(now) {
			time.Sleep(next.Sub(now)) // HLsleep
		}
	}
	// STOPNAIVE OMIT
}

func (s *naiveSub) Close() error {
	s.closed = true // HLsync
	return s.err    // HLsync
}
