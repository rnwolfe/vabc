package cli

// closest returns the nearest candidate within a small edit distance, for "did you mean".
func closest(word string, candidates []string) (string, bool) {
	best := ""
	bestDist := 1 << 30
	for _, c := range candidates {
		d := levenshtein(word, c)
		if d < bestDist {
			bestDist, best = d, c
		}
	}
	// Only suggest when reasonably close (threshold scales loosely with length).
	if best != "" && bestDist <= 3 && bestDist < len(best) {
		return best, true
	}
	return "", false
}

func levenshtein(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	prev := make([]int, len(rb)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ra); i++ {
		cur := make([]int, len(rb)+1)
		cur[0] = i
		for j := 1; j <= len(rb); j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			cur[j] = min3(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev = cur
	}
	return prev[len(rb)]
}

func min3(a, b, c int) int {
	if b < a {
		a = b
	}
	if c < a {
		a = c
	}
	return a
}
