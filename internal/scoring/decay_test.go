package scoring

import (
	"testing"
)

func TestDecayed_ZeroElapsed(t *testing.T) {
	// At t=0, score should be unchanged
	result := Decayed(0.9, 0)
	tolerance := 0.0001
	if result < 0.9-tolerance || result > 0.9+tolerance {
		t.Errorf("expected 0.9, got %f", result)
	}
}

func TestDecayed_ScoreDropsOverTime(t *testing.T) {
	// Score must be lower after time passes
	score := Decayed(0.9, 600)
	if score >= 0.9 {
		t.Errorf("expected score to decay, got %f", score)
	}
}

func TestDecayed_NegativeElapsed(t *testing.T) {
	// Clock skew guard — should treat as zero
	result := Decayed(0.9, -100)
	if result != 0.9 {
		t.Errorf("expected 0.9 for negative elapsed, got %f", result)
	}
}

func TestDecayed_Clamp(t *testing.T) {
	// A score at exactly 1.0 with zero elapsed should stay at 1.0
	result := Decayed(1.0, 0)
	tolerance := 0.0001
	if result < 1.0-tolerance || result > 1.0+tolerance {
		t.Errorf("expected clamp to 1.0, got %f", result)
	}
}

func TestEvaluate_Tiers(t *testing.T) {
	tests := []struct {
		score    float64
		expected Tier
	}{
		{0.1, TierPass},
		{0.29, TierPass},
		{0.3, TierRateLimit},
		{0.69, TierRateLimit},
		{0.7, TierBlock},
		{0.95, TierBlock},
	}

	for _, tt := range tests {
		result := Evaluate(tt.score)
		if result != tt.expected {
			t.Errorf("score %.2f: expected %s, got %s",
				tt.score, tt.expected, result)
		}
	}
}

func TestRunningAverage(t *testing.T) {
	result := RunningAverage(0.5, 1.0)
	expected := 0.65
	tolerance := 0.0001

	if result < expected-tolerance || result > expected+tolerance {
		t.Errorf("expected ~0.65, got %f", result)
	}
}
