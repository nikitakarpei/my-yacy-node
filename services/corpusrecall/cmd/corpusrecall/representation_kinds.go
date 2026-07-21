package main

import (
	"github.com/nikitakarpei/yacy-rwi-node/corpusrecall/internal/pagerecall"
	"github.com/nikitakarpei/yacy-rwi-node/corpusrecall/internal/recallgrpc"
	corpusrecallv1 "github.com/nikitakarpei/yacy-rwi-node/corpusrecallapi/corpusrecall/v1"
)

type representationKind struct {
	kind      pagerecall.Kind
	protoKind corpusrecallv1.RepresentationKind
	source    pagerecall.Source
	encode    recallgrpc.RepresentationEncode
}

func recallSources(kinds []representationKind) map[pagerecall.Kind]pagerecall.Source {
	sources := make(map[pagerecall.Kind]pagerecall.Source, len(kinds))
	for _, kind := range kinds {
		sources[kind.kind] = kind.source
	}
	return sources
}

func representationCodecs(kinds []representationKind) []recallgrpc.RepresentationCodec {
	codecs := make([]recallgrpc.RepresentationCodec, 0, len(kinds))
	for _, kind := range kinds {
		codecs = append(codecs, recallgrpc.RepresentationCodec{
			Kind:      kind.kind,
			ProtoKind: kind.protoKind,
			Encode:    kind.encode,
		})
	}
	return codecs
}
