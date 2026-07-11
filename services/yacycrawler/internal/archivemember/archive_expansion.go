package archivemember

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"path"
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
)

const (
	mediaZip  = "application/zip"
	mediaTar  = "application/x-tar"
	mediaGzip = "application/gzip"

	memberSeparator = "!/"

	msgMemberOpenFailed = "archive member open failed, skipped"
	msgMemberSkipped    = "archive member skipped"
)

type ArchiveExpansion struct {
	maxMembers     int
	maxMemberBytes int64
}

func New(maxMembers int, maxMemberBytes int64) ArchiveExpansion {
	return ArchiveExpansion{maxMembers: maxMembers, maxMemberBytes: maxMemberBytes}
}

func (ArchiveExpansion) MediaTypes() []string {
	return []string{mediaZip, mediaTar, mediaGzip}
}

func (a ArchiveExpansion) Expand(
	ctx context.Context,
	containerURL, contentType string,
	body []byte,
) ([]crawlcapability.ArchiveMember, error) {
	switch mediaType(contentType) {
	case mediaZip:
		return a.expandZip(ctx, containerURL, body)
	case mediaTar:
		return a.expandTar(ctx, containerURL, bytes.NewReader(body))
	case mediaGzip:
		return a.expandGzip(ctx, containerURL, body)
	default:
		return nil, crawlcapability.ErrUnsupportedMediaType
	}
}

func (a ArchiveExpansion) expandZip(
	ctx context.Context,
	containerURL string,
	body []byte,
) ([]crawlcapability.ArchiveMember, error) {
	reader, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return nil, fmt.Errorf("open zip: %w", err)
	}
	var members []crawlcapability.ArchiveMember
	for _, file := range reader.File {
		if file.FileInfo().IsDir() {
			continue
		}
		opened, err := file.Open()
		if err != nil {
			slog.WarnContext(ctx, msgMemberOpenFailed,
				slog.String("member", file.Name),
				slog.Any("error", err),
			)
			continue
		}
		content, err := a.readMember(opened)
		_ = opened.Close()
		if err != nil {
			logMemberSkipped(ctx, file.Name, err)
			continue
		}
		member, ok := a.member(containerURL, file.Name, content)
		if !ok {
			continue
		}
		members = append(members, member)
		if len(members) > a.maxMembers {
			return nil, crawlcapability.ErrContainerOverflow
		}
	}
	return members, nil
}

func (a ArchiveExpansion) expandGzip(
	ctx context.Context,
	containerURL string,
	body []byte,
) ([]crawlcapability.ArchiveMember, error) {
	decompressed, err := gzip.NewReader(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("open gzip: %w", err)
	}
	content, err := a.readMember(decompressed)
	if err != nil {
		logMemberSkipped(ctx, decompressedName(containerURL), err)
		return nil, nil
	}
	member, ok := a.member(containerURL, decompressedName(containerURL), content)
	if !ok {
		return nil, nil
	}
	return []crawlcapability.ArchiveMember{member}, nil
}

func decompressedName(containerURL string) string {
	return strings.TrimSuffix(path.Base(containerURL), path.Ext(containerURL))
}

func (a ArchiveExpansion) expandTar(
	ctx context.Context,
	containerURL string,
	source io.Reader,
) ([]crawlcapability.ArchiveMember, error) {
	reader := tar.NewReader(source)
	var members []crawlcapability.ArchiveMember
	for {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read tar: %w", err)
		}
		if header.Typeflag != tar.TypeReg {
			continue
		}
		content, err := a.readMember(reader)
		if err != nil {
			logMemberSkipped(ctx, header.Name, err)
			continue
		}
		member, ok := a.member(containerURL, header.Name, content)
		if !ok {
			continue
		}
		members = append(members, member)
		if len(members) > a.maxMembers {
			return nil, crawlcapability.ErrContainerOverflow
		}
	}
	return members, nil
}

func (a ArchiveExpansion) readMember(source io.Reader) ([]byte, error) {
	content, err := io.ReadAll(io.LimitReader(source, a.maxMemberBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read member: %w", err)
	}
	if int64(len(content)) > a.maxMemberBytes {
		return nil, fmt.Errorf("member exceeds size limit of %d bytes", a.maxMemberBytes)
	}
	return content, nil
}

func logMemberSkipped(ctx context.Context, name string, err error) {
	slog.WarnContext(ctx, msgMemberSkipped,
		slog.String("member", name),
		slog.Any("error", err),
	)
}

func (a ArchiveExpansion) member(
	containerURL, name string,
	content []byte,
) (crawlcapability.ArchiveMember, bool) {
	contentType := mime.TypeByExtension(path.Ext(name))
	if contentType == "" {
		return crawlcapability.ArchiveMember{}, false
	}
	return crawlcapability.ArchiveMember{
		URL:         containerURL + memberSeparator + name,
		ContentType: contentType,
		Body:        content,
	}, true
}

func mediaType(contentType string) string {
	media, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return strings.ToLower(strings.TrimSpace(strings.SplitN(contentType, ";", 2)[0]))
	}
	return media
}
