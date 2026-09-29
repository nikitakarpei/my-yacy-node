package safetensorsmodel

import (
	"encoding/binary"
	"encoding/json"
	"math"
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/spammodel"
	"github.com/nikitakarpei/yacy-rwi-node/spammodel/spamfeatures"
)

type tensor struct {
	name        string
	tensorType  string
	valueAmount int
	data        []byte
}

func SafetensorsFrom(model spammodel.Model) []byte {
	tensors := tensorsOf(model)
	encodedHeader := headerOf(model, tensors)
	file := binary.LittleEndian.AppendUint64(nil, uint64(len(encodedHeader)))
	file = append(file, encodedHeader...)
	for _, modelTensor := range tensors {
		file = append(file, modelTensor.data...)
	}
	return file
}

func tensorsOf(model spammodel.Model) []tensor {
	weightTensors := []tensor{
		weightTensorOf(metaWeights, model.MetaWeights),
		weightTensorOf(metaMeans, model.MetaMeans),
		weightTensorOf(metaDeviations, model.MetaDeviations),
	}
	indexTensors := make([]tensor, 0, len(spamfeatures.Families))
	for _, family := range spamfeatures.Families {
		var indices []int32
		var weights []float64
		for index, weight := range model.HashedWeights[family].Entries() {
			indices = append(indices, index)
			weights = append(weights, weight)
		}
		weightTensors = append(weightTensors, weightTensorOf(string(family)+weightsSuffix, weights))
		indexTensors = append(indexTensors, indexTensorOf(string(family)+indicesSuffix, indices))
	}
	return append(weightTensors, indexTensors...)
}

func weightTensorOf(name string, weights []float64) tensor {
	data := make([]byte, 0, len(weights)*weightBytes)
	for _, weight := range weights {
		data = binary.LittleEndian.AppendUint64(data, math.Float64bits(weight))
	}
	return tensor{name: name, tensorType: weightType, valueAmount: len(weights), data: data}
}

func indexTensorOf(name string, indices []int32) tensor {
	data := make([]byte, 0, len(indices)*indexBytes)
	for _, index := range indices {
		indexBits := uint32(index) //nolint:gosec // I32 bits
		data = binary.LittleEndian.AppendUint32(data, indexBits)
	}
	return tensor{name: name, tensorType: indexType, valueAmount: len(indices), data: data}
}

func headerOf(model spammodel.Model, tensors []tensor) []byte {
	fields := map[string]any{
		"__metadata__": metadata{
			RecipeVersion: spamfeatures.RecipeVersion,
			Version:       model.Version,
			Threshold:     model.Threshold,
			Intercept:     model.Intercept,
		},
	}
	offset := 0
	for _, modelTensor := range tensors {
		fields[modelTensor.name] = tensorPlacement{
			Type:        modelTensor.tensorType,
			Shape:       []int{modelTensor.valueAmount},
			DataOffsets: [2]int{offset, offset + len(modelTensor.data)},
		}
		offset += len(modelTensor.data)
	}
	encodedHeader, _ := json.Marshal(fields) //nolint:errchkjson // plain fields
	padding := strings.Repeat(
		" ",
		(headerAlignment-len(encodedHeader)%headerAlignment)%headerAlignment,
	)
	return append(encodedHeader, padding...)
}
