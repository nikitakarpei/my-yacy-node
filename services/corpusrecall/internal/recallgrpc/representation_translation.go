package recallgrpc

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/corpusrecall/internal/pagerecall"
	corpusrecallv1 "github.com/nikitakarpei/yacy-rwi-node/corpusrecallapi/corpusrecall/v1"
)

type RepresentationEncode func(pagerecall.Representation) *corpusrecallv1.Representation

type RepresentationCodec struct {
	Kind      pagerecall.Kind
	ProtoKind corpusrecallv1.RepresentationKind
	Encode    RepresentationEncode
}

type representationTranslation struct {
	kindByProto  map[corpusrecallv1.RepresentationKind]pagerecall.Kind
	protoByKind  map[pagerecall.Kind]corpusrecallv1.RepresentationKind
	encodeByKind map[pagerecall.Kind]RepresentationEncode
}

func newRepresentationTranslation(codecs []RepresentationCodec) representationTranslation {
	translation := representationTranslation{
		kindByProto:  make(map[corpusrecallv1.RepresentationKind]pagerecall.Kind, len(codecs)),
		protoByKind:  make(map[pagerecall.Kind]corpusrecallv1.RepresentationKind, len(codecs)),
		encodeByKind: make(map[pagerecall.Kind]RepresentationEncode, len(codecs)),
	}
	for _, codec := range codecs {
		translation.kindByProto[codec.ProtoKind] = codec.Kind
		translation.protoByKind[codec.Kind] = codec.ProtoKind
		translation.encodeByKind[codec.Kind] = codec.Encode
	}
	return translation
}

func (t representationTranslation) requestedKinds(
	protoKinds []corpusrecallv1.RepresentationKind,
) ([]pagerecall.Kind, error) {
	kinds := make([]pagerecall.Kind, 0, len(protoKinds))
	for _, protoKind := range protoKinds {
		kind, ok := t.kindByProto[protoKind]
		if !ok {
			return nil, fmt.Errorf("unknown representation kind %s", protoKind)
		}
		kinds = append(kinds, kind)
	}
	return kinds, nil
}

func (t representationTranslation) recallResponse(
	result pagerecall.Result,
) *corpusrecallv1.RecallResponse {
	representations := make([]*corpusrecallv1.Representation, 0, len(result.Representations))
	for _, recalled := range result.Representations {
		representations = append(representations, t.encodeByKind[recalled.Kind](recalled.Content))
	}
	unavailable := make([]corpusrecallv1.RepresentationKind, 0, len(result.Unavailable))
	for _, kind := range result.Unavailable {
		unavailable = append(unavailable, t.protoByKind[kind])
	}
	return &corpusrecallv1.RecallResponse{
		Representations: representations,
		Unavailable:     unavailable,
	}
}
