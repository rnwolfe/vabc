package vabc

import (
	"context"
	"html"
	"net/url"
	"regexp"
	"strings"
)

// priceListPagePath is the (publicly fetchable) product-downloads page that links
// the current quarterly price-list workbook.
const priceListPagePath = "/products/products-faqs/product-downloads"

// quarterlyXLSXRe extracts the quarterly price-list .xlsx href from the page.
var quarterlyXLSXRe = regexp.MustCompile(`(?i)["']([^"']*quarterly-price-list[^"']*\.xlsx[^"']*)["']`)

// LatestPriceListURL discovers the current quarterly price-list XLSX URL by reading
// the public product-downloads page. Returns an absolute URL.
func LatestPriceListURL(ctx context.Context, opts ...Option) (string, error) {
	c := NewClient(opts...).(*httpClient)
	return c.latestPriceListURL(ctx)
}

func (c *httpClient) latestPriceListURL(ctx context.Context) (string, error) {
	body, err := c.fetchBytes(ctx, c.baseURL+priceListPagePath, "text/html")
	if err != nil {
		return "", err
	}
	m := quarterlyXLSXRe.FindStringSubmatch(string(body))
	if m == nil {
		return "", schemaDrift("could not find a quarterly price-list .xlsx link on the downloads page", nil)
	}
	return c.absoluteURL(html.UnescapeString(m[1])), nil
}

// FetchLatestPriceList discovers and downloads ABC's current quarterly price-list
// XLSX, returning the bytes and the source URL. It shares the client's throttle, so
// it is a polite citizen of the undocumented site like every other call.
func FetchLatestPriceList(ctx context.Context, opts ...Option) (data []byte, sourceURL string, err error) {
	c := NewClient(opts...).(*httpClient)
	sourceURL, err = c.latestPriceListURL(ctx)
	if err != nil {
		return nil, "", err
	}
	data, err = c.fetchBytes(ctx, sourceURL, "application/octet-stream")
	if err != nil {
		return nil, sourceURL, err
	}
	return data, sourceURL, nil
}

func (c *httpClient) absoluteURL(href string) string {
	if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
		return href
	}
	base, err := url.Parse(c.baseURL)
	if err != nil {
		return c.baseURL + href
	}
	ref, err := url.Parse(href)
	if err != nil {
		return c.baseURL + href
	}
	return base.ResolveReference(ref).String()
}
