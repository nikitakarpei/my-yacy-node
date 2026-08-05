package searchendpoint

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/matchreport"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

func requestedReportFromRequest(req yacyproto.SearchRequest) matchreport.RequestedReport {
	switch req.Abstracts {
	case "":
		return matchreport.RequestedReport{Mode: matchreport.NoMatches}
	case yacyproto.SearchAbstractsAuto:
		return matchreport.RequestedReport{Mode: matchreport.TermWithMostMatches}
	default:
		return matchreport.RequestedReport{
			Mode:  matchreport.RequestedTerms,
			Terms: req.Abstracts.Hashes(),
		}
	}
}
