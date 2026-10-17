package pagefingerprint

import (
	"errors"

	"github.com/nikitakarpei/yacy-rwi-node/storedfields"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

var errBadFingerprint = errors.New("bad page fingerprint")

func AppendBytes(fingerprint Fingerprint, buffer []byte) []byte {
	var stored storedfields.Writer
	Write(&stored, fingerprint)

	return append(buffer, stored.Record()...)
}

func Write(stored *storedfields.Writer, fingerprint Fingerprint) {
	stored.Varint(int64(fingerprint.Modified))
	stored.Count(fingerprint.LocalLinks)
	stored.Count(fingerprint.ExternalLinks)
	stored.Count(fingerprint.TitleWords)
	stored.Text(fingerprint.Language.String())
	stored.Byte(byte(fingerprint.DocumentType))
}

func ParseBytes(raw []byte) (Fingerprint, error) {
	stored := storedfields.ReaderOf(raw, errBadFingerprint)
	fingerprint := From(stored)
	if err := stored.Err(); err != nil {
		return Fingerprint{}, err
	}

	return fingerprint, nil
}

func From(stored *storedfields.Reader) Fingerprint {
	return Fingerprint{
		Modified:      yacymodel.MicroDate(stored.Varint("modified")),
		LocalLinks:    stored.Count("local links"),
		ExternalLinks: stored.Count("external links"),
		TitleWords:    stored.Count("title words"),
		Language:      languageFrom(stored),
		DocumentType:  yacymodel.DocumentType(stored.Byte("document type")),
	}
}

func languageFrom(stored *storedfields.Reader) yacymodel.Language {
	var code [yacymodel.LanguageCodeLength]byte
	language, err := yacymodel.ParseLanguageBytes(stored.AppendBytes(code[:0], "language"))
	if err != nil {
		stored.Reject("language", err)

		return yacymodel.Language{}
	}

	return language
}
