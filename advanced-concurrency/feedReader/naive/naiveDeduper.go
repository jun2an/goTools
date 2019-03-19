package naive

// NaiveDedupe converts a stream of Items that may contain duplicates
// into one that doesn't.
func NaiveDedupe(in <-chan Item) <-chan Item {
	out := make(chan Item)
	go func() {
		seen := make(map[string]bool)
		for it := range in {
			if !seen[it.GUID] {
				// BUG: this send blocks if the
				// receiver closes the Subscription
				// and stops receiving.
				out <- it // HL
				seen[it.GUID] = true
			}
		}
		close(out)
	}()
	return out
}
