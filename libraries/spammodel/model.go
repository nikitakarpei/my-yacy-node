// Package spammodel scores the feature row of a page with a trained logistic
// model.
package spammodel

import (
	"math"

	"github.com/nikitakarpei/yacy-rwi-node/spammodel/spamfeatures"
)

type Model struct {
	Version        string
	Threshold      float64
	Intercept      float64
	HashedWeights  map[spamfeatures.Family]HashedWeights
	MetaWeights    []float64
	MetaMeans      []float64
	MetaDeviations []float64
}

func (m Model) ScoreOf(row spamfeatures.Row) float64 {
	return logisticOf(m.Intercept + m.hashedMarginOf(row) + m.metaMarginOf(row))
}

func logisticOf(margin float64) float64 {
	return 1 / (1 + math.Exp(-margin))
}

func (m Model) hashedMarginOf(row spamfeatures.Row) float64 {
	margin := 0.0
	for _, family := range spamfeatures.Families {
		for _, entry := range row.HashedEntries[family] {
			margin += m.HashedWeights[family].WeightOf(entry.Index) * entry.Value
		}
	}
	return margin
}

func (m Model) metaMarginOf(row spamfeatures.Row) float64 {
	margin := 0.0
	for position, weight := range m.MetaWeights {
		margin += weight * (row.MetaValues[position] - m.MetaMeans[position]) / m.MetaDeviations[position]
	}
	return margin
}
