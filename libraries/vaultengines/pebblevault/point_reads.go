package pebblevault

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/cockroachdb/pebble/v2"
)

type pointReader interface {
	valueAt(absoluteKey []byte) ([]byte, error)
}

type pointReaders interface {
	readerWithin(region keyspaceRegion) pointReader
}

type lookupPointReader struct {
	reader pebble.Reader
}

func (r lookupPointReader) readerWithin(keyspaceRegion) pointReader { return r }

func (r lookupPointReader) valueAt(absoluteKey []byte) ([]byte, error) {
	value, valueHandle, err := r.reader.Get(absoluteKey)
	if errors.Is(err, pebble.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, readFailureOf(err)
	}

	return copyOf(value), release(valueHandle)
}

func copyOf(value []byte) []byte {
	copied := make([]byte, len(value))
	copy(copied, value)

	return copied
}

type iteratorPointReaders struct {
	reader           pebble.Reader
	readersPerRegion map[string]*iteratorPointReader
}

func (r *iteratorPointReaders) readerWithin(region keyspaceRegion) pointReader {
	if reader, opened := r.readersPerRegion[string(region.prefix)]; opened {
		return reader
	}
	if r.readersPerRegion == nil {
		r.readersPerRegion = map[string]*iteratorPointReader{}
	}
	reader := &iteratorPointReader{reader: r.reader}
	r.readersPerRegion[string(region.prefix)] = reader

	return reader
}

func (r *iteratorPointReaders) release() error {
	releaseErrors := make([]error, 0, len(r.readersPerRegion))
	for _, reader := range r.readersPerRegion {
		releaseErrors = append(releaseErrors, reader.release())
	}

	return errors.Join(releaseErrors...)
}

type iteratorPointReader struct {
	reader   pebble.Reader
	iterator *pebble.Iterator
}

func (r *iteratorPointReader) valueAt(absoluteKey []byte) ([]byte, error) {
	iterator, err := r.openedIterator()
	if err != nil {
		return nil, err
	}
	if !iterator.SeekPrefixGE(absoluteKey) || !bytes.Equal(iterator.Key(), absoluteKey) {
		return nil, readFailureOf(iterator.Error())
	}
	value, err := iterator.ValueAndErr()
	if err != nil {
		return nil, readFailureOf(err)
	}

	return copyOf(value), nil
}

func (r *iteratorPointReader) openedIterator() (*pebble.Iterator, error) {
	if r.iterator != nil {
		return r.iterator, nil
	}
	iterator, err := r.reader.NewIter(nil)
	if err != nil {
		return nil, readFailureOf(err)
	}
	r.iterator = iterator

	return iterator, nil
}

func (r *iteratorPointReader) release() error {
	if r.iterator == nil {
		return nil
	}

	return release(r.iterator)
}

func readFailureOf(err error) error {
	if err == nil {
		return nil
	}

	return fmt.Errorf("read: %w", err)
}
