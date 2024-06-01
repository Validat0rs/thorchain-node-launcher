package monitor

import (
	"testing"
)

// Test the IMAGE_FILTER regex
func TestImageFilterRegex(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"thorchain/devops/node-launcher:test", true},
		{"thorchain/thornode:chaosnet-multichain-1.2.3", true},
		{"thorchain/thornode:mainnet-1.2.3", true},
		{"thorchain/midgard:1.2.3", true},
		{"invalid/repo:tag", false},
		{"thorchain/unknown:1.2.3", false},
	}

	for _, test := range tests {
		result := IMAGE_FILTER.MatchString(test.input)
		if result != test.expected {
			t.Errorf("expected %v for input %q, got %v", test.expected, test.input, result)
		}
	}
}

func TestCheckImageChanges(t *testing.T) {
	mockFetchImages := func() ([]Image, error) {
		return []Image{
			{Repo: "thorchain/devops/node-launcher", Tag: "test", Hash: "hash1"},
			{Repo: "thorchain/thornode", Tag: "chaosnet-multichain-1.2.3", Hash: "hash2"},
			{Repo: "thorchain/midgard", Tag: "1.2.3", Hash: "hash3"},
		}, nil
	}

	// Initial run, all images should be new
	alerts, err := checkImageChanges(mockFetchImages)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(alerts) != 0 {
		t.Errorf("expected no alerts, got %d", len(alerts))
	}

	// Modify the image hash to simulate a change
	mockFetchImages = func() ([]Image, error) {
		return []Image{
			{Repo: "thorchain/devops/node-launcher", Tag: "test", Hash: "hash1-modified"},
			{Repo: "thorchain/thornode", Tag: "chaosnet-multichain-1.2.3", Hash: "hash2"},
			{Repo: "thorchain/midgard", Tag: "1.2.3", Hash: "hash3"},
		}, nil
	}

	alerts, err = checkImageChanges(mockFetchImages)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(alerts) == 0 {
		t.Error("expected alerts for modified images, got none")
	}
}
