package yacyproto

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type HelloRequest struct {
	NetworkName string
	Key         string
	Seed        yacymodel.Seed
	Count       int
	Iam         yacymodel.Hash
	MagicMD5    string
	MyTime      time.Time
}

type HelloResponse struct {
	ResponseHeader
	YourIP   string
	YourType yacymodel.Optional[yacymodel.PeerType]
	MyTime   time.Time
	Message  string
	Seeds    []yacymodel.Seed
}

func (r HelloResponse) OwnSeed() yacymodel.Optional[yacymodel.Seed] {
	if len(r.Seeds) == 0 {
		return yacymodel.None[yacymodel.Seed]()
	}

	return yacymodel.Some(r.Seeds[0])
}

func (r HelloResponse) KnownSeeds() []yacymodel.Seed {
	if len(r.Seeds) < 2 {
		return nil
	}

	return r.Seeds[1:]
}

func (r HelloRequest) Form() url.Values {
	form := url.Values{}
	putString(form, FieldNetworkName, r.NetworkName)
	putString(form, FieldKey, r.Key)
	putString(form, FieldSeed, seedWireCodec{}.encode(r.Seed))
	putInt(form, FieldCount, r.Count)
	putString(form, FieldIam, r.Iam.String())
	putString(form, FieldMagicMD5, r.MagicMD5)
	putInstant(form, FieldMyTime, r.MyTime)

	return form
}

func ParseHelloRequest(ctx context.Context, form url.Values) (HelloRequest, error) {
	count, err := optionalInt(FieldCount, form.Get(FieldCount))
	if err != nil {
		return HelloRequest{}, err
	}

	myTime, err := optionalInstant(FieldMyTime, form.Get(FieldMyTime))
	if err != nil {
		return HelloRequest{}, err
	}

	req := HelloRequest{
		NetworkName: form.Get(FieldNetworkName),
		Key:         form.Get(FieldKey),
		Count:       count,
		MagicMD5:    form.Get(FieldMagicMD5),
		MyTime:      myTime,
	}

	raw := form.Get(FieldSeed)
	if raw == "" {
		return HelloRequest{}, fmt.Errorf("hello request: missing %s", FieldSeed)
	}
	req.Seed, err = decodeSeed(ctx, raw)
	if err != nil {
		return HelloRequest{}, err
	}

	if raw := form.Get(FieldIam); raw != "" {
		req.Iam, err = yacymodel.ParseHash(raw)
		if err != nil {
			return HelloRequest{}, fmt.Errorf("hello request %s: %w", FieldIam, err)
		}
	}

	return req, nil
}

func (r HelloResponse) Encode() Message {
	msg := Message{}
	setString(msg, FieldYourIP, r.YourIP)
	if yourType, ok := r.YourType.Get(); ok {
		setString(msg, FieldYourType, yourType.String())
	}
	setInstant(msg, FieldMyTime, r.MyTime)
	setString(msg, FieldMessage, r.Message)
	for i, seed := range r.Seeds {
		setString(msg, indexedKey(prefixSeed, i), seedWireCodec{}.encode(seed))
	}

	return msg
}

func ParseHelloResponse(ctx context.Context, m Message) (HelloResponse, error) {
	header, err := parseResponseHeader(m)
	if err != nil {
		return HelloResponse{}, err
	}

	myTime, err := optionalInstant(FieldMyTime, m[FieldMyTime])
	if err != nil {
		return HelloResponse{}, err
	}

	resp := HelloResponse{
		ResponseHeader: header,
		YourIP:         m[FieldYourIP],
		MyTime:         myTime,
		Message:        m[FieldMessage],
	}

	if raw := m[FieldYourType]; raw != "" {
		yourType, err := yacymodel.ParsePeerType(raw)
		if err != nil {
			return HelloResponse{}, fmt.Errorf("hello response %s: %w", FieldYourType, err)
		}
		resp.YourType = yacymodel.Some(yourType)
	}

	resp.Seeds, err = decodeSeeds(ctx, m)
	if err != nil {
		return HelloResponse{}, err
	}

	return resp, nil
}

func decodeSeed(ctx context.Context, raw string) (yacymodel.Seed, error) {
	seed, err := seedWireCodec{}.decode(ctx, raw)
	if err != nil {
		return yacymodel.Seed{}, fmt.Errorf("seed: %w", err)
	}

	return seed, nil
}

// decodeSeeds decodes the response's seed list. Index 0 is the responding
// peer's own seed and must parse for the response to be usable; a malformed
// entry beyond that is a peer we can't identify, not a reason to discard the
// rest of the batch, so it is dropped with a WARN instead.
func decodeSeeds(ctx context.Context, m Message) ([]yacymodel.Seed, error) {
	var seeds []yacymodel.Seed
	for i := 0; ; i++ {
		raw, ok := m[indexedKey(prefixSeed, i)]
		if !ok {
			return seeds, nil
		}

		seed, err := decodeSeed(ctx, raw)
		if err != nil {
			if i == 0 {
				return nil, err
			}

			slog.WarnContext(
				ctx,
				"dropped malformed known seed from hello response",
				slog.Int("index", i),
				slog.Any("error", err),
			)

			continue
		}

		seeds = append(seeds, seed)
	}
}
