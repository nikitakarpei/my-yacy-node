package judgedqueries_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
)

func TestZZText(t *testing.T) {
	in := os.Getenv("TEXT_IN")
	if in == "" {
		t.Skip()
	}
	var need map[string]map[string]string
	b, _ := os.ReadFile(in)
	_ = json.Unmarshal(b, &need)
	ex := pageExtractionOfEveryFormat(t)
	type rec struct {
		Query, Hash, Address, Title, Text string
		Stored                            bool
	}
	seen := map[string]bool{}
	var fds []queryanswers.FoundDocument
	for _, docs := range need {
		for _, addr := range docs {
			if !seen[addr] {
				seen[addr] = true
				fds = append(fds, queryanswers.FoundDocument{Address: addr})
			}
		}
	}
	pages := pagePerAddressOf(pageFetchingWithin(240*time.Second).fetchedPagesOf(context.Background(), fds))
	var out []rec
	for query, docs := range need {
		for hash, addr := range docs {
			r := rec{Query: query, Hash: hash, Address: addr}
			if p, ok := pages[addr]; ok {
				e := ex.extractedPageOf(context.Background(), p)
				r.Stored = e.text != ""
				r.Title = e.title
				txt := []rune(e.text)
				if len(txt) > 2500 {
					txt = txt[:2500]
				}
				r.Text = string(txt)
			}
			out = append(out, r)
		}
	}
	ob, _ := json.Marshal(out)
	_ = os.WriteFile(os.Getenv("TEXT_OUT"), ob, 0o644)
}
