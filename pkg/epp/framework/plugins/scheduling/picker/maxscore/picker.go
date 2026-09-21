/*
Copyright 2025 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package maxscore implements a scheduling picker that selects the endpoint(s) with the highest
// score calculated during the scoring phase.
//
// For detailed behavioral intent and configuration, see the package README.
package maxscore

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"

	"sigs.k8s.io/controller-runtime/pkg/log"

	logutil "github.com/llm-d/llm-d-router/pkg/common/observability/logging"
	fwkplugin "github.com/llm-d/llm-d-router/pkg/epp/framework/interface/plugin"
	fwksched "github.com/llm-d/llm-d-router/pkg/epp/framework/interface/scheduling"
	"github.com/llm-d/llm-d-router/pkg/epp/framework/plugins/scheduling/picker"
)

const (
	// MaxScorePickerType is the registered name of the max score picker plugin.
	MaxScorePickerType = "max-score-picker"
)

// compile-time type validation
var _ fwksched.Picker = &MaxScorePicker{}

// MaxScorePickerFactory defines the factory function for MaxScorePicker.
func MaxScorePickerFactory(name string, rawParameters *json.Decoder, _ fwkplugin.Handle) (fwkplugin.Plugin, error) {
	parameters := picker.PickerParameters{
		MaxNumOfEndpoints: picker.DefaultMaxNumOfEndpoints,
		TopScoreRatio:     picker.DefaultTopScoreRatio,
	}
	if rawParameters != nil {
		if err := rawParameters.Decode(&parameters); err != nil {
			return nil, fmt.Errorf("failed to parse the parameters of the '%s' picker - %w", MaxScorePickerType, err)
		}
	}
	if parameters.TopScoreRatio <= 0 || parameters.TopScoreRatio > 1 {
		return nil, fmt.Errorf("topScoreRatio must be within (0, 1], got %v", parameters.TopScoreRatio)
	}

	return NewMaxScorePicker(parameters.MaxNumOfEndpoints).WithTopScoreRatio(parameters.TopScoreRatio).WithName(name), nil
}

// NewMaxScorePicker initializes a new MaxScorePicker and returns its pointer.
func NewMaxScorePicker(maxNumOfEndpoints int) *MaxScorePicker {
	if maxNumOfEndpoints <= 0 {
		maxNumOfEndpoints = picker.DefaultMaxNumOfEndpoints // on invalid configuration value, fallback to default value
	}

	return &MaxScorePicker{
		typedName:         fwkplugin.TypedName{Type: MaxScorePickerType, Name: MaxScorePickerType},
		maxNumOfEndpoints: maxNumOfEndpoints,
		topScoreRatio:     picker.DefaultTopScoreRatio,
	}
}

// MaxScorePicker picks endpoint(s) with the highest score calculated during the scoring phase.
type MaxScorePicker struct {
	typedName         fwkplugin.TypedName
	maxNumOfEndpoints int     // maximum number of endpoints to pick
	topScoreRatio     float64 // minimum fraction of the highest score eligible for randomized selection
}

// WithName sets the picker's name
func (p *MaxScorePicker) WithName(name string) *MaxScorePicker {
	p.typedName.Name = name
	return p
}

// WithTopScoreRatio sets the minimum fraction of the highest score eligible for randomized selection.
func (p *MaxScorePicker) WithTopScoreRatio(topScoreRatio float64) *MaxScorePicker {
	if topScoreRatio <= 0 || topScoreRatio > 1 {
		topScoreRatio = picker.DefaultTopScoreRatio
	}
	p.topScoreRatio = topScoreRatio
	return p
}

// TypedName returns the type and name tuple of this plugin instance.
func (p *MaxScorePicker) TypedName() fwkplugin.TypedName {
	return p.typedName
}

// Pick selects the endpoint(s) with the highest score calculated during the scoring phase.
func (p *MaxScorePicker) Pick(ctx context.Context, scoredEndpoints []*fwksched.ScoredEndpoint) *fwksched.ProfileRunResult {
	log.FromContext(ctx).V(logutil.DEBUG).Info("Selecting endpoints from candidates sorted by max score", "max-num-of-endpoints", p.maxNumOfEndpoints,
		"num-of-candidates", len(scoredEndpoints), "scored-endpoints", scoredEndpoints)

	// Shuffle in-place - needed for random tie break when scores are equal
	picker.ShuffleScoredEndpoints(scoredEndpoints)

	slices.SortStableFunc(scoredEndpoints, func(i, j *fwksched.ScoredEndpoint) int { // highest score first
		if i.Score > j.Score {
			return -1
		}
		if i.Score < j.Score {
			return 1
		}
		return 0
	})

	// Randomize endpoints whose score is within the configured ratio of the maximum. Keep enough
	// candidates to preserve the legacy multi-endpoint behavior when more endpoints are requested.
	topTierSize := 0
	if len(scoredEndpoints) > 0 {
		minimumTopTierScore := scoredEndpoints[0].Score * p.topScoreRatio
		for topTierSize < len(scoredEndpoints) && scoredEndpoints[topTierSize].Score >= minimumTopTierScore {
			topTierSize++
		}
	}
	topTierSize = max(topTierSize, min(p.maxNumOfEndpoints, len(scoredEndpoints)))
	if topTierSize > p.maxNumOfEndpoints {
		picker.ShuffleScoredEndpoints(scoredEndpoints[:topTierSize])
	}

	// if we have enough endpoints to return keep only the "maxNumOfEndpoints" highest scored endpoints
	if p.maxNumOfEndpoints < len(scoredEndpoints) {
		scoredEndpoints = scoredEndpoints[:p.maxNumOfEndpoints]
	}

	targetEndpoints := make([]fwksched.Endpoint, len(scoredEndpoints))
	for i, scoredEndpoint := range scoredEndpoints {
		targetEndpoints[i] = scoredEndpoint
	}

	return &fwksched.ProfileRunResult{TargetEndpoints: targetEndpoints}
}
