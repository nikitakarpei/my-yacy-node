package yacyproto

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type MessageProcess string

const (
	ProcessPermission MessageProcess = "permission"
	ProcessPost       MessageProcess = "post"
)

const (
	MessageResponseRefused  = "-1"
	MessageResponseWelcome  = "Welcome to my peer!"
	MessageResponseAccepted = "Thank you!"
)

type MessageRequest struct {
	NetworkName string
	YouAre      yacymodel.Hash
	Iam         yacymodel.Hash
	Key         string
	MagicMD5    string
	MyTime      time.Time
	Process     MessageProcess
	MySeed      yacymodel.Optional[yacymodel.Seed]
	Subject     string
	Body        string
}

type MessageResponse struct {
	ResponseHeader
	Response       string
	MessageSize    int
	AttachmentSize int
}

func (r MessageRequest) Form() url.Values {
	form := url.Values{}
	putString(form, FieldNetworkName, r.NetworkName)
	putString(form, FieldYouAre, r.YouAre.String())
	putString(form, FieldIam, r.Iam.String())
	putString(form, FieldKey, r.Key)
	putString(form, FieldMagicMD5, r.MagicMD5)
	putInstant(form, FieldMyTime, r.MyTime)
	putString(form, FieldProcess, string(r.Process))
	if seed, ok := r.MySeed.Get(); ok {
		putString(form, FieldMySeed, seedWireCodec{}.encode(seed))
	}
	putString(form, FieldSubject, r.Subject)
	putString(form, FieldMessage, r.Body)

	return form
}

func ParseMessageRequest(ctx context.Context, form url.Values) (MessageRequest, error) {
	myTime, err := optionalInstant(FieldMyTime, form.Get(FieldMyTime))
	if err != nil {
		return MessageRequest{}, err
	}

	req := MessageRequest{
		NetworkName: form.Get(FieldNetworkName),
		Key:         form.Get(FieldKey),
		MagicMD5:    form.Get(FieldMagicMD5),
		MyTime:      myTime,
		Subject:     form.Get(FieldSubject),
		Body:        form.Get(FieldMessage),
	}

	req.Process, err = parseMessageProcess(form.Get(FieldProcess))
	if err != nil {
		return MessageRequest{}, err
	}

	req.YouAre, err = parseHashField("message request", FieldYouAre, form.Get(FieldYouAre))
	if err != nil {
		return MessageRequest{}, err
	}

	req.Iam, err = parseHashField("message request", FieldIam, form.Get(FieldIam))
	if err != nil {
		return MessageRequest{}, err
	}

	req.MySeed, err = optionalSeed(ctx, form.Get(FieldMySeed))
	if err != nil {
		return MessageRequest{}, err
	}

	return req, nil
}

func parseMessageProcess(raw string) (MessageProcess, error) {
	if raw == "" {
		return ProcessPermission, nil
	}

	process := MessageProcess(raw)
	switch process {
	case ProcessPermission, ProcessPost:
		return process, nil
	default:
		return "", fmt.Errorf("%w: %s=%q", ErrBadField, FieldProcess, raw)
	}
}

func optionalSeed(ctx context.Context, raw string) (yacymodel.Optional[yacymodel.Seed], error) {
	if raw == "" {
		return yacymodel.None[yacymodel.Seed](), nil
	}

	seed, err := decodeSeed(ctx, raw)
	if err != nil {
		return yacymodel.None[yacymodel.Seed](), err
	}

	return yacymodel.Some(seed), nil
}

func (r MessageResponse) Encode() ResponseFields {
	fields := ResponseFields{}
	setString(fields, FieldResponse, r.Response)
	setInt(fields, FieldMessageSize, r.MessageSize)
	setInt(fields, FieldAttachmentSize, r.AttachmentSize)

	return fields
}

func ParseMessageResponse(m ResponseFields) (MessageResponse, error) {
	header, err := parseResponseHeader(m)
	if err != nil {
		return MessageResponse{}, err
	}

	messageSize, err := optionalInt(FieldMessageSize, m[FieldMessageSize])
	if err != nil {
		return MessageResponse{}, err
	}

	attachmentSize, err := optionalInt(FieldAttachmentSize, m[FieldAttachmentSize])
	if err != nil {
		return MessageResponse{}, err
	}

	return MessageResponse{
		ResponseHeader: header,
		Response:       m[FieldResponse],
		MessageSize:    messageSize,
		AttachmentSize: attachmentSize,
	}, nil
}
