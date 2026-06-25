package geocode

import (
	"context"
	"testing"
)

func TestParseLatLng(t *testing.T) {
	lat, lng, ok := ParseLatLng(" 38.91 , -77.23 ")
	if !ok || lat != 38.91 || lng != -77.23 {
		t.Fatalf("got %v,%v ok=%v", lat, lng, ok)
	}
	if _, _, ok := ParseLatLng("22182"); ok {
		t.Fatalf("a bare ZIP is not lat,lng")
	}
}

func TestZIPCentroid(t *testing.T) {
	// 22182 is Vienna, VA — sanity-check the embedded centroid is in the right region.
	p, ok := LookupZIP("22182")
	if !ok {
		t.Fatal("22182 not found in embedded ZCTA table")
	}
	if p.Lat < 38 || p.Lat > 39.5 || p.Lng > -76 || p.Lng < -78 {
		t.Fatalf("22182 centroid out of NoVA range: %+v", p)
	}
	if _, ok := LookupZIP("00000"); ok {
		t.Fatal("00000 should not resolve")
	}
}

func TestResolveZIPOffline(t *testing.T) {
	p, label, err := Resolve(context.Background(), "22182")
	if err != nil {
		t.Fatal(err)
	}
	if label != "ZIP 22182" || p.Lat < 38 || p.Lat > 39.5 {
		t.Fatalf("unexpected resolve: %v %q", p, label)
	}
}
