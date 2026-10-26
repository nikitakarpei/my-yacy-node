package yacyproto

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	wireFramePlain  = 'p'
	wireFrameBase64 = 'b'
	wireFrameGzip   = 'z'
	wireFrameSep    = '|'
)

var errBadWireFrame = errors.New("bad wire frame")

func encodeBase64WireForm(payload string) string {
	return tagged(wireFrameBase64, yacymodel.Encode([]byte(payload)))
}

func tagged(tag byte, body string) string {
	return string([]byte{tag, wireFrameSep}) + body
}

func decodeWireForm(
	ctx context.Context,
	form string,
	allowance *plainTextAllowance,
) (string, error) {
	plain, err := plainTextFrom(ctx, form, allowance)
	if err != nil {
		return "", err
	}
	if !allowance.take(plain) {
		return "", fmt.Errorf("%w: %d bytes", errPlainTextAllowanceSpent, len(plain))
	}

	return plain, nil
}

func plainTextFrom(
	ctx context.Context,
	form string,
	allowance *plainTextAllowance,
) (string, error) {
	if len(form) < 2 || form[1] != wireFrameSep {
		return form, nil
	}
	body := form[2:]
	switch form[0] {
	case wireFramePlain:
		return body, nil
	case wireFrameBase64:
		raw, err := yacymodel.Decode(body)
		if err != nil {
			return "", fmt.Errorf("decode wire frame body: %w", err)
		}
		return string(raw), nil
	case wireFrameGzip:
		raw, err := yacymodel.Decode(body)
		if err != nil {
			return "", fmt.Errorf("decode wire frame body: %w", err)
		}
		plain, err := gzipDecompress(ctx, raw, allowance)
		if err != nil {
			return "", fmt.Errorf("inflate wire frame body: %w", err)
		}
		return plain, nil
	default:
		return "", fmt.Errorf("%w: tag %q", errBadWireFrame, form[0])
	}
}

func gzipDecompress(ctx context.Context, b []byte, allowance *plainTextAllowance) (string, error) {
	r, err := gzip.NewReader(bytes.NewReader(b))
	if err != nil {
		return "", fmt.Errorf("gzip reader: %w", err)
	}
	defer func() {
		if err := r.Close(); err != nil {
			slog.WarnContext(
				ctx,
				"gzip reader close failed",
				slog.Any("error", err),
			)
		}
	}()
	out, err := io.ReadAll(allowance.readerWithin(r))
	if err != nil {
		return "", fmt.Errorf("gzip read: %w", err)
	}
	return string(out), nil
}
