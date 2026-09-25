package judgedqueries_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/documentrelevance"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type dumpDoc struct {
	Hash      string  `json:"hash"`
	Address   string  `json:"address"`
	Site      string  `json:"site"`
	Title     string  `json:"title"`
	Relevance float64 `json:"relevance"`
	Grade     *int    `json:"grade"`
	Spam      bool    `json:"spam"`
	Holders   int     `json:"holders"`
	Words     []string `json:"words"`
	ServiceRank int   `json:"serviceRank"`
}

type dumpQuery struct {
	Query string    `json:"query"`
	Docs  []dumpDoc `json:"docs"`
}

func TestZZDump(t *testing.T) {
	out := os.Getenv("DUMP_OUT")
	if out == "" {
		t.Skip()
	}
	scorer := documentrelevance.RelevanceScorerWeighedBy(documentrelevance.DefaultRelevanceWeights())
	var all []dumpQuery
	for _, jf := range queryJudgmentsFiles(t) {
		j := queryJudgmentsAt(t, jf)
		g := j.gradedDocuments()
		rec := recordedAnswersAt(t, recordedAnswersFileOf(j.Query))
		ans := rec.answers()
		rel := scorer.RelevancePerDocumentOf(ans)
		order := defaultServiceOrdering().OrderedDocumentsOf(ans)
		rank := map[yacymodel.URLHash]int{}
		for i, d := range order {
			rank[d.Hash] = i
		}
		q := dumpQuery{Query: j.Query}
		for _, d := range ans.FoundDocuments {
			dd := dumpDoc{Hash: d.Hash.String(), Address: d.Address, Site: yacymodel.SiteOf(d.Address), Title: d.Title, Relevance: rel[d.Hash], ServiceRank: rank[d.Hash]}
			if gd, ok := g[d.Hash]; ok {
				gr := gd.grade
				dd.Grade = &gr
				dd.Spam = gd.spam
			}
			holders := map[string]bool{}
			words := map[string]bool{}
			for _, p := range d.PostingReplicas {
				holders[p.Holder.String()] = true
				words[p.Word.String()] = true
			}
			dd.Holders = len(holders)
			for w := range words {
				dd.Words = append(dd.Words, w)
			}
			q.Docs = append(q.Docs, dd)
		}
		all = append(all, q)
	}
	b, _ := json.Marshal(all)
	if err := os.WriteFile(out, b, 0o644); err != nil {
		t.Fatal(err)
	}
}
