package discovery

import (
	shared "github.com/HailoOSS/discovery-service/proto"
	"sort"
)

type lessFunc func(s1, s2 *shared.Service) bool

type multiSorter struct {
	services []*shared.Service
	less     []lessFunc
}

func (ms *multiSorter) Sort(services []*shared.Service) {
	ms.services = services
	sort.Sort(ms)
}

func OrderedBy(less ...lessFunc) *multiSorter {
	return &multiSorter{
		less: less,
	}
}

func (ms *multiSorter) Len() int {
	return len(ms.services)
}

func (ms *multiSorter) Swap(i, j int) {
	ms.services[i], ms.services[j] = ms.services[j], ms.services[i]
}

func (ms *multiSorter) Less(i, j int) bool {
	p, q := ms.services[i], ms.services[j]
	// Try all but the last comparison.
	var k int
	for k = 0; k < len(ms.less)-1; k++ {
		less := ms.less[k]
		switch {
		case less(p, q):
			// p < q, so we have a decision.
			return true
		case less(q, p):
			// p > q, so we have a decision.
			return false
		}
		// p == q; try the next comparison.
	}
	// All comparisons to here said "equal", so just return whatever
	// the final comparison reports.
	return ms.less[k](p, q)
}
