package monitor

import (
	"encoding/json"
	"testing"
)

// Mock fetch function
func mockFetch(url string, target interface{}) error {
	mockData := map[string]interface{}{
		"https://api.github.com/repos/bnb-chain/tss-lib/branches/master": map[string]interface{}{
			"commit": map[string]interface{}{
				"sha":     "mock-sha-1",
				"message": "Mock commit message",
			},
		},
		"https://api.github.com/repos/bnb-chain/tss-lib/branches": []map[string]interface{}{
			{"name": "mock-branch-1"},
			{"name": "mock-branch-2"},
		},
		"https://api.github.com/repos/bnb-chain/tss-lib/pulls": []map[string]interface{}{
			{"number": 1, "title": "Mock PR 1", "html_url": "https://github.com/bnb-chain/tss-lib/pull/1"},
			{"number": 2, "title": "Mock PR 2", "html_url": "https://github.com/bnb-chain/tss-lib/pull/2"},
		},
	}

	if data, ok := mockData[url]; ok {
		dataBytes, _ := json.Marshal(data)
		return json.Unmarshal(dataBytes, target)
	}

	return nil
}

// Test the SecurityUpdateMonitor
func TestSecurityUpdateMonitor(t *testing.T) {
	monitor := NewSecurityUpdatesMonitor()

	// Initial run, all states should be new
	alerts, err := checkSecurityUpdates(mockFetch, monitor.lastCommit, monitor.lastBranches, monitor.lastPRs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(alerts) != 0 {
		t.Errorf("expected no alerts, got %d", len(alerts))
	}

	// Modify the state to simulate changes
	monitor.lastCommit["bnb-chain/tss-lib"] = "old-sha"
	monitor.lastBranches["bnb-chain/tss-lib"] = map[string]struct{}{"old-branch": {}}
	monitor.lastPRs["bnb-chain/tss-lib"] = map[int]struct{}{0: {}}

	alerts, err = checkSecurityUpdates(mockFetch, monitor.lastCommit, monitor.lastBranches, monitor.lastPRs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(alerts) == 0 {
		t.Error("expected alerts for new commits, branches, and PRs, got none")
	}
}
