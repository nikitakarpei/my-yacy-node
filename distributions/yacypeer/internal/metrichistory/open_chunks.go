package metrichistory

import (
	"bytes"
	"math"

	"github.com/prometheus/prometheus/tsdb/chunkenc"
	"github.com/prometheus/prometheus/tsdb/chunks"
)

type openChunk struct {
	chunk    *chunkenc.XORChunk
	appender chunkenc.Appender
	minTime  int64
	maxTime  int64
}

func openChunkFrom(at int64) *openChunk {
	chunk := chunkenc.NewXORChunk()
	appender, _ := chunk.Appender()
	return &openChunk{chunk: chunk, appender: appender, minTime: at, maxTime: at}
}

func (c *openChunk) cutFor(id seriesID) cutChunk {
	return cutChunk{
		key:     chunkKey{series: id, maxTime: c.maxTime},
		samples: chunkSamples{data: c.chunk.Bytes(), minTime: c.minTime},
	}
}

type openChunks struct {
	chunks map[seriesID]*openChunk
}

func newOpenChunks() *openChunks {
	return &openChunks{chunks: map[seriesID]*openChunk{}}
}

func (o *openChunks) append(id seriesID, at int64, value float64) (full cutChunk, isFull bool) {
	open, exists := o.chunks[id]
	if !exists {
		open = openChunkFrom(at)
		o.chunks[id] = open
	} else if at <= open.maxTime {
		return cutChunk{}, false
	}
	open.appender.Append(0, at, value)
	open.maxTime = at
	if open.chunk.NumSamples() < samplesPerChunk {
		return cutChunk{}, false
	}
	delete(o.chunks, id)
	return open.cutFor(id), true
}

func (o *openChunks) cutAll() []cutChunk {
	return o.cutIdleSince(math.MaxInt64)
}

func (o *openChunks) cutIdleSince(idleSince int64) []cutChunk {
	var cut []cutChunk
	for id, open := range o.chunks {
		if open.maxTime < idleSince {
			cut = append(cut, open.cutFor(id))
			delete(o.chunks, id)
		}
	}
	return cut
}

func (o *openChunks) copyWithin(id seriesID, mint, maxt int64) (chunks.Meta, bool) {
	open, exists := o.chunks[id]
	if !exists || open.maxTime < mint || open.minTime > maxt {
		return chunks.Meta{}, false
	}
	chunk := chunkenc.NewXORChunk()
	chunk.Reset(bytes.Clone(open.chunk.Bytes()))
	return chunks.Meta{Chunk: chunk, MinTime: open.minTime, MaxTime: open.maxTime}, true
}

func (o *openChunks) withoutOpenChunk(ids []seriesID) []seriesID {
	var without []seriesID
	for _, id := range ids {
		if _, exists := o.chunks[id]; !exists {
			without = append(without, id)
		}
	}
	return without
}
