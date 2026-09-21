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

package maxscore

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	k8stypes "k8s.io/apimachinery/pkg/types"

	fwkdl "github.com/llm-d/llm-d-router/pkg/epp/framework/interface/datalayer"
	fwkplugin "github.com/llm-d/llm-d-router/pkg/epp/framework/interface/plugin"
	fwksched "github.com/llm-d/llm-d-router/pkg/epp/framework/interface/scheduling"
)

func TestPickMaxScorePicker(t *testing.T) {
	endpoint1 := fwksched.NewEndpoint(&fwkdl.EndpointMetadata{ID: k8stypes.NamespacedName{Name: "pod1"}}, nil, nil)
	endpoint2 := fwksched.NewEndpoint(&fwkdl.EndpointMetadata{ID: k8stypes.NamespacedName{Name: "pod2"}}, nil, nil)
	endpoint3 := fwksched.NewEndpoint(&fwkdl.EndpointMetadata{ID: k8stypes.NamespacedName{Name: "pod3"}}, nil, nil)

	tests := []struct {
		name               string
		picker             fwksched.Picker
		input              []*fwksched.ScoredEndpoint
		output             []fwksched.Endpoint
		tieBreakCandidates int // tie break is random, specify how many candidate with max score
	}{
		{
			name:   "Single max score",
			picker: NewMaxScorePicker(1),
			input: []*fwksched.ScoredEndpoint{
				{Endpoint: endpoint1, Score: 10},
				{Endpoint: endpoint2, Score: 25},
				{Endpoint: endpoint3, Score: 15},
			},
			output: []fwksched.Endpoint{
				&fwksched.ScoredEndpoint{Endpoint: endpoint2, Score: 25},
			},
		},
		{
			name:   "Multiple max scores, all are equally scored",
			picker: NewMaxScorePicker(2),
			input: []*fwksched.ScoredEndpoint{
				{Endpoint: endpoint1, Score: 50},
				{Endpoint: endpoint2, Score: 50},
				{Endpoint: endpoint3, Score: 30},
			},
			output: []fwksched.Endpoint{
				&fwksched.ScoredEndpoint{Endpoint: endpoint1, Score: 50},
				&fwksched.ScoredEndpoint{Endpoint: endpoint2, Score: 50},
			},
			tieBreakCandidates: 2,
		},
		{
			name:   "Multiple results sorted by highest score, more pods than needed",
			picker: NewMaxScorePicker(2),
			input: []*fwksched.ScoredEndpoint{
				{Endpoint: endpoint1, Score: 20},
				{Endpoint: endpoint2, Score: 25},
				{Endpoint: endpoint3, Score: 30},
			},
			output: []fwksched.Endpoint{
				&fwksched.ScoredEndpoint{Endpoint: endpoint3, Score: 30},
				&fwksched.ScoredEndpoint{Endpoint: endpoint2, Score: 25},
			},
		},
		{
			name:   "Multiple results sorted by highest score, less pods than needed",
			picker: NewMaxScorePicker(4), // picker is required to return 4 pods at most, but we have only 3.
			input: []*fwksched.ScoredEndpoint{
				{Endpoint: endpoint1, Score: 20},
				{Endpoint: endpoint2, Score: 25},
				{Endpoint: endpoint3, Score: 30},
			},
			output: []fwksched.Endpoint{
				&fwksched.ScoredEndpoint{Endpoint: endpoint3, Score: 30},
				&fwksched.ScoredEndpoint{Endpoint: endpoint2, Score: 25},
				&fwksched.ScoredEndpoint{Endpoint: endpoint1, Score: 20},
			},
		},
		{
			name:   "Multiple results sorted by highest score, num of pods exactly needed",
			picker: NewMaxScorePicker(3), // picker is required to return 3 pods at most, we have only 3.
			input: []*fwksched.ScoredEndpoint{
				{Endpoint: endpoint1, Score: 30},
				{Endpoint: endpoint2, Score: 25},
				{Endpoint: endpoint3, Score: 30},
			},
			output: []fwksched.Endpoint{
				&fwksched.ScoredEndpoint{Endpoint: endpoint1, Score: 30},
				&fwksched.ScoredEndpoint{Endpoint: endpoint3, Score: 30},
				&fwksched.ScoredEndpoint{Endpoint: endpoint2, Score: 25},
			},
			tieBreakCandidates: 2,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := test.picker.Pick(context.Background(), test.input)
			got := result.TargetEndpoints

			if test.tieBreakCandidates > 0 {
				testMaxScoredEndpoints := test.output[:test.tieBreakCandidates]
				gotMaxScoredEndpoints := got[:test.tieBreakCandidates]
				diff := cmp.Diff(testMaxScoredEndpoints, gotMaxScoredEndpoints, cmpopts.SortSlices(func(a, b fwksched.Endpoint) bool {
					return a.String() < b.String() // predictable order within the endpoints with equal scores
				}), cmp.Comparer(fwksched.ScoredEndpointComparer))
				if diff != "" {
					t.Errorf("Unexpected output (-want +got): %v", diff)
				}
				test.output = test.output[test.tieBreakCandidates:]
				got = got[test.tieBreakCandidates:]
			}

			if diff := cmp.Diff(test.output, got, cmp.Comparer(fwksched.ScoredEndpointComparer)); diff != "" {
				t.Errorf("Unexpected output (-want +got): %v", diff)
			}
		})
	}
}

func TestPickRandomlyWithinTopScoreRatio(t *testing.T) {
	endpoint1 := fwksched.NewEndpoint(&fwkdl.EndpointMetadata{ID: k8stypes.NamespacedName{Name: "pod1"}}, nil, nil)
	endpoint2 := fwksched.NewEndpoint(&fwkdl.EndpointMetadata{ID: k8stypes.NamespacedName{Name: "pod2"}}, nil, nil)
	endpoint3 := fwksched.NewEndpoint(&fwkdl.EndpointMetadata{ID: k8stypes.NamespacedName{Name: "pod3"}}, nil, nil)
	endpoint4 := fwksched.NewEndpoint(&fwkdl.EndpointMetadata{ID: k8stypes.NamespacedName{Name: "pod4"}}, nil, nil)

	tests := []struct {
		name           string
		topScoreRatio  float64
		scores         []float64
		allowed        map[string]bool
		wantAllSeen    bool
		iterationCount int
	}{
		{
			name:           "ratio one preserves highest score selection",
			topScoreRatio:  1,
			scores:         []float64{40, 30, 20, 10},
			allowed:        map[string]bool{"pod1": true},
			wantAllSeen:    true,
			iterationCount: 20,
		},
		{
			name:           "samples endpoints within five percent of maximum",
			topScoreRatio:  0.95,
			scores:         []float64{100, 96, 94, 20},
			allowed:        map[string]bool{"pod1": true, "pod2": true},
			wantAllSeen:    true,
			iterationCount: 500,
		},
		{
			name:           "includes endpoint exactly at ratio boundary",
			topScoreRatio:  0.95,
			scores:         []float64{100, 95, 94, 20},
			allowed:        map[string]bool{"pod1": true, "pod2": true},
			wantAllSeen:    true,
			iterationCount: 500,
		},
		{
			name:           "all zero scores sample all candidates",
			topScoreRatio:  0.95,
			scores:         []float64{0, 0, 0, 0},
			allowed:        map[string]bool{"pod1": true, "pod2": true, "pod3": true, "pod4": true},
			wantAllSeen:    true,
			iterationCount: 500,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			p := NewMaxScorePicker(1).WithTopScoreRatio(test.topScoreRatio)
			seen := make(map[string]bool)
			for range test.iterationCount {
				input := []*fwksched.ScoredEndpoint{
					{Endpoint: endpoint1, Score: test.scores[0]},
					{Endpoint: endpoint2, Score: test.scores[1]},
					{Endpoint: endpoint3, Score: test.scores[2]},
					{Endpoint: endpoint4, Score: test.scores[3]},
				}
				result := p.Pick(context.Background(), input)
				if len(result.TargetEndpoints) != 1 {
					t.Fatalf("expected one endpoint, got %d", len(result.TargetEndpoints))
				}
				name := result.TargetEndpoints[0].GetMetadata().ID.Name
				if !test.allowed[name] {
					t.Fatalf("selected endpoint %q outside top score ratio %v", name, test.topScoreRatio)
				}
				seen[name] = true
			}

			if test.wantAllSeen && len(seen) != len(test.allowed) {
				t.Fatalf("expected to observe every allowed endpoint, saw %v", seen)
			}
		})
	}
}

func TestMaxScorePickerFactoryTopScoreRatio(t *testing.T) {
	tests := []struct {
		name              string
		config            string
		wantTopScoreRatio float64
		wantError         bool
	}{
		{name: "explicit ratio", config: `{"maxNumOfEndpoints":1,"topScoreRatio":0.95}`, wantTopScoreRatio: 0.95},
		{name: "omitted ratio defaults to one", config: `{"maxNumOfEndpoints":1}`, wantTopScoreRatio: 1},
		{name: "zero ratio is invalid", config: `{"maxNumOfEndpoints":1,"topScoreRatio":0}`, wantError: true},
		{name: "ratio above one is invalid", config: `{"maxNumOfEndpoints":1,"topScoreRatio":1.01}`, wantError: true},
		{name: "legacy top k is rejected", config: `{"maxNumOfEndpoints":1,"topK":3}`, wantError: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			decoder := fwkplugin.StrictDecoder(json.RawMessage(test.config))
			plugin, err := MaxScorePickerFactory("decode-top-tier-picker", decoder, nil)
			if test.wantError {
				if err == nil {
					t.Fatal("expected factory error")
				}
				return
			}
			if err != nil {
				t.Fatalf("factory returned error: %v", err)
			}

			p, ok := plugin.(*MaxScorePicker)
			if !ok {
				t.Fatalf("expected *MaxScorePicker, got %T", plugin)
			}
			if p.topScoreRatio != test.wantTopScoreRatio {
				t.Fatalf("expected topScoreRatio %v, got %v", test.wantTopScoreRatio, p.topScoreRatio)
			}
			if p.TypedName().Name != "decode-top-tier-picker" {
				t.Fatalf("expected configured name, got %q", p.TypedName().Name)
			}
		})
	}
}
