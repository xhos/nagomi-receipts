package server

import (
	"testing"
)

func TestParseReceipt(t *testing.T) {
	reply := "Sure! Here it is:\n```json\n" + `{
		"merchant": "Costco",
		"date": null,
		"currency": "",
		"items": [
			{"raw": "KS EGGS", "name": "Eggs", "unit_price": 8.99},
			{"raw": "MILK", "name": "", "qty": 2, "unit_price": 3.50}
		],
		"subtotal": 15.99,
		"tax": null,
		"total": 15.99
	}` + "\n```"

	r, err := parseReceipt(reply)
	if err != nil {
		t.Fatal(err)
	}

	if deref(r.Merchant, "") != "Costco" {
		t.Errorf("merchant = %v", r.Merchant)
	}
	if r.Date != nil || r.Currency != nil || r.Tax != nil {
		t.Errorf("unreadable values should stay absent: date=%v currency=%v tax=%v", r.Date, r.Currency, r.Tax)
	}
	if len(r.Items) != 2 {
		t.Fatalf("items = %d", len(r.Items))
	}
	if r.Items[0].Qty != 1 {
		t.Errorf("missing qty should default to 1, got %v", r.Items[0].Qty)
	}
	if r.Items[1].Name != nil {
		t.Errorf("empty name should be absent, got %q", *r.Items[1].Name)
	}
	// merchant, total, items, and items sum to subtotal; no date
	if r.Confidence < 0.89 || r.Confidence > 0.91 {
		t.Errorf("confidence = %v, want 0.9", r.Confidence)
	}
}

func TestParseReceiptRejectsNonJSON(t *testing.T) {
	for _, reply := range []string{"", "I can't read this receipt.", "{not json}"} {
		if _, err := parseReceipt(reply); err == nil {
			t.Errorf("parseReceipt(%q) should fail", reply)
		}
	}
}
