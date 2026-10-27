package yacyproto

import (
	"context"
	"log/slog"
	"net/url"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type TransferURLRequest struct {
	NetworkName string
	Iam         yacymodel.Hash
	YouAre      yacymodel.Hash
	URLCount    int
	URLs        []yacymodel.URLMetadata
}

type TransferURLResponse struct {
	ResponseHeader
	Result   TransferURLResult
	Double   int
	ErrorURL []yacymodel.URLHash
}

func (r TransferURLRequest) Form() url.Values {
	form := url.Values{}
	putString(form, FieldNetworkName, r.NetworkName)
	putString(form, FieldIam, r.Iam.String())
	putString(form, FieldYouAre, r.YouAre.String())
	putInt(form, FieldURLCount, r.URLCount)
	for i, row := range r.URLs {
		putString(form, indexedKey(prefixURL, i), urlMetadataWireCodec{}.encode(row))
	}

	return form
}

func ParseTransferURLRequest(ctx context.Context, form url.Values) (TransferURLRequest, error) {
	urlCount, err := optionalInt(FieldURLCount, form.Get(FieldURLCount))
	if err != nil {
		return TransferURLRequest{}, err
	}

	req := TransferURLRequest{
		NetworkName: form.Get(FieldNetworkName),
		URLCount:    urlCount,
	}

	req.Iam, err = parseHashField("transferURL request", FieldIam, form.Get(FieldIam))
	if err != nil {
		return TransferURLRequest{}, err
	}

	req.YouAre, err = parseHashField("transferURL request", FieldYouAre, form.Get(FieldYouAre))
	if err != nil {
		return TransferURLRequest{}, err
	}

	req.URLs = urlMetadataFromForm(ctx, form, req.URLCount)

	return req, nil
}

func urlMetadataFromForm(
	ctx context.Context,
	form url.Values,
	declaredURLCount int,
) []yacymodel.URLMetadata {
	if declaredURLCount > maxRWIPostingsPerTransfer {
		slog.WarnContext(
			ctx,
			"transfer url count above limit",
			slog.Int("urlCount", declaredURLCount),
			slog.Int("limit", maxRWIPostingsPerTransfer),
		)
	}

	allowance := &plainTextAllowance{remainingBytes: messageMaxPlainBytes}
	var metadata []yacymodel.URLMetadata
	missingRows := 0
	for i := range min(declaredURLCount, maxRWIPostingsPerTransfer) {
		raw := form.Get(indexedKey(prefixURL, i))
		if raw == "" {
			missingRows++
			continue
		}

		row, err := urlMetadataWireCodec{}.decode(ctx, raw, allowance)
		if err != nil {
			slog.WarnContext(
				ctx,
				"transfer url row discarded",
				slog.String("reason", "parse failed"),
				slog.Int("index", i),
				slog.Any("error", err),
			)
			continue
		}

		metadata = append(metadata, row)
	}

	if missingRows > 0 {
		slog.WarnContext(
			ctx,
			"transfer url rows missing",
			slog.Int("missingRows", missingRows),
		)
	}

	return metadata
}

func (r TransferURLResponse) Encode() ResponseFields {
	fields := ResponseFields{}
	setString(fields, FieldResult, string(r.Result))
	setInt(fields, FieldDouble, r.Double)
	setString(fields, FieldErrorURL, joinURLHashes(r.ErrorURL))

	return fields
}

func ParseTransferURLResponse(m ResponseFields) (TransferURLResponse, error) {
	header, err := parseResponseHeader(m)
	if err != nil {
		return TransferURLResponse{}, err
	}

	double, err := optionalInt(FieldDouble, m[FieldDouble])
	if err != nil {
		return TransferURLResponse{}, err
	}

	errorURL, err := splitURLHashes("transferURL response", FieldErrorURL, m[FieldErrorURL])
	if err != nil {
		return TransferURLResponse{}, err
	}

	result, err := parseTransferURLResult(m[FieldResult])
	if err != nil {
		return TransferURLResponse{}, err
	}

	return TransferURLResponse{
		ResponseHeader: header,
		Result:         result,
		Double:         double,
		ErrorURL:       errorURL,
	}, nil
}
