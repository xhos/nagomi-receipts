package server

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"

	pb "nagomi-receipts/internal/gen/nagomi/v1"
)

const prompt = `Look at this receipt image and extract all information as JSON.

Return ONLY valid JSON in this exact format (no markdown, no backticks, no explanation):
{
  "merchant": "Store Name",
  "date": "YYYY-MM-DD",
  "currency": "CAD",
  "items": [
    {"raw": "KIRKLAND ORG EGGS 2DZ", "name": "Organic Eggs 2 Dozen", "qty": 1.0, "unit_price": 8.99}
  ],
  "subtotal": 45.67,
  "tax": 5.94,
  "total": 51.61
}

Rules:
- "raw" is exactly as printed on receipt
- "name" is your best guess at the full product name
- "qty" defaults to 1.0 if not specified
- "unit_price" is price per unit
- "currency" is 3-letter ISO code
- "date" is YYYY-MM-DD if visible, otherwise null
- Use null for values you cannot read
- Skip items with $0.00 price`

// receiptJSON mirrors the shape the prompt asks for. Pointers distinguish
// "unreadable" (null) from zero, which must stay absent in the response.
type receiptJSON struct {
	Merchant string     `json:"merchant"`
	Date     *string    `json:"date"`
	Currency string     `json:"currency"`
	Items    []itemJSON `json:"items"`
	Subtotal *float64   `json:"subtotal"`
	Tax      *float64   `json:"tax"`
	Total    *float64   `json:"total"`
}

type itemJSON struct {
	Raw       string   `json:"raw"`
	Name      string   `json:"name"`
	Qty       *float64 `json:"qty"`
	UnitPrice *float64 `json:"unit_price"`
}

// parseReceipt turns the model's reply into a receipt, ignoring any markdown
// fences or chatter around the JSON object.
func parseReceipt(reply string) (*pb.ParsedReceipt, error) {
	start, end := strings.Index(reply, "{"), strings.LastIndex(reply, "}")
	if start < 0 || end < start {
		return nil, fmt.Errorf("no JSON object in model reply")
	}

	var r receiptJSON
	if err := json.Unmarshal([]byte(reply[start:end+1]), &r); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	receipt := &pb.ParsedReceipt{
		Merchant: nonEmpty(r.Merchant),
		Date:     r.Date,
		Currency: nonEmpty(r.Currency),
		Subtotal: r.Subtotal,
		Tax:      r.Tax,
		Total:    r.Total,
	}
	for _, it := range r.Items {
		receipt.Items = append(receipt.Items, &pb.ParsedItem{
			Raw:       it.Raw,
			Name:      nonEmpty(it.Name),
			Qty:       deref(it.Qty, 1),
			UnitPrice: deref(it.UnitPrice, 0),
		})
	}
	receipt.Confidence = confidence(receipt)

	return receipt, nil
}

// confidence is a heuristic: 0.5 base, +0.1 for each key field present,
// and +0.1 when the items add up to the subtotal within 5%.
func confidence(r *pb.ParsedReceipt) float32 {
	score := float32(0.5)
	for _, present := range []bool{r.Merchant != nil, r.Date != nil, r.Total != nil, len(r.Items) > 0} {
		if present {
			score += 0.1
		}
	}

	if r.Subtotal != nil && len(r.Items) > 0 {
		var sum float64
		for _, it := range r.Items {
			sum += it.Qty * it.UnitPrice
		}
		if math.Abs(sum-*r.Subtotal) <= *r.Subtotal*0.05 {
			score += 0.1
		}
	}

	return min(score, 1)
}

func nonEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
