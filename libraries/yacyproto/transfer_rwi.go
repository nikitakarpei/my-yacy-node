package yacyproto

import (
	"context"
	"net/url"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type TransferRWIRequest struct {
	NetworkName string
	Iam         yacymodel.Hash
	YouAre      yacymodel.Hash
	WordCount   int
	EntryCount  int
	Indexes     []yacymodel.RWIPosting
	Key         string
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

	return req, nil
}

func (r TransferRWIResponse) Encode() Message {
	msg := Message{}
	setString(msg, FieldResult, string(r.Result))
	setInt(msg, FieldPause, int(r.Pause/time.Millisecond))
	msg[FieldUnknownURL] = joinURLHashes(r.UnknownURL)
	msg[FieldErrorURL] = joinURLHashes(r.ErrorURL)

	return msg
}

func ParseTransferRWIResponse(m Message) (TransferRWIResponse, error) {
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
