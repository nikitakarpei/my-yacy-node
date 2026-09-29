// Package safetensorsmodel writes a spam model as a safetensors file and reads
// it back.
package safetensorsmodel

const (
	headerLengthBytes = 8
	headerAlignment   = 8
	weightBytes       = 8
	indexBytes        = 4
	weightType        = "F64"
	indexType         = "I32"
	metaWeights       = "meta.weights"
	metaMeans         = "meta.means"
	metaDeviations    = "meta.deviations"
	weightsSuffix     = ".weights"
	indicesSuffix     = ".indices"
)

type header struct {
	Metadata metadata `json:"__metadata__"`
}

type metadata struct {
	RecipeVersion int     `json:"recipeVersion,string"`
	Version       string  `json:"version"`
	Threshold     float64 `json:"threshold,string"`
	Intercept     float64 `json:"intercept,string"`
}

type tensorPlacement struct {
	Type        string `json:"dtype"`
	Shape       []int  `json:"shape"`
	DataOffsets [2]int `json:"data_offsets"`
}
