package naive

import "time"

// A Subscription delivers Items over a channel.  Close cancels the
// subscription, closes the Updates channel, and returns the last fetch error,
// if any.
type Subscription interface {
	Updates() <-chan Item
	Close() error
}

// Subscribe returns a new Subscription that uses fetcher to fetch Items.
func Subscribe(fetcher Fetcher) Subscription {
	s := &sub{
		fetcher: fetcher,
		updates: make(chan Item),       // for Updates
		closing: make(chan chan error), // for Close
	}
	go s.loop()
	return s
}

// sub implements the Subscription interface.
type sub struct {
	fetcher Fetcher         // fetches items
	updates chan Item       // sends items to the user
	closing chan chan error // for Close
}

func (s *sub) Updates() <-chan Item {
	return s.updates
}

func (s *sub) Close() error {
	errc := make(chan error)
	s.closing <- errc
	return <-errc
}

// loopCloseOnly is a version of loop that includes only the logic
// that handles Close.
func (s *sub) loopCloseOnly() {
	var err error // set when Fetch fails
	for {
		select {
		case errc := <-s.closing:
			errc <- err
			close(s.updates) // tells receiver we're done
			return
		}
	}
}

// loopFetchOnly is a version of loop that includes only the logic
// that calls Fetch.
func (s *sub) loopFetchOnly() {
	var pending []Item // appended by fetch; consumed by send
	var next time.Time // initially January 1, year 0
	var err error
	for {
		var fetchDelay time.Duration // initally 0 (no delay)
		if now := time.Now(); next.After(now) {
			fetchDelay = next.Sub(now)
		}
		startFetch := time.After(fetchDelay)

		select {
		case <-startFetch:
			var fetched []Item
			fetched, next, err = s.fetcher.Fetch()
			if err != nil {
				next = time.Now().Add(10 * time.Second)
				break
			}
			pending = append(pending, fetched...)
		}
	}
}

// loopSendOnly is a version of loop that includes only the logic for
// sending items to s.updates.
func (s *sub) loopSendOnly() {
	var pending []Item // appended by fetch; consumed by send
	for {
		var first Item
		var updates chan Item // HLupdates
		if len(pending) > 0 {
			first = pending[0]
			updates = s.updates // enable send case // HLupdates
		}

		select {
		case updates <- first:
			pending = pending[1:]
		}
	}
}

// mergedLoop is a version of loop that combines loopCloseOnly,
// loopFetchOnly, and loopSendOnly.
func (s *sub) mergedLoop() {
	var pending []Item
	var next time.Time
	var err error
	for {
		var fetchDelay time.Duration
		if now := time.Now(); next.After(now) {
			fetchDelay = next.Sub(now)
		}
		startFetch := time.After(fetchDelay)
		var first Item
		var updates chan Item
		if len(pending) > 0 {
			first = pending[0]
			updates = s.updates // enable send case
		}

		select {
		case errc := <-s.closing: // HLcases
			errc <- err
			close(s.updates)
			return
		case <-startFetch: // HLcases
			var fetched []Item
			fetched, next, err = s.fetcher.Fetch() // HLfetch
			if err != nil {
				next = time.Now().Add(10 * time.Second)
				break
			}
			pending = append(pending, fetched...) // HLfetch
		case updates <- first: // HLcases
			pending = pending[1:]
		}
	}
}

// dedupeLoop extends mergedLoop with deduping of fetched items.
func (s *sub) dedupeLoop() {
	const maxPending = 10
	// STARTSEEN OMIT
	var pending []Item
	var next time.Time
	var err error
	var seen = make(map[string]bool) // set of item.GUIDs // HLseen
	// STOPSEEN OMIT
	for {
		// STARTCAP OMIT
		var fetchDelay time.Duration
		if now := time.Now(); next.After(now) {
			fetchDelay = next.Sub(now)
		}
		var startFetch <-chan time.Time // HLcap
		if len(pending) < maxPending {  // HLcap
			startFetch = time.After(fetchDelay) // enable fetch case  // HLcap
		} // HLcap
		// STOPCAP OMIT
		var first Item
		var updates chan Item
		if len(pending) > 0 {
			first = pending[0]
			updates = s.updates // enable send case
		}
		select {
		case errc := <-s.closing:
			errc <- err
			close(s.updates)
			return
		// STARTDEDUPE OMIT
		case <-startFetch:
			var fetched []Item
			fetched, next, err = s.fetcher.Fetch() // HLfetch
			if err != nil {
				next = time.Now().Add(10 * time.Second)
				break
			}
			for _, item := range fetched {
				if !seen[item.GUID] { // HLdupe
					pending = append(pending, item) // HLdupe
					seen[item.GUID] = true          // HLdupe
				} // HLdupe
			}
			// STOPDEDUPE OMIT
		case updates <- first:
			pending = pending[1:]
		}
	}
}

// loop periodically fecthes Items, sends them on s.updates, and exits
// when Close is called.  It extends dedupeLoop with logic to run
// Fetch asynchronously.
func (s *sub) loop() {
	const maxPending = 10
	type fetchResult struct {
		fetched []Item
		next    time.Time
		err     error
	}
	// STARTFETCHDONE OMIT
	var fetchDone chan fetchResult // if non-nil, Fetch is running // HL
	// STOPFETCHDONE OMIT
	var pending []Item
	var next time.Time
	var err error
	var seen = make(map[string]bool)
	for {
		var fetchDelay time.Duration
		if now := time.Now(); next.After(now) {
			fetchDelay = next.Sub(now)
		}
		// STARTFETCHIF OMIT
		var startFetch <-chan time.Time
		if fetchDone == nil && len(pending) < maxPending { // HLfetch
			startFetch = time.After(fetchDelay) // enable fetch case
		}
		// STOPFETCHIF OMIT
		var first Item
		var updates chan Item
		if len(pending) > 0 {
			first = pending[0]
			updates = s.updates // enable send case
		}
		// STARTFETCHASYNC OMIT
		select {
		case <-startFetch: // HLfetch
			fetchDone = make(chan fetchResult, 1) // HLfetch
			go func() {
				fetched, next, err := s.fetcher.Fetch()
				fetchDone <- fetchResult{fetched, next, err}
			}()
		case result := <-fetchDone: // HLfetch
			fetchDone = nil // HLfetch
			// Use result.fetched, result.next, result.err
			// STOPFETCHASYNC OMIT
			fetched := result.fetched
			next, err = result.next, result.err
			if err != nil {
				next = time.Now().Add(10 * time.Second)
				break
			}
			for _, item := range fetched {
				if id := item.GUID; !seen[id] { // HLdupe
					pending = append(pending, item)
					seen[id] = true // HLdupe
				}
			}
		case errc := <-s.closing:
			errc <- err
			close(s.updates)
			return
		case updates <- first:
			pending = pending[1:]
		}
	}
}
