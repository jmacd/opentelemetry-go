package trace

// pipelines are immutable
type pipelines struct {
	provider *TracerProvider
	readers  []SpanReader
}

func (provider *TracerProvider) newPipelines(readers []SpanReader) *pipelines {
	return &pipelines{
		provider: provider,
		readers:  readers,
	}
}

func (p pipelines) add(r SpanReader) *pipelines {
	rs := make([]SpanReader, len(p.readers)+1)
	copy(rs[:len(p.readers)], p.readers)
	rs[len(p.readers)] = r
	return p.provider.newPipelines(rs)
}

func (p pipelines) remove(remFunc func(SpanReader) bool) *pipelines {
	rs := make([]SpanReader, 0, len(p.readers))
	for _, r := range p.readers {
		if !remFunc(r) {
			rs = append(rs, r)
		}
	}
	return p.provider.newPipelines(rs)
}
