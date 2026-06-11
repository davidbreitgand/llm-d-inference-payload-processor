/*
Copyright 2026 The llm-d Authors.

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

package pricing

import (
	"testing"

	"github.com/caio/go-tdigest/v5"
)

// newDigest builds a tdigest with the proposal's default compression and
// adds the given samples. It fails the test on any library error so that
// the call sites stay free of error handling.
func newDigest(t *testing.T, samples ...float64) *tdigest.TDigest {
	t.Helper()
	d, err := tdigest.New(tdigest.Compression(200))
	if err != nil {
		t.Fatalf("tdigest.New: %v", err)
	}
	for _, s := range samples {
		if err := d.Add(s); err != nil {
			t.Fatalf("tdigest.Add(%v): %v", s, err)
		}
	}
	return d
}

// TestCostDigestClone_IndependentMutation verifies the fundamental
// Cloneable contract: adding a sample to the clone must not bleed into
// the original, and vice versa. AttributeMap.Get clones on every read,
// so violating this would let consumers mutate stored state.
func TestCostDigestClone_IndependentMutation(t *testing.T) {
	original := &CostDigest{Digest: newDigest(t, 1.0, 2.0, 3.0)}

	cloned := original.Clone()
	cd, ok := cloned.(*CostDigest)
	if !ok {
		t.Fatalf("Clone() returned %T, want *CostDigest", cloned)
	}

	originalCount := original.Digest.Count()
	if cd.Digest.Count() != originalCount {
		t.Fatalf("clone count = %d, want %d", cd.Digest.Count(), originalCount)
	}

	// Mutate the clone; the original must not see the new sample.
	if err := cd.Digest.Add(1000.0); err != nil {
		t.Fatalf("clone Add: %v", err)
	}
	if original.Digest.Count() != originalCount {
		t.Errorf("original mutated by clone: count = %d, want %d",
			original.Digest.Count(), originalCount)
	}

	// Mutate the original; the clone must not see the new sample.
	cloneCount := cd.Digest.Count()
	if err := original.Digest.Add(2000.0); err != nil {
		t.Fatalf("original Add: %v", err)
	}
	if cd.Digest.Count() != cloneCount {
		t.Errorf("clone mutated by original: count = %d, want %d",
			cd.Digest.Count(), cloneCount)
	}
}

// TestCostDigestClone_PreservesQuantiles verifies that a clone reports
// the same percentile estimates as its source, so CostGuard reading the
// snapshot via AttributeMap.Get sees the same distribution the extractor
// observed.
func TestCostDigestClone_PreservesQuantiles(t *testing.T) {
	// A skewed distribution: most cheap, a few expensive — matches the
	// shape CostGuard's TrimmedMean + CTE rank is designed to handle.
	samples := []float64{1, 1, 1, 2, 2, 3, 3, 5, 8, 100}
	original := &CostDigest{Digest: newDigest(t, samples...)}

	cloned := original.Clone().(*CostDigest)

	for _, q := range []float64{0.5, 0.9, 0.95, 0.99} {
		want := original.Digest.Quantile(q)
		got := cloned.Digest.Quantile(q)
		if got != want {
			t.Errorf("Quantile(%v): clone = %v, original = %v", q, got, want)
		}
	}
}
