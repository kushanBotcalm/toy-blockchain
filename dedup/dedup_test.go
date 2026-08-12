package dedup

import (
	"encoding/json"
	"testing"
)

func TestDeduperJSONRoundTrip(t *testing.T) {
	d := NewDeduper()
	if d.SeenTransaction("tx-1") {
		t.Fatal("expected first transaction to be unseen")
	}
	if d.SeenBlock("block-1") {
		t.Fatal("expected first block hash to be unseen")
	}

	data, err := json.Marshal(d)
	if err != nil {
		t.Fatalf("marshal deduper: %v", err)
	}

	var restored Deduper
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("unmarshal deduper: %v", err)
	}

	if !restored.SeenTransaction("tx-1") {
		t.Fatal("expected restored deduper to remember tx-1")
	}
	if !restored.SeenBlock("block-1") {
		t.Fatal("expected restored deduper to remember block-1")
	}
}
