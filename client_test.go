package mudrex

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

type recordedRequest struct {
	method string
	path   string
	query  url.Values
	body   map[string]any
}

type testServer struct {
	server  *httptest.Server
	request *recordedRequest
}

func newTestServer(t *testing.T) *testServer {
	t.Helper()
	ts := &testServer{}
	ts.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := map[string]any{}
		if r.Body != nil {
			raw, _ := io.ReadAll(r.Body)
			if len(raw) > 0 {
				_ = json.Unmarshal(raw, &body)
			}
		}
		ts.request = &recordedRequest{
			method: r.Method,
			path:   r.URL.Path,
			query:  r.URL.Query(),
			body:   body,
		}
		writeJSON(w, map[string]any{"success": true, "data": responseDataForPath(r.URL.Path)})
	}))
	t.Cleanup(ts.server.Close)
	return ts
}

func newTestClient(t *testing.T, ts *testServer) *TradeClient {
	t.Helper()
	client, err := NewClientWithOptions(ClientOptions{
		APISecret: "test_secret",
		BaseURL:   ts.server.URL,
		SkipPing:  true,
	})
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func strPtr(v string) *string { return &v }
func int64Ptr(v int64) *int64 { return &v }
func boolPtr(v bool) *bool    { return &v }

func TestClientInitDefaultTradeCurrency(t *testing.T) {
	ts := newTestServer(t)
	client, err := NewClientWithOptions(ClientOptions{
		APISecret: "s",
		BaseURL:   ts.server.URL,
		SkipPing:  true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if client.tradeCurrency != "USDT" {
		t.Fatalf("got %q", client.tradeCurrency)
	}
}

func TestClientInitNonUSDTTradeCurrencyRaises(t *testing.T) {
	_, err := NewClientWithOptions(ClientOptions{APISecret: "s", TradeCurrency: "INR", SkipPing: true})
	if err == nil || !strings.Contains(err.Error(), "Only USDT") {
		t.Fatalf("expected USDT error, got %v", err)
	}
}

func TestClientInitPingCalled(t *testing.T) {
	pinged := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/fapi/v1/futures/ping" {
			pinged = true
		}
		writeJSON(w, map[string]any{"code": 200})
	}))
	t.Cleanup(server.Close)

	_, err := NewClientWithOptions(ClientOptions{APISecret: "s", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	if !pinged {
		t.Fatal("expected ping on init")
	}
}

func TestClientPingAnytime(t *testing.T) {
	pings := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/fapi/v1/futures/ping" {
			pings++
		}
		writeJSON(w, map[string]any{"code": 200})
	}))
	t.Cleanup(server.Close)

	client, err := NewClientWithOptions(ClientOptions{APISecret: "s", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	pingsAtInit := pings
	if err := client.Ping(); err != nil {
		t.Fatal(err)
	}
	if pings != pingsAtInit+1 {
		t.Fatalf("expected one additional ping, got %d total", pings)
	}
}

func TestResolveAssetSymbol(t *testing.T) {
	ref, err := resolveAsset("BTCUSDT", "")
	if err != nil {
		t.Fatal(err)
	}
	if ref.identifier != "BTCUSDT" || ref.params["is_symbol"] != "true" {
		t.Fatalf("unexpected ref: %+v", ref)
	}
}

func TestResolveAssetID(t *testing.T) {
	ref, err := resolveAsset("", "uuid-123")
	if err != nil {
		t.Fatal(err)
	}
	if ref.identifier != "uuid-123" || ref.params != nil {
		t.Fatalf("unexpected ref: %+v", ref)
	}
}

func TestResolveAssetBothRaises(t *testing.T) {
	_, err := resolveAsset("BTC", "uuid")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestResolveAssetNeitherRaises(t *testing.T) {
	_, err := resolveAsset("", "")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestListFuturesDefaultParams(t *testing.T) {
	ts := newTestServer(t)
	client := newTestClient(t, ts)
	if _, err := client.ListFutures(0, 0, "", ""); err != nil {
		t.Fatal(err)
	}
	if ts.request.method != http.MethodGet || ts.request.path != "/fapi/v1/futures" {
		t.Fatalf("unexpected request: %+v", ts.request)
	}
	if ts.request.query.Get("limit") != "10" || ts.request.query.Get("offset") != "0" {
		t.Fatalf("query=%v", ts.request.query)
	}
}

func TestListFuturesCustomParams(t *testing.T) {
	ts := newTestServer(t)
	client := newTestClient(t, ts)
	if _, err := client.ListFutures(5, 10, "volume", "desc"); err != nil {
		t.Fatal(err)
	}
	if ts.request.query.Get("limit") != "5" || ts.request.query.Get("offset") != "10" ||
		ts.request.query.Get("sort") != "volume" || ts.request.query.Get("order") != "desc" {
		t.Fatalf("query=%v", ts.request.query)
	}
}

func TestGetFutureBySymbol(t *testing.T) {
	ts := newTestServer(t)
	client := newTestClient(t, ts)
	if _, err := client.GetFuture("BTCUSDT", ""); err != nil {
		t.Fatal(err)
	}
	if ts.request.path != "/fapi/v1/futures/BTCUSDT" || ts.request.query.Get("is_symbol") != "true" {
		t.Fatalf("request=%+v", ts.request)
	}
}

func TestGetFutureByAssetID(t *testing.T) {
	ts := newTestServer(t)
	client := newTestClient(t, ts)
	if _, err := client.GetFuture("", "uuid-abc"); err != nil {
		t.Fatal(err)
	}
	if ts.request.path != "/fapi/v1/futures/uuid-abc" || len(ts.request.query) != 0 {
		t.Fatalf("request=%+v", ts.request)
	}
}

func TestGetAvailableFundsNoSource(t *testing.T) {
	ts := newTestServer(t)
	client := newTestClient(t, ts)
	if _, err := client.GetAvailableFunds(""); err != nil {
		t.Fatal(err)
	}
	if ts.request.query.Get("trade_currency") != "USDT" || ts.request.query.Has("source") {
		t.Fatalf("query=%v", ts.request.query)
	}
}

func TestGetAvailableFundsWithSource(t *testing.T) {
	ts := newTestServer(t)
	client := newTestClient(t, ts)
	if _, err := client.GetAvailableFunds("transfer"); err != nil {
		t.Fatal(err)
	}
	if ts.request.query.Get("source") != "transfer" || ts.request.query.Get("trade_currency") != "USDT" {
		t.Fatalf("query=%v", ts.request.query)
	}
}

func TestGetLeverageBySymbol(t *testing.T) {
	ts := newTestServer(t)
	client := newTestClient(t, ts)
	if _, err := client.GetLeverage("ETHUSDT", ""); err != nil {
		t.Fatal(err)
	}
	if ts.request.path != "/fapi/v1/futures/ETHUSDT/leverage" ||
		ts.request.query.Get("is_symbol") != "true" ||
		ts.request.query.Get("trade_currency") != "USDT" {
		t.Fatalf("request=%+v", ts.request)
	}
}

func TestSetLeverageBasic(t *testing.T) {
	ts := newTestServer(t)
	client := newTestClient(t, ts)
	if _, err := client.SetLeverage("BTCUSDT", "", "20", ""); err != nil {
		t.Fatal(err)
	}
	if ts.request.method != http.MethodPost || ts.request.path != "/fapi/v1/futures/BTCUSDT/leverage" {
		t.Fatalf("request=%+v", ts.request)
	}
	if ts.request.query.Get("is_symbol") != "true" ||
		ts.request.body["leverage"] != "20" ||
		ts.request.body["margin_type"] != "ISOLATED" ||
		ts.request.body["trade_currency"] != "USDT" {
		t.Fatalf("request=%+v", ts.request)
	}
}

func TestSetLeverageCustomMarginType(t *testing.T) {
	ts := newTestServer(t)
	client := newTestClient(t, ts)
	if _, err := client.SetLeverage("BTCUSDT", "", "5", "CROSS"); err != nil {
		t.Fatal(err)
	}
	if ts.request.body["margin_type"] != "CROSS" {
		t.Fatalf("body=%v", ts.request.body)
	}
}

func TestPlaceOrderMarket(t *testing.T) {
	ts := newTestServer(t)
	client := newTestClient(t, ts)
	_, err := client.PlaceOrder(PlaceOrderRequest{
		Symbol:      "BTCUSDT",
		Leverage:    "10",
		Quantity:    "0.001",
		OrderType:   "LONG",
		TriggerType: "MARKET",
	})
	if err != nil {
		t.Fatal(err)
	}
	if ts.request.method != http.MethodPost || ts.request.path != "/fapi/v1/futures/BTCUSDT/order" {
		t.Fatalf("request=%+v", ts.request)
	}
	if ts.request.query.Get("is_symbol") != "true" ||
		ts.request.body["leverage"] != "10" ||
		ts.request.body["quantity"] != "0.001" ||
		ts.request.body["order_type"] != "LONG" ||
		ts.request.body["trigger_type"] != "MARKET" ||
		ts.request.body["is_stoploss"] != false ||
		ts.request.body["is_takeprofit"] != false ||
		ts.request.body["reduce_only"] != false ||
		ts.request.body["trade_currency"] != "USDT" {
		t.Fatalf("request=%+v", ts.request)
	}
}

func TestPlaceOrderLimitWithSLTP(t *testing.T) {
	ts := newTestServer(t)
	client := newTestClient(t, ts)
	price := "3000"
	sl := "3100"
	tp := "2800"
	_, err := client.PlaceOrder(PlaceOrderRequest{
		Symbol:          "ETHUSDT",
		Leverage:        "5",
		Quantity:        "0.1",
		OrderType:       "SHORT",
		TriggerType:     "LIMIT",
		OrderPrice:      &price,
		IsStoploss:      true,
		StoplossPrice:   &sl,
		IsTakeprofit:    true,
		TakeprofitPrice: &tp,
	})
	if err != nil {
		t.Fatal(err)
	}
	if ts.request.body["order_price"] != "3000" ||
		ts.request.body["is_stoploss"] != true ||
		ts.request.body["stoploss_price"] != "3100" ||
		ts.request.body["is_takeprofit"] != true ||
		ts.request.body["takeprofit_price"] != "2800" {
		t.Fatalf("body=%v", ts.request.body)
	}
}

func TestPlaceOrderByAssetID(t *testing.T) {
	ts := newTestServer(t)
	client := newTestClient(t, ts)
	_, err := client.PlaceOrder(PlaceOrderRequest{
		AssetID:     "uuid-btc",
		Leverage:    "10",
		Quantity:    "0.001",
		OrderType:   "LONG",
		TriggerType: "MARKET",
	})
	if err != nil {
		t.Fatal(err)
	}
	if ts.request.path != "/fapi/v1/futures/uuid-btc/order" || len(ts.request.query) != 0 {
		t.Fatalf("request=%+v", ts.request)
	}
}

func TestPlaceOrderReduceOnly(t *testing.T) {
	ts := newTestServer(t)
	client := newTestClient(t, ts)
	_, err := client.PlaceOrder(PlaceOrderRequest{
		Symbol:      "BTCUSDT",
		Leverage:    "10",
		Quantity:    "0.001",
		OrderType:   "SHORT",
		TriggerType: "MARKET",
		ReduceOnly:  true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if ts.request.body["reduce_only"] != true {
		t.Fatalf("body=%v", ts.request.body)
	}
}

func TestGetOrdersDefaults(t *testing.T) {
	ts := newTestServer(t)
	client := newTestClient(t, ts)
	if _, err := client.GetOrders(0, nil); err != nil {
		t.Fatal(err)
	}
	if ts.request.path != "/fapi/v1/futures/orders" ||
		ts.request.query.Get("limit") != "20" ||
		ts.request.query.Get("trade_currency") != "USDT" ||
		ts.request.query.Has("offset") {
		t.Fatalf("request=%+v", ts.request)
	}
}

func TestGetOrdersWithOffset(t *testing.T) {
	ts := newTestServer(t)
	client := newTestClient(t, ts)
	offset := int64(1234567890)
	if _, err := client.GetOrders(5, &offset); err != nil {
		t.Fatal(err)
	}
	if ts.request.query.Get("limit") != "5" || ts.request.query.Get("offset") != "1234567890" {
		t.Fatalf("query=%v", ts.request.query)
	}
}

func TestGetOrderBasic(t *testing.T) {
	ts := newTestServer(t)
	client := newTestClient(t, ts)
	if _, err := client.GetOrder("order-uuid-123"); err != nil {
		t.Fatal(err)
	}
	if ts.request.method != http.MethodGet || ts.request.path != "/fapi/v1/futures/orders/order-uuid-123" {
		t.Fatalf("request=%+v", ts.request)
	}
}

func TestGetOrderHistoryDefaults(t *testing.T) {
	ts := newTestServer(t)
	client := newTestClient(t, ts)
	if _, err := client.GetOrderHistory(0, nil); err != nil {
		t.Fatal(err)
	}
	if ts.request.path != "/fapi/v1/futures/orders/history" {
		t.Fatalf("request=%+v", ts.request)
	}
}

func TestAmendOrderPrice(t *testing.T) {
	ts := newTestServer(t)
	client := newTestClient(t, ts)
	price := "50000"
	if _, err := client.AmendOrder(AmendOrderRequest{OrderID: "oid-1", OrderPrice: &price}); err != nil {
		t.Fatal(err)
	}
	if ts.request.method != http.MethodPatch || ts.request.path != "/fapi/v1/futures/orders/oid-1" {
		t.Fatalf("request=%+v", ts.request)
	}
	if ts.request.body["order_price"] != "50000" {
		t.Fatalf("body=%v", ts.request.body)
	}
}

func TestAmendOrderWithSLTPIDs(t *testing.T) {
	ts := newTestServer(t)
	client := newTestClient(t, ts)
	price := "50000"
	sl := "48000"
	tp := "55000"
	slID := "sl-uuid"
	tpID := "tp-uuid"
	if _, err := client.AmendOrder(AmendOrderRequest{
		OrderID:           "oid-1",
		OrderPrice:        &price,
		IsStoploss:        boolPtr(true),
		StoplossPrice:     &sl,
		StoplossOrderID:   &slID,
		IsTakeprofit:      boolPtr(true),
		TakeprofitPrice:   &tp,
		TakeprofitOrderID: &tpID,
	}); err != nil {
		t.Fatal(err)
	}
	if ts.request.body["stoploss_order_id"] != "sl-uuid" ||
		ts.request.body["takeprofit_order_id"] != "tp-uuid" ||
		ts.request.body["is_stoploss"] != true ||
		ts.request.body["is_takeprofit"] != true {
		t.Fatalf("body=%v", ts.request.body)
	}
}

func TestAmendOrderNoneFieldsStripped(t *testing.T) {
	ts := newTestServer(t)
	client := newTestClient(t, ts)
	price := "50000"
	if _, err := client.AmendOrder(AmendOrderRequest{OrderID: "oid-1", OrderPrice: &price}); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"stoploss_price", "takeprofit_price", "stoploss_order_id"} {
		if _, ok := ts.request.body[key]; ok {
			t.Fatalf("unexpected field %s in body", key)
		}
	}
}

func TestCancelOrderBasic(t *testing.T) {
	ts := newTestServer(t)
	client := newTestClient(t, ts)
	if _, err := client.CancelOrder("oid-1"); err != nil {
		t.Fatal(err)
	}
	if ts.request.method != http.MethodDelete || ts.request.path != "/fapi/v1/futures/orders/oid-1" {
		t.Fatalf("request=%+v", ts.request)
	}
}

func TestGetPositionsDefaults(t *testing.T) {
	ts := newTestServer(t)
	client := newTestClient(t, ts)
	if _, err := client.GetPositions(0, nil); err != nil {
		t.Fatal(err)
	}
	if ts.request.path != "/fapi/v1/futures/positions" {
		t.Fatalf("request=%+v", ts.request)
	}
}

func TestGetPositionHistoryDefaults(t *testing.T) {
	ts := newTestServer(t)
	client := newTestClient(t, ts)
	if _, err := client.GetPositionHistory(0, nil); err != nil {
		t.Fatal(err)
	}
	if ts.request.path != "/fapi/v1/futures/positions/history" {
		t.Fatalf("request=%+v", ts.request)
	}
}

func TestClosePositionBasic(t *testing.T) {
	ts := newTestServer(t)
	client := newTestClient(t, ts)
	if _, err := client.ClosePosition("pid-1"); err != nil {
		t.Fatal(err)
	}
	if ts.request.method != http.MethodPost || ts.request.path != "/fapi/v1/futures/positions/pid-1/close" {
		t.Fatalf("request=%+v", ts.request)
	}
}

func TestClosePositionPartialMarket(t *testing.T) {
	ts := newTestServer(t)
	client := newTestClient(t, ts)
	if _, err := client.ClosePositionPartial("pid-1", "0.001", "SHORT", nil); err != nil {
		t.Fatal(err)
	}
	if ts.request.path != "/fapi/v1/futures/positions/pid-1/close/partial" ||
		ts.request.body["quantity"] != "0.001" ||
		ts.request.body["order_type"] != "SHORT" {
		t.Fatalf("request=%+v", ts.request)
	}
}

func TestClosePositionPartialLimit(t *testing.T) {
	ts := newTestServer(t)
	client := newTestClient(t, ts)
	limit := "70000"
	if _, err := client.ClosePositionPartial("pid-1", "0.001", "SHORT", &limit); err != nil {
		t.Fatal(err)
	}
	if ts.request.body["limit_price"] != "70000" {
		t.Fatalf("body=%v", ts.request.body)
	}
}

func TestReversePositionBasic(t *testing.T) {
	ts := newTestServer(t)
	client := newTestClient(t, ts)
	if _, err := client.ReversePosition("pid-1"); err != nil {
		t.Fatal(err)
	}
	if ts.request.path != "/fapi/v1/futures/positions/pid-1/reverse" {
		t.Fatalf("request=%+v", ts.request)
	}
}

func TestPlaceRiskOrderStoplossOnly(t *testing.T) {
	ts := newTestServer(t)
	client := newTestClient(t, ts)
	sl := "65000"
	if _, err := client.PlaceRiskOrder("pid-1", true, false, &sl, nil); err != nil {
		t.Fatal(err)
	}
	if ts.request.path != "/fapi/v1/futures/positions/pid-1/riskorder" ||
		ts.request.body["is_stoploss"] != true ||
		ts.request.body["is_takeprofit"] != false ||
		ts.request.body["stoploss_price"] != "65000" {
		t.Fatalf("request=%+v", ts.request)
	}
}

func TestAmendRiskOrderStoploss(t *testing.T) {
	ts := newTestServer(t)
	client := newTestClient(t, ts)
	sl := "64000"
	slID := "sl-uuid"
	if _, err := client.AmendRiskOrder(AmendRiskOrderRequest{
		PositionID:      "pid-1",
		IsStoploss:      boolPtr(true),
		StoplossOrderID: &slID,
		StoplossPrice:   &sl,
	}); err != nil {
		t.Fatal(err)
	}
	if ts.request.body["is_stoploss"] != true ||
		ts.request.body["stoploss_order_id"] != "sl-uuid" ||
		ts.request.body["stoploss_price"] != "64000" {
		t.Fatalf("body=%v", ts.request.body)
	}
}

func TestAddMarginBasic(t *testing.T) {
	ts := newTestServer(t)
	client := newTestClient(t, ts)
	if _, err := client.AddMargin("pid-1", "50"); err != nil {
		t.Fatal(err)
	}
	if ts.request.path != "/fapi/v1/futures/positions/pid-1/add-margin" ||
		ts.request.body["margin"] != "50" {
		t.Fatalf("request=%+v", ts.request)
	}
}

func TestGetLiquidationPriceBasic(t *testing.T) {
	ts := newTestServer(t)
	ts.server.Close()
	ts.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ts.request = &recordedRequest{
			method: r.Method,
			path:   r.URL.Path,
			query:  r.URL.Query(),
		}
		writeJSON(w, map[string]any{"success": true, "data": "62888.3"})
	}))
	t.Cleanup(ts.server.Close)

	client := newTestClient(t, ts)
	resp, err := client.GetLiquidationPrice("pid-1", nil)
	if err != nil {
		t.Fatal(err)
	}
	if ts.request.path != "/fapi/v1/futures/positions/pid-1/liq-price" ||
		ts.request.query.Get("trade_currency") != "USDT" {
		t.Fatalf("request=%+v", ts.request)
	}
	raw, ok := resp.Result()
	if !ok || string(raw) != `"62888.3"` {
		t.Fatalf("result=%s ok=%v", raw, ok)
	}
}

func TestGetLiquidationPriceWithExtMargin(t *testing.T) {
	ts := newTestServer(t)
	client := newTestClient(t, ts)
	margin := "10"
	if _, err := client.GetLiquidationPrice("pid-1", &margin); err != nil {
		t.Fatal(err)
	}
	if ts.request.query.Get("ext_margin") != "10" || ts.request.query.Get("trade_currency") != "USDT" {
		t.Fatalf("query=%v", ts.request.query)
	}
}

func TestGetFeeHistoryDefaults(t *testing.T) {
	ts := newTestServer(t)
	client := newTestClient(t, ts)
	if _, err := client.GetFeeHistory(0, nil); err != nil {
		t.Fatal(err)
	}
	if ts.request.path != "/fapi/v1/futures/fee/history" ||
		ts.request.query.Get("limit") != "10" {
		t.Fatalf("request=%+v", ts.request)
	}
}

func TestGetWalletFundsBasic(t *testing.T) {
	ts := newTestServer(t)
	client := newTestClient(t, ts)
	if _, err := client.GetWalletFunds(); err != nil {
		t.Fatal(err)
	}
	if ts.request.method != http.MethodGet || ts.request.path != "/fapi/v1/wallet/funds" {
		t.Fatalf("request=%+v", ts.request)
	}
}

func TestTransferSpotToFutures(t *testing.T) {
	ts := newTestServer(t)
	client := newTestClient(t, ts)
	if _, err := client.Transfer("SPOT", "FUTURES", "100"); err != nil {
		t.Fatal(err)
	}
	if ts.request.path != "/fapi/v1/wallet/futures/transfer" ||
		ts.request.body["from_wallet_type"] != "SPOT" ||
		ts.request.body["to_wallet_type"] != "FUTURES" ||
		ts.request.body["amount"] != "100" {
		t.Fatalf("request=%+v", ts.request)
	}
}

func TestTradeCurrencySentOnEndpoints(t *testing.T) {
	ts := newTestServer(t)
	client := newTestClient(t, ts)

	calls := []func() error{
		func() error { _, err := client.GetAvailableFunds(""); return err },
		func() error { _, err := client.GetLeverage("BTCUSDT", ""); return err },
		func() error { _, err := client.SetLeverage("BTCUSDT", "", "10", ""); return err },
		func() error {
			_, err := client.PlaceOrder(PlaceOrderRequest{
				Symbol: "BTCUSDT", Leverage: "10", Quantity: "0.001",
				OrderType: "LONG", TriggerType: "MARKET",
			})
			return err
		},
		func() error { _, err := client.GetOrders(0, nil); return err },
		func() error { _, err := client.GetOrderHistory(0, nil); return err },
		func() error { _, err := client.GetPositions(0, nil); return err },
		func() error { _, err := client.GetPositionHistory(0, nil); return err },
		func() error { _, err := client.GetFeeHistory(0, nil); return err },
		func() error { _, err := client.GetLiquidationPrice("pid", nil); return err },
	}

	for _, call := range calls {
		if err := call(); err != nil {
			t.Fatal(err)
		}
		sent := ts.request.query.Get("trade_currency")
		if sent == "" {
			sent, _ = ts.request.body["trade_currency"].(string)
		}
		if sent != "USDT" {
			t.Fatalf("expected USDT on %s, got %q", ts.request.path, sent)
		}
	}
}
