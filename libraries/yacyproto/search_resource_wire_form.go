package yacyproto

import (
	"context"
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type SearchResource struct {
	Metadata yacymodel.URLMetadata
	Posting  yacymodel.RWIPosting
}

const searchResourceColPosting = "wi"

type searchResourceWireCodec struct{}

func (searchResourceWireCodec) encode(resource SearchResource) string {
	form := urlMetadataWireFormFromDomain(resource.Metadata)
	if resource.Posting.URLHash.String() != "" {
		form.put(
			searchResourceColPosting,
			yacymodel.Encode([]byte(rwiPostingWireCodec{}.encodePropertyForm(resource.Posting))),
		)
	}

	return form.row()
}

func (searchResourceWireCodec) decode(
	ctx context.Context,
	row string,
) (SearchResource, error) {
	properties, err := propertyPairsOfRow(row)
	if err != nil {
		return SearchResource{}, fmt.Errorf("%w: %w", yacymodel.ErrBadURLMetadata, err)
	}

	metadata, err := urlMetadataWireForm{properties: properties}.domain(ctx)
	if err != nil {
		return SearchResource{}, fmt.Errorf("%w: %w", yacymodel.ErrBadURLMetadata, err)
	}

	posting, err := searchResourcePostingFrom(properties[searchResourceColPosting])
	if err != nil {
		return SearchResource{}, err
	}

	return SearchResource{Metadata: metadata, Posting: posting}, nil
}

func searchResourcePostingFrom(encoded string) (yacymodel.RWIPosting, error) {
	if encoded == "" {
		return yacymodel.RWIPosting{}, nil
	}
	form, err := yacymodel.Decode(encoded)
	if err != nil {
		return yacymodel.RWIPosting{}, fmt.Errorf("search resource posting: %w", err)
	}

	return rwiPostingWireCodec{}.decodePropertyForm(string(form))
}
