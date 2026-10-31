package yacyproto

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const MaxTransferRWIEntryCount = 1000

func TransferRWIKey(entryCount int) string {
	return MagicMD5("", "", fmt.Sprintf("%d", entryCount))
}

type TransferRWIRequest struct {
	NetworkName    string
	Iam            yacymodel.Hash
	YouAre         yacymodel.Hash
	WordCount      int
	EntryCount     int
	Indexes        []yacymodel.RWIPosting
	PostingsProofs map[yacymodel.URLHash]yacymodel.PostingsProof
	Key            string
}

type TransferRWIResponse struct {
	ResponseHeader
	Result     TransferRWIResult
	Pause      time.Duration
	UnknownURL []yacymodel.URLHash
	ErrorURL   []yacymodel.URLHash
}

func (r TransferRWIRequest) Form() url.Values {
	form := url.Values{}
	putString(form, FieldNetworkName, r.NetworkName)
	putString(form, FieldIam, r.Iam.String())
	putString(form, FieldYouAre, r.YouAre.String())
	putInt(form, FieldWordCount, r.WordCount)
	putInt(form, FieldEntryCount, r.EntryCount)
	putString(form, FieldIndexes, rwiPostingWireCodec{}.encodeLines(r.Indexes))
	putString(
		form,
		FieldPostingsProofs,
		postingsProofWireCodec{}.encodeLines(r.PostingsProofs),
	)
	putString(form, FieldKey, r.Key)

	return form
}

func ParseTransferRWIRequest(ctx context.Context, form url.Values) (TransferRWIRequest, error) {
	wordCount, err := optionalInt(FieldWordCount, form.Get(FieldWordCount))
	if err != nil {
		return TransferRWIRequest{}, err
	}

	entryCount, err := optionalInt(FieldEntryCount, form.Get(FieldEntryCount))
	if err != nil {
		return TransferRWIRequest{}, err
	}

	req := TransferRWIRequest{
		NetworkName: form.Get(FieldNetworkName),
		WordCount:   wordCount,
		EntryCount:  entryCount,
		Key:         form.Get(FieldKey),
	}

	req.Iam, err = parseHashField("transferRWI request", FieldIam, form.Get(FieldIam))
	if err != nil {
		return TransferRWIRequest{}, err
	}

	req.YouAre, err = parseHashField("transferRWI request", FieldYouAre, form.Get(FieldYouAre))
	if err != nil {
		return TransferRWIRequest{}, err
	}

	req.Indexes = rwiPostingWireCodec{}.decodeLines(ctx, form.Get(FieldIndexes))
	req.PostingsProofs = postingsProofWireCodec{}.decodeLinesFor(
		ctx,
		form.Get(FieldPostingsProofs),
		req.Indexes,
	)

	return req, nil
}

func (r TransferRWIResponse) Encode() ResponseFields {
	fields := ResponseFields{}
	setString(fields, FieldResult, string(r.Result))
	setInt(fields, FieldPause, int(r.Pause/time.Millisecond))
	fields[FieldUnknownURL] = joinURLHashes(r.UnknownURL)
	fields[FieldErrorURL] = joinURLHashes(r.ErrorURL)

	return fields
}

func ParseTransferRWIResponse(m ResponseFields) (TransferRWIResponse, error) {
	header, err := parseResponseHeader(m)
	if err != nil {
		return TransferRWIResponse{}, err
	}

	pause, err := optionalInt(FieldPause, m[FieldPause])
	if err != nil {
		return TransferRWIResponse{}, err
	}

	unknown, err := splitURLHashes("transferRWI response", FieldUnknownURL, m[FieldUnknownURL])
	if err != nil {
		return TransferRWIResponse{}, err
	}

	errorURL, err := splitURLHashes("transferRWI response", FieldErrorURL, m[FieldErrorURL])
	if err != nil {
		return TransferRWIResponse{}, err
	}

	result, err := parseTransferRWIResult(m[FieldResult])
	if err != nil {
		return TransferRWIResponse{}, err
	}

	return TransferRWIResponse{
		ResponseHeader: header,
		Result:         result,
		Pause:          time.Duration(pause) * time.Millisecond,
		UnknownURL:     unknown,
		ErrorURL:       errorURL,
	}, nil
}
