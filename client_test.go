package vabc_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/rnwolfe/vabc"
)

func testClient(t *testing.T, srv *httptest.Server, opts ...vabc.Option) vabc.Client {
	t.Helper()
	base := []vabc.Option{
		vabc.WithBaseURL(srv.URL),
		vabc.WithStoresURL(srv.URL + "/arcgis"),
		vabc.WithMinInterval(0),
		vabc.WithStatePath(filepath.Join(t.TempDir(), "throttle.json")),
	}
	return vabc.NewClient(append(base, opts...)...)
}

const storeNearbyJSON = `{"products":[{"productId":"010807","storeInfo":{
  "storeId":219,"quantity":17,"distance":0.0,"latitude":38.915434,"longitude":-77.236379,
  "address":"8413 Old Courthouse Road Vienna VA 22182","address1":"8413 Old Courthouse Road",
  "address2":null,"city":"Vienna","state":"VA","zip":"22182",
  "PhoneNumber":{"FormattedPhoneNumber":"(571) 620-1255"},"url":"/stores/219",
  "hours":"Mon-Sat 10 am-10 pm","shoppingCenter":"8415 Building"},
  "nearbyStores":[{"storeId":231,"quantity":26,"distance":1.2,"latitude":38.9,"longitude":-77.2,
  "city":"Fairfax","state":"VA","zip":"22030","PhoneNumber":{"FormattedPhoneNumber":"(703) 1"}}]}]}`

func TestStoreNearby(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("storeNumber") == "" || r.URL.Query().Get("productCode") == "" {
			http.Error(w, `{"message":"Missing required parameter"}`, 400)
			return
		}
		_, _ = w.Write([]byte(storeNearbyJSON))
	}))
	defer srv.Close()

	res, err := testClient(t, srv).StoreNearby(context.Background(), 219, "10807")
	if err != nil {
		t.Fatal(err)
	}
	if res.ProductCode != "010807" {
		t.Fatalf("productCode = %q, want 010807 (zero-padded)", res.ProductCode)
	}
	if res.Store.StoreNumber != 219 || res.Store.Quantity != 17 {
		t.Fatalf("store = %+v", res.Store)
	}
	if res.Store.Phone != "(571) 620-1255" {
		t.Fatalf("phone not mapped: %q", res.Store.Phone)
	}
	if len(res.NearbyStores) != 1 || res.NearbyStores[0].StoreNumber != 231 || res.NearbyStores[0].Quantity != 26 {
		t.Fatalf("nearby = %+v", res.NearbyStores)
	}
}

func TestStoreNearbyInvalidStore(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"message":"No Store exists for store number '100'"}`, 400)
	}))
	defer srv.Close()

	_, err := testClient(t, srv).StoreNearby(context.Background(), 100, "010807")
	var ae *vabc.APIError
	if !errors.As(err, &ae) || ae.Kind != vabc.KindNotFound {
		t.Fatalf("want NotFound APIError, got %v", err)
	}
	if !strings.Contains(ae.Msg, "No Store exists") {
		t.Fatalf("message not surfaced: %q", ae.Msg)
	}
}

func TestWarehouseParsesString(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"warehouseInventory":"100"}`))
	}))
	defer srv.Close()

	res, err := testClient(t, srv).Warehouse(context.Background(), "010807")
	if err != nil {
		t.Fatal(err)
	}
	if res.WarehouseInventory != 100 {
		t.Fatalf("want 100, got %d", res.WarehouseInventory)
	}
}

const arcgisJSON = `{"features":[
  {"attributes":{"LandmkName":"ABC Store 088","Address":"1 Main St","City":"Richmond","State":"VA","Zip":"23220","Phone":"804-1","URL":"/stores/88"},"geometry":{"x":-77.5,"y":37.5}},
  {"attributes":{"LandmkName":"ABC Store 219","Zip":"22182"},"geometry":{"x":-77.236,"y":38.915}}
]}`

func TestStoresAndStoreNear(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "arcgis") {
			_, _ = w.Write([]byte(arcgisJSON))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()
	c := testClient(t, srv)

	stores, err := c.Stores(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(stores) != 2 || stores[0].StoreNumber != 88 {
		t.Fatalf("stores not parsed: %+v", stores)
	}
	if stores[0].Lat != 37.5 || stores[0].Lng != -77.5 {
		t.Fatalf("geometry not mapped: %+v", stores[0])
	}

	near, err := c.StoreNear(context.Background(), 38.915, -77.236, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(near) != 1 || near[0].StoreNumber != 219 {
		t.Fatalf("nearest should be 219, got %+v", near)
	}
	if near[0].Distance == nil || *near[0].Distance > 1 {
		t.Fatalf("nearest distance should be ~0, got %v", near[0].Distance)
	}
}

func TestLimitedAvailability(t *testing.T) {
	body := `{}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()
	c := testClient(t, srv)

	res, err := c.LimitedAvailability(context.Background(), "010807")
	if err != nil {
		t.Fatal(err)
	}
	if res.Active || len(res.EventLinks) != 0 {
		t.Fatalf("empty {} should be inactive, got %+v", res)
	}

	body = `[{"title":"Allocated Drop","url":"/event/1"}]`
	res, err = c.LimitedAvailability(context.Background(), "010807")
	if err != nil {
		t.Fatal(err)
	}
	if !res.Active || len(res.EventLinks) != 1 || res.EventLinks[0].Title != "Allocated Drop" {
		t.Fatalf("populated events not extracted: %+v", res)
	}
}

func TestRateLimitTripsCircuitBreaker(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	// Shared throttle state across two client instances (simulates fresh processes).
	state := filepath.Join(t.TempDir(), "throttle.json")
	c1 := vabc.NewClient(vabc.WithBaseURL(srv.URL), vabc.WithMinInterval(0), vabc.WithStatePath(state))
	if _, err := c1.Warehouse(context.Background(), "010807"); !isRate(err) {
		t.Fatalf("first call should be rate-limited, got %v", err)
	}
	// Second "process" should fail fast from the open breaker, NOT call the server.
	c2 := vabc.NewClient(vabc.WithBaseURL(srv.URL), vabc.WithMinInterval(0), vabc.WithStatePath(state))
	if _, err := c2.Warehouse(context.Background(), "010807"); !isRate(err) {
		t.Fatalf("second call should fail fast as rate-limited, got %v", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("breaker should have prevented a 2nd server call, got %d calls", calls.Load())
	}
}

func isRate(err error) bool {
	var ae *vabc.APIError
	return errors.As(err, &ae) && ae.Kind == vabc.KindRateLimited
}

func TestFetchLatestPriceList(t *testing.T) {
	page := `<html><body>
	  <a href="/library/products/pdfs/quarterly-price-list-april--june-2026.pdf?rev=1">PDF</a>
	  <a href="/library/products/other-documents/quarterly-price-list-april--june-2026.xlsx?rev=abc&amp;hash=XYZ">XLSX</a>
	  <a href="/library/products/other-documents/monthly-specials-june-2026.xlsx">specials</a>
	</body></html>`
	var gotXLSX string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "product-downloads"):
			_, _ = w.Write([]byte(page))
		case strings.HasSuffix(r.URL.Path, ".xlsx"):
			gotXLSX = r.URL.Path
			_, _ = w.Write([]byte("PK\x03\x04 fake xlsx bytes"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	data, src, err := vabc.FetchLatestPriceList(context.Background(),
		vabc.WithBaseURL(srv.URL), vabc.WithMinInterval(0),
		vabc.WithStatePath(filepath.Join(t.TempDir(), "t.json")))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(src, "quarterly-price-list-april--june-2026.xlsx") {
		t.Fatalf("picked wrong link: %s", src)
	}
	if !strings.HasSuffix(gotXLSX, "quarterly-price-list-april--june-2026.xlsx") {
		t.Fatalf("downloaded wrong file: %s", gotXLSX)
	}
	if !strings.HasPrefix(string(data), "PK") {
		t.Fatalf("did not return xlsx bytes: %q", string(data))
	}
}

func TestSearchProducts(t *testing.T) {
	coveo := `{"totalCount":2,"results":[
	  {"clickUri":"https://x/products/rum/planteray-oftd","raw":{
	    "z95xproductz32xskuz32xids":["953714"],
	    "productz32xlabelz32xname":"Planteray O.f.t.d Overproof Rum",
	    "hierarchyz32xcategory":"Rum","hierarchyz32xtype":"[Dark]",
	    "z95xproductz32xsiz122xes":"1 L","z95xproductz32xpricez32xsort":32.99,
	    "proofmin":138,"z95xproductz32xlimitedz32xavailability":0,"z95xnewz32xproduct":1}},
	  {"clickUri":"https://x/page","raw":{"ftitle79429":"Some FAQ page"}}
	]}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !strings.Contains(r.URL.Path, "/coveo/rest/search") {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(coveo))
	}))
	defer srv.Close()

	products, err := testClient(t, srv).SearchProducts(context.Background(), "oftd", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(products) != 1 { // the non-product (no SKU) result is dropped
		t.Fatalf("want 1 product, got %d: %+v", len(products), products)
	}
	p := products[0]
	if p.ProductCode != "953714" || !strings.Contains(p.Name, "Overproof") {
		t.Fatalf("bad mapping: %+v", p)
	}
	if p.Category != "Rum" || p.Size != "1 L" {
		t.Fatalf("category/size mismapped: %+v", p)
	}
	if p.RetailPrice == nil || *p.RetailPrice != 32.99 {
		t.Fatalf("price not mapped: %+v", p.RetailPrice)
	}
	if p.Proof == nil || *p.Proof != 138 || !p.New {
		t.Fatalf("proof/new not mapped: %+v", p)
	}
}
