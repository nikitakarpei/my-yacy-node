package yacymodel

import (
	"errors"
	"fmt"
)

var ErrBadNewsCategory = errors.New("bad news category")

// NewsCategory names the kind of peer event a PeerNews record gossips. The
// values are YaCy's fixed eight-character category codes.
type NewsCategory string

const (
	NewsProfileUpdate     NewsCategory = "prfleupd"
	NewsProfileBroadcast  NewsCategory = "prflecst"
	NewsProfileGoodVote   NewsCategory = "prflegvt"
	NewsProfileBadVote    NewsCategory = "prflebvt"
	NewsCrawlStart        NewsCategory = "crwlstrt"
	NewsCrawlStop         NewsCategory = "crwlstop"
	NewsCrawlComment      NewsCategory = "crwlcomm"
	NewsBlacklistAdd      NewsCategory = "blckladd"
	NewsBlacklistGoodVote NewsCategory = "blcklavt"
	NewsBlacklistDelete   NewsCategory = "blckldel"
	NewsBlacklistBadVote  NewsCategory = "blckldvt"
	NewsFileShareAdd      NewsCategory = "flshradd"
	NewsFileShareDelete   NewsCategory = "flshrdel"
	NewsFileShareComment  NewsCategory = "flshrcom"
	NewsBookmarkAdd       NewsCategory = "bkmrkadd"
	NewsBookmarkGoodVote  NewsCategory = "bkmrkavt"
	NewsBookmarkMove      NewsCategory = "bkmrkmov"
	NewsBookmarkMoveVote  NewsCategory = "bkmrkmvt"
	NewsBookmarkDelete    NewsCategory = "bkmrkdel"
	NewsBookmarkBadVote   NewsCategory = "bkmrkdvt"
	NewsSurfTipAdd        NewsCategory = "stippadd"
	NewsSurfTipGoodVote   NewsCategory = "stippavt"
	NewsWikiUpdate        NewsCategory = "wiki_upd"
	NewsWikiDelete        NewsCategory = "wiki_del"
	NewsBlogAdd           NewsCategory = "blog_add"
	NewsBlogDelete        NewsCategory = "blog_del"
	NewsTranslationAdd    NewsCategory = "transadd"
	NewsTranslationVote   NewsCategory = "transavt"
)

func ParseNewsCategory(s string) (NewsCategory, error) {
	switch NewsCategory(s) {
	case NewsProfileUpdate, NewsProfileBroadcast, NewsProfileGoodVote, NewsProfileBadVote,
		NewsCrawlStart, NewsCrawlStop, NewsCrawlComment,
		NewsBlacklistAdd, NewsBlacklistGoodVote, NewsBlacklistDelete, NewsBlacklistBadVote,
		NewsFileShareAdd, NewsFileShareDelete, NewsFileShareComment,
		NewsBookmarkAdd, NewsBookmarkGoodVote, NewsBookmarkMove, NewsBookmarkMoveVote,
		NewsBookmarkDelete, NewsBookmarkBadVote,
		NewsSurfTipAdd, NewsSurfTipGoodVote,
		NewsWikiUpdate, NewsWikiDelete, NewsBlogAdd, NewsBlogDelete,
		NewsTranslationAdd, NewsTranslationVote:
		return NewsCategory(s), nil
	default:
		return "", fmt.Errorf("%w: %q", ErrBadNewsCategory, s)
	}
}

func (c NewsCategory) String() string { return string(c) }
