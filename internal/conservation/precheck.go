package conservation

import "time"

type PrecheckItem struct {
	Code     string `json:"code"`
	Stage    string `json:"stage"`
	Message  string `json:"message"`
	Blocking bool   `json:"blocking"`
	Passed   bool   `json:"passed"`
}

type PrecheckResult struct {
	CheckedAt      time.Time      `json:"checked_at"`
	Items          []PrecheckItem `json:"items"`
	BlockingItems  []PrecheckItem `json:"blocking_items"`
	HintItems      []PrecheckItem `json:"hint_items"`
	ConfirmedHints []string       `json:"confirmed_hints,omitempty"`
}

func (p PrecheckResult) Blocking() []PrecheckItem {
	result := make([]PrecheckItem, 0)
	for _, item := range p.Items {
		if item.Blocking && !item.Passed {
			result = append(result, item)
		}
	}
	return result
}
