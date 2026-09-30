package safetensorsmodel

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math"

	"github.com/nikitakarpei/yacy-rwi-node/spammodel"
	"github.com/nikitakarpei/yacy-rwi-node/spammodel/spamfeatures"
)

var errShortFile = errors.New("spam model file ends inside its header")

type RecipeVersionMismatchError struct {
	RecipeVersion int
}

func (m RecipeVersionMismatchError) Error() string {
	return fmt.Sprintf(
		"spam model of recipe version %d, this library reads %d",
		m.RecipeVersion,
		spamfeatures.RecipeVersion,
	)
}

func ModelFrom(file []byte) (spammodel.Model, error) {
	encodedHeader, tensorData, err := headerAndTensorDataOf(file)
	if err != nil {
		return spammodel.Model{}, err
	}
	modelHeader, placements, err := headerFrom(encodedHeader)
	if err != nil {
		return spammodel.Model{}, err
	}
	return modelOf(modelHeader.Metadata, &tensorReader{data: tensorData, placements: placements})
}

func headerAndTensorDataOf(file []byte) ([]byte, []byte, error) {
	if len(file) < headerLengthBytes {
		return nil, nil, errShortFile
	}
	headerLength := binary.LittleEndian.Uint64(file)
	//nolint:gosec // a slice length is never negative
	if headerLength > uint64(len(file)-headerLengthBytes) {
		return nil, nil, errShortFile
	}
	//nolint:gosec // the header length is at most the file length
	headerEnd := headerLengthBytes + int(headerLength)
	return file[headerLengthBytes:headerEnd], file[headerEnd:], nil
}

func headerFrom(encodedHeader []byte) (header, map[string]tensorPlacement, error) {
	var modelHeader header
	var placements map[string]tensorPlacement
	if err := errors.Join(
		json.Unmarshal(encodedHeader, &placements),
		json.Unmarshal(encodedHeader, &modelHeader),
	); err != nil {
		return header{}, nil, fmt.Errorf("spam model header: %w", err)
	}
	if modelHeader.Metadata.RecipeVersion != spamfeatures.RecipeVersion {
		return header{}, nil, RecipeVersionMismatchError{
			RecipeVersion: modelHeader.Metadata.RecipeVersion,
		}
	}
	return modelHeader, placements, nil
}

func modelOf(modelMetadata metadata, tensors *tensorReader) (spammodel.Model, error) {
	model := spammodel.Model{
		Version:   modelMetadata.Version,
		Threshold: modelMetadata.Threshold,
		Intercept: modelMetadata.Intercept,
		HashedWeights: make(
			map[spamfeatures.Family]spammodel.HashedWeights,
			len(spamfeatures.Families),
		),
		MetaWeights:    tensors.weightsOf(metaWeights),
		MetaMeans:      tensors.weightsOf(metaMeans),
		MetaDeviations: tensors.weightsOf(metaDeviations),
	}
	for _, family := range spamfeatures.Families {
		model.HashedWeights[family] = spammodel.HashedWeightsFrom(
			tensors.indicesOf(string(family)+indicesSuffix),
			tensors.weightsOf(string(family)+weightsSuffix),
		)
	}
	return model, tensors.err
}

type tensorReader struct {
	data       []byte
	placements map[string]tensorPlacement
	err        error
}

func (r *tensorReader) weightsOf(name string) []float64 {
	data := r.dataOf(name, weightType, weightBytes)
	weights := make([]float64, len(data)/weightBytes)
	for position := range weights {
		weights[position] = math.Float64frombits(
			binary.LittleEndian.Uint64(data[position*weightBytes:]),
		)
	}
	return weights
}

func (r *tensorReader) indicesOf(name string) []int32 {
	data := r.dataOf(name, indexType, indexBytes)
	indices := make([]int32, len(data)/indexBytes)
	for position := range indices {
		indexBits := binary.LittleEndian.Uint32(data[position*indexBytes:])
		indices[position] = int32(indexBits) //nolint:gosec // I32 bits
	}
	return indices
}

func (r *tensorReader) dataOf(name, tensorType string, valueBytes int) []byte {
	placement, found := r.placements[name]
	begin, end := placement.DataOffsets[0], placement.DataOffsets[1]
	if !found || placement.Type != tensorType || begin < 0 || begin > end || end > len(r.data) ||
		(end-begin)%valueBytes != 0 {
		r.err = errors.Join(
			r.err,
			fmt.Errorf("spam model tensor %s is not a readable %s tensor", name, tensorType),
		)
		return nil
	}
	return r.data[begin:end]
}
