package safetensorsmodel_test

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/spammodel"
	"github.com/nikitakarpei/yacy-rwi-node/spammodel/safetensorsmodel"
	"github.com/nikitakarpei/yacy-rwi-node/spammodel/spamfeatures"
)

func trainedModel() spammodel.Model {
	hashedWeights := make(
		map[spamfeatures.Family]spammodel.HashedWeights,
		len(spamfeatures.Families),
	)
	for position, family := range spamfeatures.Families {
		hashedWeights[family] = spammodel.HashedWeightsFrom(
			[]int32{int32(position) + 1000, int32(position)},
			[]float64{-0.25 * float64(position), 1.5},
		)
	}
	return spammodel.Model{
		Version:        "2026-09-29-recipe-5",
		Threshold:      0.62,
		Intercept:      -1.125,
		HashedWeights:  hashedWeights,
		MetaWeights:    []float64{0.5, -2},
		MetaMeans:      []float64{3, 4.25},
		MetaDeviations: []float64{1, 0.75},
	}
}

func TestAWrittenModelReadsBackAsTheSameModel(t *testing.T) {
	model, err := safetensorsmodel.ModelFrom(safetensorsmodel.SafetensorsFrom(trainedModel()))
	if err != nil {
		t.Fatalf("ModelFrom: %v", err)
	}

	if !reflect.DeepEqual(model, trainedModel()) {
		t.Errorf("ModelFrom = %+v, want %+v", model, trainedModel())
	}
}

func TestTheHeaderIsAlignedToEightBytes(t *testing.T) {
	file := safetensorsmodel.SafetensorsFrom(trainedModel())

	if headerLength := int(file[0]) | int(file[1])<<8; headerLength%8 != 0 {
		t.Errorf("header length = %d, want a multiple of 8", headerLength)
	}
}

func TestAModelOfAnotherRecipeIsRefused(t *testing.T) {
	otherRecipeVersion := spamfeatures.RecipeVersion + 1
	file := bytes.Replace(safetensorsmodel.SafetensorsFrom(trainedModel()),
		fmt.Appendf(nil, `"recipeVersion":"%d"`, spamfeatures.RecipeVersion),
		fmt.Appendf(nil, `"recipeVersion":"%d"`, otherRecipeVersion), 1)

	_, err := safetensorsmodel.ModelFrom(file)

	var mismatch safetensorsmodel.RecipeVersionMismatchError
	if !errors.As(err, &mismatch) || mismatch.RecipeVersion != otherRecipeVersion ||
		!strings.Contains(err.Error(), strconv.Itoa(otherRecipeVersion)) {
		t.Errorf(
			"ModelFrom error = %v, want a mismatch of recipe version %d",
			err,
			otherRecipeVersion,
		)
	}
}

func TestADamagedFileIsRefused(t *testing.T) {
	file := safetensorsmodel.SafetensorsFrom(trainedModel())
	damagedFiles := map[string][]byte{
		"shorter than a header length": file[:4],
		"cut inside the header":        file[:40],
		"cut inside the tensors":       file[:len(file)-4],
		"not JSON in the header": append(
			append([]byte{}, file[:8]...),
			bytes.Replace(file[8:], []byte("{"), []byte("["), 1)...),
		"a threshold that is no number": bytes.Replace(
			file,
			[]byte(`"threshold":"0.62"`),
			[]byte(`"threshold":"x.62"`),
			1,
		),
		"a tensor of another type": bytes.Replace(
			file,
			[]byte(`"dtype":"F64"`),
			[]byte(`"dtype":"F32"`),
			1,
		),
	}
	for damage, damagedFile := range damagedFiles {
		if _, err := safetensorsmodel.ModelFrom(damagedFile); err == nil {
			t.Errorf("ModelFrom of a file %s: no error", damage)
		}
	}
}
