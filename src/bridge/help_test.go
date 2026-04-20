package bridge

import "testing"

func TestReadHelpDocPrefersEmbeddedProvider(t *testing.T) {
	bridge := &Bridge{
		readHelp: func(docID string) ([]byte, error) {
			return []byte("embedded-" + docID), nil
		},
	}

	content, err := bridge.ReadHelpDoc("help")
	if err != nil {
		t.Fatalf("ReadHelpDoc returned error: %v", err)
	}
	if content != "embedded-help" {
		t.Fatalf("expected embedded help content, got %q", content)
	}
}
