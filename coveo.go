package vabc

import (
	"context"
	"fmt"
	"strings"
)

// coveoSearchPath is the site's Sitecore-proxied Coveo search endpoint. It is
// undocumented but publicly queryable (anonymous), and indexes the full web
// catalog — including products absent from the downloadable price list.
const coveoSearchPath = "/coveo/rest/search/v2"

type coveoResponse struct {
	TotalCount int `json:"totalCount"`
	Results    []struct {
		ClickURI string         `json:"clickUri"`
		Raw      map[string]any `json:"raw"`
	} `json:"results"`
}

// SearchProducts queries Coveo for products matching query (max `limit`).
func (c *httpClient) SearchProducts(ctx context.Context, query string, limit int) ([]Product, error) {
	if limit <= 0 {
		limit = 50
	}
	reqBody := map[string]any{
		"q":               query,
		"numberOfResults": limit,
		"firstResult":     0,
	}
	var resp coveoResponse
	if err := c.postJSON(ctx, c.baseURL+coveoSearchPath, reqBody, &resp); err != nil {
		return nil, err
	}
	out := make([]Product, 0, len(resp.Results))
	for _, r := range resp.Results {
		p, ok := productFromCoveo(r.Raw, r.ClickURI)
		if !ok {
			continue // skip non-product results (no SKU)
		}
		out = append(out, p)
	}
	return out, nil
}

// productFromCoveo maps a Coveo raw-field map to a Product. Coveo encodes special
// characters in field names (z32x=space, z95x=underscore, z120x=x, z122x=z), hence
// the opaque keys. Returns ok=false when the result has no product SKU.
func productFromCoveo(raw map[string]any, clickURI string) (Product, bool) {
	code := pad6(firstToken(rawString(raw["z95xproductz32xskuz32xids"])))
	if code == "" || code == "000000" {
		return Product{}, false
	}
	p := Product{
		ProductCode:     code,
		Name:            rawString(raw["productz32xlabelz32xname"]),
		Category:        rawString(raw["hierarchyz32xcategory"]),
		Type:            strings.Trim(rawString(raw["hierarchyz32xtype"]), "[]"),
		Size:            rawString(raw["z95xproductz32xsiz122xes"]),
		Proof:           rawFloatPtr(raw["proofmin"]),
		RetailPrice:     rawFloatPtr(raw["z95xproductz32xpricez32xsort"]),
		Allocated:       rawBool01(raw["z95xproductz32xlimitedz32xavailability"]) || rawBool01(raw["z95xproductz32xlottery"]),
		OnlineOrderable: rawBool01(raw["z95xproductz32xdirectz32xship"]),
		New:             rawBool01(raw["z95xnewz32xproduct"]),
		URL:             clickURI,
	}
	if p.Name == "" {
		p.Name = rawString(raw["pagez32xtitle"])
	}
	return p, true
}

// rawString renders a Coveo value (string, number, or single-element slice) as a
// trimmed string.
func rawString(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(t)
	case float64:
		return strings.TrimSpace(fmt.Sprintf("%v", t))
	case []any:
		if len(t) == 0 {
			return ""
		}
		return rawString(t[0])
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", t))
	}
}

func rawFloatPtr(v any) *float64 {
	switch t := v.(type) {
	case float64:
		return &t
	case []any:
		if len(t) > 0 {
			return rawFloatPtr(t[0])
		}
	case string:
		var f float64
		if _, err := fmt.Sscanf(strings.TrimSpace(t), "%g", &f); err == nil {
			return &f
		}
	}
	return nil
}

func rawBool01(v any) bool {
	switch t := v.(type) {
	case float64:
		return t != 0
	case bool:
		return t
	case string:
		return t == "1" || strings.EqualFold(t, "true")
	case []any:
		if len(t) > 0 {
			return rawBool01(t[0])
		}
	}
	return false
}

// firstToken returns the first whitespace/comma-separated token of s.
func firstToken(s string) string {
	s = strings.TrimSpace(s)
	for i, r := range s {
		if r == ' ' || r == ',' {
			return s[:i]
		}
	}
	return s
}
