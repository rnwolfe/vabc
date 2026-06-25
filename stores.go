package vabc

import (
	"context"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type rawArcGIS struct {
	Features []struct {
		Attributes map[string]any `json:"attributes"`
		Geometry   struct {
			X float64 `json:"x"`
			Y float64 `json:"y"`
		} `json:"geometry"`
	} `json:"features"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

var storeNumRe = regexp.MustCompile(`(\d+)`)

// Stores returns all Virginia ABC retail stores from the ArcGIS FeatureServer.
func (c *httpClient) Stores(ctx context.Context) ([]Store, error) {
	// outSR=4326 forces WGS84 lat/lng; the layer's native SR is Web Mercator (meters).
	url := c.storesURL + "?where=1%3D1&outFields=*&returnGeometry=true&outSR=4326&f=json"
	var raw rawArcGIS
	if err := c.getJSON(ctx, url, &raw); err != nil {
		return nil, err
	}
	if raw.Error != nil {
		return nil, schemaDrift("ArcGIS error: "+raw.Error.Message, nil)
	}
	stores := make([]Store, 0, len(raw.Features))
	for _, f := range raw.Features {
		a := f.Attributes
		s := Store{
			StoreNumber: storeNumberFrom(attrStr(a, "LandmkName")),
			Name:        attrStr(a, "LandmkName"),
			Address:     attrStr(a, "Address"),
			City:        attrStr(a, "City"),
			State:       attrStr(a, "State"),
			Zip:         attrStr(a, "Zip"),
			Phone:       attrStr(a, "Phone"),
			URL:         attrStr(a, "URL"),
			Lng:         f.Geometry.X,
			Lat:         f.Geometry.Y,
		}
		if s.Lat == 0 && s.Lng == 0 {
			s.Lng, s.Lat = attrFloat(a, "X"), attrFloat(a, "Y")
		}
		stores = append(stores, s)
	}
	return stores, nil
}

// StoreNear returns stores nearest a point, ranked by distance (miles). limit<=0
// returns all.
func (c *httpClient) StoreNear(ctx context.Context, lat, lng float64, limit int) ([]Store, error) {
	stores, err := c.Stores(ctx)
	if err != nil {
		return nil, err
	}
	for i := range stores {
		stores[i].Distance = round1(haversineMiles(lat, lng, stores[i].Lat, stores[i].Lng))
	}
	sort.Slice(stores, func(i, j int) bool { return stores[i].Distance < stores[j].Distance })
	if limit > 0 && len(stores) > limit {
		stores = stores[:limit]
	}
	return stores, nil
}

func storeNumberFrom(landmark string) int {
	m := storeNumRe.FindString(landmark)
	if m == "" {
		return 0
	}
	n, _ := strconv.Atoi(m)
	return n
}

func attrStr(a map[string]any, key string) string {
	switch v := a[key].(type) {
	case string:
		return strings.TrimSpace(v)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	default:
		return ""
	}
}

func attrFloat(a map[string]any, key string) float64 {
	switch v := a[key].(type) {
	case float64:
		return v
	case string:
		f, _ := strconv.ParseFloat(strings.TrimSpace(v), 64)
		return f
	default:
		return 0
	}
}

func haversineMiles(lat1, lon1, lat2, lon2 float64) float64 {
	const earthMiles = 3958.8
	rad := math.Pi / 180
	dLat := (lat2 - lat1) * rad
	dLon := (lon2 - lon1) * rad
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*rad)*math.Cos(lat2*rad)*math.Sin(dLon/2)*math.Sin(dLon/2)
	return earthMiles * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}

func round1(f float64) float64 { return math.Round(f*10) / 10 }
