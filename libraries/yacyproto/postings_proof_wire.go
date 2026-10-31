package yacyproto

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const postingsProofSeparator = "="

type postingsProofWireCodec struct{}

func (c postingsProofWireCodec) encodeLines(
	proofs map[yacymodel.URLHash]yacymodel.PostingsProof,
) string {
	lines := make([]string, 0, len(proofs))
	for urlHash, proof := range proofs {
		lines = append(lines, c.encode(urlHash, proof))
	}
	slices.Sort(lines)

	return strings.Join(lines, "\n")
}

func (postingsProofWireCodec) encode(
	urlHash yacymodel.URLHash,
	proof yacymodel.PostingsProof,
) string {
	return urlHash.String() + postingsProofSeparator + yacymodel.Encode(proof.Bytes())
}

func (c postingsProofWireCodec) decodeLinesFor(
	ctx context.Context,
	raw string,
	offeredPostings []yacymodel.RWIPosting,
) map[yacymodel.URLHash]yacymodel.PostingsProof {
	offeredURLHashes := urlHashesOf(offeredPostings)
	var (
		proofs         map[yacymodel.URLHash]yacymodel.PostingsProof
		discardedLines int
		firstError     error
	)
	for line := range strings.SplitSeq(raw, "\n") {
		line = strings.TrimRight(line, "\r")
		if line == "" {
			continue
		}
		if len(proofs) == len(offeredURLHashes) {
			discardedLines++

			continue
		}

		urlHash, proof, err := c.decode(line)
		_, offered := offeredURLHashes[urlHash]
		_, repeated := proofs[urlHash]
		if err != nil || !offered || repeated {
			discardedLines++
			firstError = cmp.Or(firstError, err)

			continue
		}
		if proofs == nil {
			proofs = make(map[yacymodel.URLHash]yacymodel.PostingsProof, len(offeredURLHashes))
		}
		proofs[urlHash] = proof
	}
	if discardedLines > 0 {
		slog.WarnContext(
			ctx,
			"transfer rwi postings proofs discarded",
			slog.Int("discardedLines", discardedLines),
			slog.Any("firstError", firstError),
		)
	}

	return proofs
}

func urlHashesOf(postings []yacymodel.RWIPosting) map[yacymodel.URLHash]struct{} {
	urlHashes := make(map[yacymodel.URLHash]struct{}, len(postings))
	for _, posting := range postings {
		urlHashes[posting.URLHash] = struct{}{}
	}

	return urlHashes
}

func (postingsProofWireCodec) decode(
	line string,
) (yacymodel.URLHash, yacymodel.PostingsProof, error) {
	rawURLHash, rawProof, separated := strings.Cut(line, postingsProofSeparator)
	if !separated {
		return yacymodel.URLHash{}, yacymodel.PostingsProof{}, fmt.Errorf(
			"%w: missing separator",
			yacymodel.ErrBadPostingsProof,
		)
	}
	urlHash, err := yacymodel.ParseURLHash(rawURLHash)
	if err != nil {
		return yacymodel.URLHash{}, yacymodel.PostingsProof{}, fmt.Errorf(
			"%w: url hash: %w",
			yacymodel.ErrBadPostingsProof,
			err,
		)
	}
	proofBytes, err := yacymodel.Decode(rawProof)
	if err != nil {
		return yacymodel.URLHash{}, yacymodel.PostingsProof{}, fmt.Errorf(
			"%w: %w",
			yacymodel.ErrBadPostingsProof,
			err,
		)
	}
	proof, err := yacymodel.ParsePostingsProofBytes(proofBytes)
	if err != nil {
		return yacymodel.URLHash{}, yacymodel.PostingsProof{}, err
	}

	return urlHash, proof, nil
}
