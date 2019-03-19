package naive

type deduper struct {
	s       Subscription
	updates chan Item
	closing chan chan error
}

// Dedupe converts a Subscription that may send duplicate Items into
// one that doesn't.
func Dedupe(s Subscription) Subscription {
	d := &deduper{
		s:       s,
		updates: make(chan Item),
		closing: make(chan chan error),
	}
	go d.loop()
	return d
}

func (d *deduper) loop() {
	in := d.s.Updates() // enable receive
	var pending Item
	var out chan Item // disable send
	seen := make(map[string]bool)
	for {
		select {
		case it := <-in:
			if !seen[it.GUID] {
				pending = it
				in = nil        // disable receive
				out = d.updates // enable send
				seen[it.GUID] = true
			}
		case out <- pending:
			in = d.s.Updates() // enable receive
			out = nil          // disable send
		case errc := <-d.closing:
			err := d.s.Close()
			errc <- err
			close(d.updates)
			return
		}
	}
}

func (d *deduper) Close() error {
	errc := make(chan error)
	d.closing <- errc
	return <-errc
}

func (d *deduper) Updates() <-chan Item {
	return d.updates
}
