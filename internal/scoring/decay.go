package scoring

import "math"

// Tier represents the decision made after evaluating a decayed threat score.
// It tells the gateway what to do with this request.
type Tier int

const (
	// TierPass - score is low, request goes through normally
	TierPass Tier = iota

	// TireRateLimit - score is medium, respond with 429
	TierRateLimit

	// TierBlock - score is high, respond with 403 and close connection
	TierBlock
)

const (
	// Lambda (λ) controls how fast scores decay.
	// 0.001 means a score of 0.9 takes ~1200 seconds (~20 min) to drop below 0.3.
	// Increase λ to forgive clients faster. Decrease to punish longer.
	// This is a tuning knob
	lambda = 0.001

	// Thresholds for tiered decisions.
	// These are the only two numbers you change to tune gateway aggression.
	blockThreshold     = 0.7 // score >= this -> block
	rateLimitThreshold = 0.3 // score >= this -> rate limit
)

// Decayed computes the efective threat score at the current moment.
//
// Formula: effectiveScore = storeScore x e^(-λ x secondsElapsed)
//
// Parameters:
// 	 storedScore 	- the raw score written by the ML worker (0.0 to 1.0)
// 	 secondsElapsed - time.Since(lastWritten).Seconds() computed by caller
func Decayed(storedScore, secondsElapsed float64) float64 {
	// Guard: negative elapsed time should not happend in practice (clock skew
	// between nodes could theoretically cause it). Treat as zero elapsed.
	if secondsElapsed < 0 {
		secondsElapsed = 0
	}

	decayed := storedScore * math.Exp(-lambda*secondsElapsed)

	// Clamp to [0.0, 1.0]. math.Exp never produces negative values, but
	// floating point arithmentic can produce values infinitesimally above 1.0
	// if storedScore was written slightly above 1.0 by a bug.
	if decayed > 1.0 {
		return 1.0
	}

	return decayed
}

// Evaluate takes a decayed score and returns the Tier decision.
func Evaluate(decayedScore float64) Tier {
	switch {
	case decayedScore >= blockThreshold:
		return TierBlock
	case decayedScore >= rateLimitThreshold:
		return TierRateLimit
	default:
		return TierPass
	}
}

// RunningAverage computes the new score using an exponential moving average.
//
// Formula: newScore = 0.7*oldScore + 0.3*latestScore
//
// The 0.7 weight on the old score gives it inertia — one anomalous request
// from an otherwise clean client doesn't spike their score dramatically.
// The 0.3 weight on the latest observation means new signals do matter,
// just not overwhelmingly. These weights are a tuning decision; 0.8/0.2
// would be more conservative, 0.5/0.5 more reactive.
func RunningAverage(oldScore, latestScore float64) float64 {
	return 0.7*oldScore + 0.3*latestScore
}

// String implements fmt.Stringer for readable logs and test output.
func (t Tier) String() string {
	switch t {
	case TierPass:
		return "pass"
	case TierRateLimit:
		return "rate_limit"
	case TierBlock:
		return "block"
	default:
		return "unknown"
	}
}
