package yacymodel

// SeedArc is the run of ring positions after one seed's predecessor, through the
// seed. Every position in an arc has the same closest seeds.
type SeedArc struct {
	After   DHTRingPosition
	Through DHTRingPosition
	Window  []Seed
}

// Spans are the arc as spans that do not cross the end of the ring.
func (a SeedArc) Spans() []DHTRingSpan {
	if a.After < a.Through {
		return []DHTRingSpan{{First: a.After + 1, Last: a.Through}}
	}
	var spans []DHTRingSpan
	if a.After < MaxDHTRingPosition {
		spans = append(spans, DHTRingSpan{First: a.After + 1, Last: MaxDHTRingPosition})
	}

	return append(spans, DHTRingSpan{First: 0, Last: a.Through})
}

func (a SeedArc) Holds(position DHTRingPosition) bool {
	if a.After < a.Through {
		return a.After < position && position <= a.Through
	}

	return position > a.After || position <= a.Through
}

func (a SeedArc) Overlaps(other SeedArc) bool {
	return a.Holds(other.Through) || other.Holds(a.Through)
}

// Arcs divides the ring into one arc per seed position, each with the first
// windowSize seeds clockwise from its end.
func (r SeedRing) Arcs(windowSize int) []SeedArc {
	arcs := make([]SeedArc, 0, len(r.seeds))
	for index, through := range r.positions {
		if index > 0 && r.positions[index-1] == through {
			continue
		}
		after := r.positions[(index+len(r.positions)-1)%len(r.positions)]
		arcs = append(arcs, SeedArc{
			After:   after,
			Through: through,
			Window:  r.SeedsClosestTo(through, windowSize),
		})
	}

	return arcs
}
