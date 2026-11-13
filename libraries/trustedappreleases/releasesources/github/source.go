// Package github reads the releases of a repository through the GitHub REST API, which it
// names by repository ID. One call lists the release tags. A release takes three calls at
// most: the release, the reference of its tag, and the tag object of an annotated tag. Its
// release record and Sigstore bundle come from their download addresses.
package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/trustedappreleases"
)

const (
	recordAsset      = "release.json"
	bundleAsset      = "release.sigstore.json"
	listedReleases   = 100
	maxResponseBytes = 4 << 20
	annotatedTagType = "tag"
	commitType       = "commit"
)

var (
	ErrUnexpectedStatus = errors.New("GitHub answered with an unexpected status")
	ErrTooLarge         = errors.New("the GitHub answer is too large")
	ErrMissingAsset     = errors.New("the release has no such asset")
	ErrTagWithoutCommit = errors.New("the tag does not point to a commit")
)

type Source struct {
	client     *http.Client
	repository string
}

type publishedRelease struct {
	TagName   string  `json:"tag_name"`
	Assets    []asset `json:"assets"`
	Immutable bool    `json:"immutable"`
}

type asset struct {
	Name        string `json:"name"`
	DownloadURL string `json:"browser_download_url"`
}

type gitObject struct {
	Object struct {
		SHA  string `json:"sha"`
		Type string `json:"type"`
	} `json:"object"`
}

func New(client *http.Client, apiAddress, repositoryID string) Source {
	return Source{
		client:     client,
		repository: apiAddress + "/repositories/" + url.PathEscape(repositoryID),
	}
}

func (s Source) TagsOldestFirst(ctx context.Context) ([]string, error) {
	var releases []publishedRelease
	if err := s.decoded(
		ctx,
		s.repository+fmt.Sprintf("/releases?per_page=%d", listedReleases),
		&releases,
	); err != nil {
		return nil, err
	}
	tags := make([]string, 0, len(releases))
	for _, release := range slices.Backward(releases) {
		tags = append(tags, release.TagName)
	}

	return tags, nil
}

func (s Source) ReleaseOf(ctx context.Context, tag string) (trustedappreleases.Release, error) {
	var release publishedRelease
	if err := s.decoded(
		ctx,
		s.repository+"/releases/tags/"+url.PathEscape(tag),
		&release,
	); err != nil {
		return trustedappreleases.Release{}, err
	}
	commit, err := s.commitOf(ctx, tag)
	if err != nil {
		return trustedappreleases.Release{}, err
	}
	record, err := s.assetOf(ctx, release, recordAsset)
	if err != nil {
		return trustedappreleases.Release{}, err
	}
	signedBundle, err := s.assetOf(ctx, release, bundleAsset)
	if err != nil {
		return trustedappreleases.Release{}, err
	}

	return trustedappreleases.Release{
		Tag:          tag,
		Commit:       commit,
		Record:       record,
		SignedBundle: signedBundle,
		Immutable:    release.Immutable,
	}, nil
}

func (s Source) decoded(ctx context.Context, address string, value any) error {
	body, err := s.bodyOf(ctx, address)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(body, value); err != nil {
		return fmt.Errorf("read the GitHub answer of %s: %w", address, err)
	}

	return nil
}

func (s Source) bodyOf(ctx context.Context, address string) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return nil, fmt.Errorf("ask GitHub for %s: %w", address, err)
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	response, err := s.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("ask GitHub for %s: %w", address, err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: %d for %s", ErrUnexpectedStatus, response.StatusCode, address)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read the GitHub answer of %s: %w", address, err)
	}
	if len(body) > maxResponseBytes {
		return nil, fmt.Errorf("%w: %s", ErrTooLarge, address)
	}

	return body, nil
}

func (s Source) commitOf(ctx context.Context, tag string) (string, error) {
	var reference gitObject
	if err := s.decoded(
		ctx,
		s.repository+"/git/ref/tags/"+url.PathEscape(tag),
		&reference,
	); err != nil {
		return "", err
	}
	if reference.Object.Type == annotatedTagType {
		if err := s.decoded(
			ctx,
			s.repository+"/git/tags/"+reference.Object.SHA,
			&reference,
		); err != nil {
			return "", err
		}
	}
	if reference.Object.Type != commitType {
		return "", fmt.Errorf("%w: %s", ErrTagWithoutCommit, tag)
	}

	return reference.Object.SHA, nil
}

func (s Source) assetOf(
	ctx context.Context,
	release publishedRelease,
	name string,
) ([]byte, error) {
	for _, candidate := range release.Assets {
		if candidate.Name == name {
			return s.bodyOf(ctx, candidate.DownloadURL)
		}
	}

	return nil, fmt.Errorf("%w: %s of %s", ErrMissingAsset, name, release.TagName)
}
