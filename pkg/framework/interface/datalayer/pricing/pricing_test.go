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

import "testing"

// TestAttributeKey locks the wire-visible attribute key string: producers (config
// loaders) and consumers (scorers) read this value, and changing it silently
// would break the contract between them.
func TestAttributeKey(t *testing.T) {
	if TokenPricesAttributeKey != "token_prices" {
		t.Errorf("TokenPricesAttributeKey = %q, want %q", TokenPricesAttributeKey, "token_prices")
	}
}

// TestTokenPricesClone verifies that Clone returns an independent *TokenPrices
// carrying the same field values, and that mutating either field on the clone
// does not affect the original.
func TestTokenPricesClone(t *testing.T) {
	original := &TokenPrices{InputTokenPrice: 1.5, OutputTokenPrice: 4.5}
	cloned := original.Clone()

	c, ok := cloned.(*TokenPrices)
	if !ok {
		t.Fatal("Clone() did not return *TokenPrices type")
	}
	if c.InputTokenPrice != original.InputTokenPrice || c.OutputTokenPrice != original.OutputTokenPrice {
		t.Errorf("Clone() = %+v, want %+v", c, original)
	}

	c.InputTokenPrice = 100.0
	c.OutputTokenPrice = 200.0
	if original.InputTokenPrice == 100.0 || original.OutputTokenPrice == 200.0 {
		t.Errorf("Clone() did not create an independent copy: original mutated to %+v", original)
	}
}

// TestToTokenPrices_Nil verifies that a nil *ModelPriceShape produces a zero-valued
// TokenPrices ("free model"), which is the invariant downstream consumers rely on
// when an operator omits the pricing block from a model entry.
func TestToTokenPrices_Nil(t *testing.T) {
	tp := ToTokenPrices(nil)
	if tp == nil {
		t.Fatal("ToTokenPrices(nil) returned nil; want zero-valued *TokenPrices")
	}
	if tp.InputTokenPrice != 0 || tp.OutputTokenPrice != 0 {
		t.Errorf("ToTokenPrices(nil) = %+v, want {0, 0}", tp)
	}
}

// TestToTokenPrices_DividesByOneMillion verifies the per-million-to-per-token
// conversion is applied to both fields.
func TestToTokenPrices_DividesByOneMillion(t *testing.T) {
	tp := ToTokenPrices(&ModelPriceShape{InputPerMillion: 2.0, OutputPerMillion: 8.0})
	const eps = 1e-15
	wantIn, wantOut := 2.0/1e6, 8.0/1e6
	if absDiff(tp.InputTokenPrice, wantIn) >= eps {
		t.Errorf("InputTokenPrice = %v, want %v", tp.InputTokenPrice, wantIn)
	}
	if absDiff(tp.OutputTokenPrice, wantOut) >= eps {
		t.Errorf("OutputTokenPrice = %v, want %v", tp.OutputTokenPrice, wantOut)
	}
}

func absDiff(a, b float64) float64 {
	d := a - b
	if d < 0 {
		d = -d
	}
	return d
}
