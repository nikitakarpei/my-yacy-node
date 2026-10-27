package metrichistory

import (
	"bytes"
	"math"

	"github.com/prometheus/prometheus/tsdb/chunkenc"
	"github.com/prometheus/prometheus/tsdb/chunks"
)

type openChunk struct {
	chunk      *chunkenc.XORChunk
	appender   chunkenc.Appender
	checkpoint *chunkKey
	minTime    int64
	maxTime    int64
}

func openChunkFrom(at int64) *openChunk {
	chunk := chunkenc.NewXORChunk()
	appender, _ := chunk.Appender()
	return &openChunk{chunk: chunk, appender: appender, minTime: at, maxTime: at}
}

func (c *openChunk) revisionFor(id seriesID) chunkRevision {
	revision := chunkRevision{
		key:     chunkKey{series: id, maxTime: c.maxTime},
		encoded: encodedChunk{data: c.chunk.Bytes(), minTime: c.minTime},
	}
	if c.checkpoint != nil && *c.checkpoint != revision.key {
		revision.replacedCheckpoint = c.checkpoint
	}
	return revision
}

func (c *openChunk) isCheckpointDue() bool {
	return c.checkpoint == nil || c.checkpoint.maxTime != c.maxTime
}

type chunkRevision struct {
	encoded            encodedChunk
	key                chunkKey
	replacedCheckpoint *chunkKey
}

type openChunks struct {
	chunks map[seriesID]*openChunk
}

func newOpenChunks() *openChunks {
	return &openChunks{chunks: map[seriesID]*openChunk{}}
}

func (o *openChunks) append(
	id seriesID,
	at int64,
	value float64,
) (full chunkRevision, isFull bool) {
	open, exists := o.chunks[id]
	if !exists {
		open = openChunkFrom(at)
		o.chunks[id] = open
	} else if at <= open.maxTime {
		return chunkRevision{}, false
	}
	open.appender.Append(0, at, value)
	open.maxTime = at
	if open.chunk.NumSamples() < samplesPerChunk {
		return chunkRevision{}, false
	}
	delete(o.chunks, id)
	return open.revisionFor(id), true
}

func (o *openChunks) cutAll() []chunkRevision {
	return o.cutIdleSince(math.MaxInt64)
}

func (o *openChunks) cutIdleSince(idleSince int64) []chunkRevision {
	var cut []chunkRevision
	for id, open := range o.chunks {
		if open.maxTime < idleSince {
			cut = append(cut, open.revisionFor(id))
			delete(o.chunks, id)
		}
	}
	return cut
}

func (o *openChunks) dueCheckpoints() []chunkRevision {
	var dueCheckpoints []chunkRevision
	for id, open := range o.chunks {
		if open.isCheckpointDue() {
			dueCheckpoints = append(dueCheckpoints, open.revisionFor(id))
		}
	}
	return dueCheckpoints
}

func (o *openChunks) markCheckpointed(checkpoints []chunkRevision) {
	for _, checkpoint := range checkpoints {
		if open, exists := o.chunks[checkpoint.key.series]; exists {
			open.checkpoint = &checkpoint.key
		}
	}
}

func (o *openChunks) chunkWithin(id seriesID, mint, maxt int64) (chunks.Meta, bool) {
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
