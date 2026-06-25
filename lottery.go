package vabc

import (
	"context"
	"encoding/json"
	"fmt"
)

// LimitedAvailability returns the limited-availability ("lottery"/allocated) event
// hook for a product. The endpoint returns {} when there is no active drop; the
// populated shape is undocumented, so this decodes defensively and surfaces any
// event-link-shaped entries it can find. Allocated is left to the caller to fill
// from the catalog flag.
func (c *httpClient) LimitedAvailability(ctx context.Context, productCode string) (LotteryResult, error) {
	code := pad6(productCode)
	url := fmt.Sprintf("%s/webapi/limitedavailability/eventLinks?productCode=%s", c.baseURL, code)

	var raw json.RawMessage
	if err := c.getJSON(ctx, url, &raw); err != nil {
		return LotteryResult{}, err
	}

	res := LotteryResult{ProductCode: code, EventLinks: []LotteryEvent{}}

	// Empty object / null => no active event.
	trimmed := trimSpaceBytes(raw)
	if len(trimmed) == 0 || string(trimmed) == "{}" || string(trimmed) == "null" {
		return res, nil
	}

	res.EventLinks = extractEvents(raw)
	res.Active = len(res.EventLinks) > 0 || hasContent(raw)
	return res, nil
}

// extractEvents pulls {title,url}-shaped entries from an arbitrary JSON value:
// an array of link objects, or an object whose values contain such arrays.
func extractEvents(raw json.RawMessage) []LotteryEvent {
	var events []LotteryEvent

	var arr []map[string]any
	if json.Unmarshal(raw, &arr) == nil {
		for _, m := range arr {
			if e, ok := eventFrom(m); ok {
				events = append(events, e)
			}
		}
		if len(events) > 0 {
			return events
		}
	}

	var obj map[string]json.RawMessage
	if json.Unmarshal(raw, &obj) == nil {
		for _, v := range obj {
			events = append(events, extractEvents(v)...)
		}
	}
	return events
}

func eventFrom(m map[string]any) (LotteryEvent, bool) {
	title := firstString(m, "title", "name", "text", "label")
	url := firstString(m, "url", "link", "href")
	if title == "" && url == "" {
		return LotteryEvent{}, false
	}
	return LotteryEvent{Title: title, URL: url}, true
}

func firstString(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k].(string); ok && v != "" {
			return v
		}
	}
	return ""
}

func hasContent(raw json.RawMessage) bool {
	var obj map[string]json.RawMessage
	if json.Unmarshal(raw, &obj) == nil {
		return len(obj) > 0
	}
	var arr []json.RawMessage
	if json.Unmarshal(raw, &arr) == nil {
		return len(arr) > 0
	}
	return false
}

func trimSpaceBytes(b []byte) []byte {
	i, j := 0, len(b)
	for i < j && (b[i] == ' ' || b[i] == '\n' || b[i] == '\t' || b[i] == '\r') {
		i++
	}
	for j > i && (b[j-1] == ' ' || b[j-1] == '\n' || b[j-1] == '\t' || b[j-1] == '\r') {
		j--
	}
	return b[i:j]
}
