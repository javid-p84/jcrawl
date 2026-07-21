package models

import "time"

// PreferenceCheck is a record of one worker pass over a preference: whether
// it succeeded, how many sites/slots were examined, how many matched, and
// (if any) the best candidate found — so users can review check history
// instead of only ever seeing the outcome of matches that triggered a
// notification or booking.
type PreferenceCheck struct {
	ID           string    `json:"id"`
	PreferenceID string    `json:"preference_id"`
	UserID       string    `json:"user_id"`
	CheckedAt    time.Time `json:"checked_at"`
	Success      bool      `json:"success"`
	ErrorMessage string    `json:"error_message,omitempty"`
	SitesChecked int       `json:"sites_checked"`
	MatchesFound int       `json:"matches_found"`

	// BestMatch is the most likely candidate among the matches found (soonest
	// check-in date), if any.
	BestMatchLabel string     `json:"best_match_label,omitempty"`
	BestMatchDate  *time.Time `json:"best_match_date,omitempty"`
	BestMatchURL   string     `json:"best_match_url,omitempty"`
}
