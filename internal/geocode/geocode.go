// Package geocode resolves a location string (a "lat,lng" pair, a 5-digit ZIP, or
// a street address) into coordinates, so distances can be measured from the user's
// actual location rather than snapped to a store.
//
//   - ZIP → an embedded US ZCTA centroid table (offline, public-domain Census data).
//   - address → the free US Census geocoder (no API key).
//
// It is internal so the embedded ZIP table and the geocoder dependency never reach
// importers of the public vabc packages.
package geocode

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	_ "embed"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

//go:embed data/zcta.csv.gz
var zctaGz []byte

// Point is a WGS84 coordinate.
type Point struct{ Lat, Lng float64 }

var (
	zipsOnce sync.Once
	zips     map[string]Point
)

func loadZips() {
	zips = make(map[string]Point, 34000)
	gr, err := gzip.NewReader(bytes.NewReader(zctaGz))
	if err != nil {
		return
	}
	defer gr.Close()
	sc := bufio.NewScanner(gr)
	sc.Buffer(make([]byte, 64*1024), 1<<20)
	for sc.Scan() {
		parts := strings.Split(sc.Text(), ",")
		if len(parts) != 3 {
			continue
		}
		lat, err1 := strconv.ParseFloat(parts[1], 64)
		lng, err2 := strconv.ParseFloat(parts[2], 64)
		if err1 != nil || err2 != nil {
			continue
		}
		zips[parts[0]] = Point{Lat: lat, Lng: lng}
	}
}

// LookupZIP returns the centroid of a 5-digit ZIP from the embedded table.
func LookupZIP(zip string) (Point, bool) {
	zipsOnce.Do(loadZips)
	p, ok := zips[zip]
	return p, ok
}

// Resolve turns a location string into coordinates and a human label.
func Resolve(ctx context.Context, loc string) (Point, string, error) {
	loc = strings.TrimSpace(loc)
	if lat, lng, ok := ParseLatLng(loc); ok {
		return Point{Lat: lat, Lng: lng}, loc, nil
	}
	if IsZIP(loc) {
		if p, ok := LookupZIP(loc); ok {
			return p, "ZIP " + loc, nil
		}
		return Point{}, "", fmt.Errorf("unknown ZIP %q", loc)
	}
	return geocodeAddress(ctx, loc)
}

// geocodeAddress resolves a street address via the free US Census geocoder.
func geocodeAddress(ctx context.Context, addr string) (Point, string, error) {
	u := "https://geocoding.geo.census.gov/geocoder/locations/onelineaddress" +
		"?benchmark=Public_AR_Current&format=json&address=" + url.QueryEscape(addr)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return Point{}, "", err
	}
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return Point{}, "", fmt.Errorf("geocoder request failed: %w", err)
	}
	defer resp.Body.Close()

	var out struct {
		Result struct {
			AddressMatches []struct {
				MatchedAddress string `json:"matchedAddress"`
				Coordinates    struct {
					X float64 `json:"x"` // lng
					Y float64 `json:"y"` // lat
				} `json:"coordinates"`
			} `json:"addressMatches"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return Point{}, "", fmt.Errorf("geocoder response: %w", err)
	}
	if len(out.Result.AddressMatches) == 0 {
		return Point{}, "", fmt.Errorf("no geocoding match for %q", addr)
	}
	m := out.Result.AddressMatches[0]
	return Point{Lat: m.Coordinates.Y, Lng: m.Coordinates.X}, m.MatchedAddress, nil
}

// ParseLatLng parses a "lat,lng" string.
func ParseLatLng(s string) (lat, lng float64, ok bool) {
	parts := strings.SplitN(strings.TrimSpace(s), ",", 2)
	if len(parts) != 2 {
		return 0, 0, false
	}
	a, err1 := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	b, err2 := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if err1 != nil || err2 != nil {
		return 0, 0, false
	}
	return a, b, true
}

// IsZIP reports whether s is a 5-digit US ZIP.
func IsZIP(s string) bool {
	s = strings.TrimSpace(s)
	if len(s) != 5 {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
