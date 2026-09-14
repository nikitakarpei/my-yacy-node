package yacyproto

import (
	"context"
	"net/url"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const QueryResponseRejected = -1

type QueryRequest struct {
	NetworkName string
	YouAre      yacymodel.Hash
	Iam         yacymodel.Hash
	Object      QueryObject
	Env         string
	Key         string
	MagicMD5    string
	MyTime      time.Time
}

type QueryResponse struct {
	ResponseHeader
	Response int
	MyTime   time.Time
	Magic    string
}

func (r QueryRequest) Form() url.Values {
	form := url.Values{}
	putString(form, FieldNetworkName, r.NetworkName)
	putString(form, FieldYouAre, r.YouAre.String())
	putString(form, FieldIam, r.Iam.String())
	putString(form, FieldObject, string(r.Object))
	putString(form, FieldEnv, r.Env)
	putString(form, FieldKey, r.Key)
	putString(form, FieldMagicMD5, r.MagicMD5)
	putInstant(form, FieldMyTime, r.MyTime)

	return form
}

func ParseQueryRequest(_ context.Context, form url.Values) (QueryRequest, error) {
	myTime, err := optionalInstant(FieldMyTime, form.Get(FieldMyTime))
	if err != nil {
		return QueryRequest{}, err
	}

	req := QueryRequest{
		NetworkName: form.Get(FieldNetworkName),
		Env:         form.Get(FieldEnv),
		Key:         form.Get(FieldKey),
		MagicMD5:    form.Get(FieldMagicMD5),
		MyTime:      myTime,
	}

	req.Object, err = parseQueryObject(form.Get(FieldObject))
	if err != nil {
		return QueryRequest{}, err
	}

	req.YouAre, err = parseHashField("query request", FieldYouAre, form.Get(FieldYouAre))
	if err != nil {
		return QueryRequest{}, err
	}

	req.Iam, err = parseHashField("query request", FieldIam, form.Get(FieldIam))
	if err != nil {
		return QueryRequest{}, err
	}

	return req, nil
}

func (r QueryResponse) Encode() Message {
	msg := Message{}
	setInt(msg, FieldResponse, r.Response)
	setInstant(msg, FieldMyTime, r.MyTime)
	setString(msg, FieldMagic, r.Magic)

	return msg
}

func ParseQueryResponse(m Message) (QueryResponse, error) {
	header, err := parseResponseHeader(m)
	if err != nil {
		return QueryResponse{}, err
	}

	response, err := optionalInt(FieldResponse, m[FieldResponse])
	if err != nil {
		return QueryResponse{}, err
	}

	myTime, err := optionalInstant(FieldMyTime, m[FieldMyTime])
	if err != nil {
		return QueryResponse{}, err
	}

	return QueryResponse{
		ResponseHeader: header,
		Response:       response,
		MyTime:         myTime,
		Magic:          m[FieldMagic],
	}, nil
}
