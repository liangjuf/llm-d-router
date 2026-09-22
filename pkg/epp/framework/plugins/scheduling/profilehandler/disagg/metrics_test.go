/*
Copyright 2026 The Kubernetes Authors.

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

package disagg

import (
	"context"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/llm-d/llm-d-router/pkg/epp/framework/interface/scheduling"
)

func TestSchedulerPDDecisionCount(t *testing.T) {
	SchedulerPDDecisionCount.Reset()
	LlmdPDDecisionCount.Reset()

	model := "test-model"

	RecordPDDecision("test-plugin", "test-type", model, DecisionTypePrefillDecode)
	RecordPDDecision("test-plugin", "test-type", model, DecisionTypeDecodeOnly)
	RecordPDDecision("test-plugin", "test-type", model, DecisionTypePrefillDecode)

	expected := `
		# HELP llm_d_inference_scheduler_pd_decision_total [ALPHA] [Deprecated: Use llm_d_epp_pd_decision_total] Total number of P/D disaggregation decisions made
		# TYPE llm_d_inference_scheduler_pd_decision_total counter
		llm_d_inference_scheduler_pd_decision_total{decision_type="decode-only",model_name="test-model"} 1
		llm_d_inference_scheduler_pd_decision_total{decision_type="prefill-decode",model_name="test-model"} 2
	`

	if err := testutil.CollectAndCompare(SchedulerPDDecisionCount, strings.NewReader(expected),
		"llm_d_inference_scheduler_pd_decision_total"); err != nil {
		t.Errorf("RecordPDDecision() failed: %v", err)
	}

	expectedNew := `
		# HELP llm_d_epp_pd_decision_total [ALPHA] Total number of P/D disaggregation decisions made
		# TYPE llm_d_epp_pd_decision_total counter
		llm_d_epp_pd_decision_total{decision_type="decode-only",model_name="test-model",plugin_name="test-plugin",plugin_type="test-type"} 1
		llm_d_epp_pd_decision_total{decision_type="prefill-decode",model_name="test-model",plugin_name="test-plugin",plugin_type="test-type"} 2
	`

	if err := testutil.CollectAndCompare(LlmdPDDecisionCount, strings.NewReader(expectedNew),
		"llm_d_epp_pd_decision_total"); err != nil {
		t.Errorf("RecordPDDecision() new failed: %v", err)
	}
}

func TestRecordDisaggDecision(t *testing.T) {
	// Reset the counters before the test to avoid interference from other tests.
	SchedulerDisaggDecisionCount.Reset()
	LlmdDisaggDecisionCount.Reset()

	model := "test-model"
	RecordDisaggDecision("test-plugin", "test-type", model, DecisionTypeDecodeOnly)
	RecordDisaggDecision("test-plugin", "test-type", model, DecisionTypePrefillDecode)
	RecordDisaggDecision("test-plugin", "test-type", model, DecisionTypePrefillDecode)
	RecordDisaggDecision("test-plugin", "test-type", model, DecisionTypeEncodeDecode)
	RecordDisaggDecision("test-plugin", "test-type", model, DecisionTypeEncodePrefillDecode)
	RecordDisaggDecision("test-plugin", "test-type", model, DecisionTypeEncodePrefillDecode)
	RecordDisaggDecision("test-plugin", "test-type", model, DecisionTypeEncodePrefillDecode)

	expected := `
		# HELP llm_d_inference_scheduler_disagg_decision_total [ALPHA] [Deprecated: Use llm_d_epp_disagg_decision_total] Total number of disaggregation routing decisions made
		# TYPE llm_d_inference_scheduler_disagg_decision_total counter
		llm_d_inference_scheduler_disagg_decision_total{decision_type="decode-only",model_name="test-model"} 1
		llm_d_inference_scheduler_disagg_decision_total{decision_type="encode-decode",model_name="test-model"} 1
		llm_d_inference_scheduler_disagg_decision_total{decision_type="encode-prefill-decode",model_name="test-model"} 3
		llm_d_inference_scheduler_disagg_decision_total{decision_type="prefill-decode",model_name="test-model"} 2
	`

	if err := testutil.CollectAndCompare(SchedulerDisaggDecisionCount, strings.NewReader(expected),
		"llm_d_inference_scheduler_disagg_decision_total"); err != nil {
		t.Errorf("RecordDisaggDecision() failed: %v", err)
	}

	expectedNew := `
		# HELP llm_d_epp_disagg_decision_total [ALPHA] Total number of disaggregation routing decisions made
		# TYPE llm_d_epp_disagg_decision_total counter
		llm_d_epp_disagg_decision_total{decision_type="decode-only",model_name="test-model",plugin_name="test-plugin",plugin_type="test-type"} 1
		llm_d_epp_disagg_decision_total{decision_type="encode-decode",model_name="test-model",plugin_name="test-plugin",plugin_type="test-type"} 1
		llm_d_epp_disagg_decision_total{decision_type="encode-prefill-decode",model_name="test-model",plugin_name="test-plugin",plugin_type="test-type"} 3
		llm_d_epp_disagg_decision_total{decision_type="prefill-decode",model_name="test-model",plugin_name="test-plugin",plugin_type="test-type"} 2
	`

	if err := testutil.CollectAndCompare(LlmdDisaggDecisionCount, strings.NewReader(expectedNew),
		"llm_d_epp_disagg_decision_total"); err != nil {
		t.Errorf("RecordDisaggDecision() new failed: %v", err)
	}
}

func TestRecordDisaggDecisionEmptyModel(t *testing.T) {
	SchedulerDisaggDecisionCount.Reset()
	LlmdDisaggDecisionCount.Reset()

	RecordDisaggDecision("test-plugin", "test-type", "", DecisionTypeDecodeOnly)

	expected := `
		# HELP llm_d_inference_scheduler_disagg_decision_total [ALPHA] [Deprecated: Use llm_d_epp_disagg_decision_total] Total number of disaggregation routing decisions made
		# TYPE llm_d_inference_scheduler_disagg_decision_total counter
		llm_d_inference_scheduler_disagg_decision_total{decision_type="decode-only",model_name="unknown"} 1
	`

	if err := testutil.CollectAndCompare(SchedulerDisaggDecisionCount, strings.NewReader(expected),
		"llm_d_inference_scheduler_disagg_decision_total"); err != nil {
		t.Errorf("RecordDisaggDecision() with empty model failed: %v", err)
	}

	expectedNew := `
		# HELP llm_d_epp_disagg_decision_total [ALPHA] Total number of disaggregation routing decisions made
		# TYPE llm_d_epp_disagg_decision_total counter
		llm_d_epp_disagg_decision_total{decision_type="decode-only",model_name="unknown",plugin_name="test-plugin",plugin_type="test-type"} 1
	`

	if err := testutil.CollectAndCompare(LlmdDisaggDecisionCount, strings.NewReader(expectedNew),
		"llm_d_epp_disagg_decision_total"); err != nil {
		t.Errorf("RecordDisaggDecision() new empty model failed: %v", err)
	}
}

func TestDisaggDecisionType(t *testing.T) {
	tests := []struct {
		encodeUsed  bool
		prefillUsed bool
		want        string
	}{
		{false, false, DecisionTypeDecodeOnly},
		{false, true, DecisionTypePrefillDecode},
		{true, false, DecisionTypeEncodeDecode},
		{true, true, DecisionTypeEncodePrefillDecode},
	}
	for _, tt := range tests {
		got := DisaggDecisionType(tt.encodeUsed, tt.prefillUsed)
		if got != tt.want {
			t.Errorf("DisaggDecisionType(%v, %v) = %q, want %q", tt.encodeUsed, tt.prefillUsed, got, tt.want)
		}
	}
}

func TestHandlerRecordsPrefillRouteSelection(t *testing.T) {
	LlmdRouteSelectionsTotal.Reset()
	h := NewDisaggProfileHandler(defaultDecodeProfile, defaultPrefillProfile, "", nil, nil)
	request := &scheduling.InferenceRequest{Headers: map[string]string{}}

	_, err := h.ProcessResults(context.Background(), request, map[string]*scheduling.ProfileRunResult{
		defaultDecodeProfile:  makeProfileRunResult("decode-a"),
		defaultPrefillProfile: makeProfileRunResult("prefill-a", "prefill-b"),
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = h.ProcessResults(context.Background(), request, map[string]*scheduling.ProfileRunResult{
		defaultDecodeProfile: makeProfileRunResult("decode-a"),
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := `
		# HELP llm_d_epp_route_selections_total [ALPHA] Total number of selected endpoints by disaggregated serving role.
		# TYPE llm_d_epp_route_selections_total counter
		llm_d_epp_route_selections_total{endpoint_name="prefill-a",role="prefill"} 1
	`
	if err := testutil.CollectAndCompare(LlmdRouteSelectionsTotal, strings.NewReader(expected),
		"llm_d_epp_route_selections_total"); err != nil {
		t.Errorf("prefill route selection metric comparison failed: %v", err)
	}
}
