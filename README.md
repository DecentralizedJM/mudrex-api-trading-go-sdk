# Mudrex Go SDK

Unofficial Go SDK for the [Mudrex Trading API](https://docs.trade.mudrex.com). It supports the **Futures Trading API** (orders, positions, leverage, wallet, etc.) via a flat `TradeClient`.

**Repository:** [github.com/DecentralizedJM/mudrex-api-trading-go-sdk](https://github.com/DecentralizedJM/mudrex-api-trading-go-sdk)  
**Default branch:** `main` (this is the only branch — all development and releases happen here)  
**Built and maintained by [DecentralizedJM](https://github.com/DecentralizedJM)**

## Installation

Requires Go 1.21 or higher:

```bash
go get github.com/DecentralizedJM/mudrex-api-trading-go-sdk
```

Import the module in your code:

```go
import mudrex "github.com/DecentralizedJM/mudrex-api-trading-go-sdk"
```

## Quick Start

```go
package main

import (
	"fmt"
	"log"

	mudrex "github.com/DecentralizedJM/mudrex-api-trading-go-sdk"
)

func main() {
	client, err := mudrex.NewClient("your_api_secret")
	if err != nil {
		log.Fatal(err)
	}

	resp, err := client.PlaceOrder(mudrex.PlaceOrderRequest{
		Symbol:      "BTCUSDT",
		Leverage:    "10",
		Quantity:    "0.001",
		OrderType:   "LONG",
		TriggerType: "MARKET",
	})
	if err != nil {
		log.Fatal(err)
	}

	orderID, err := resp.GetString("order_id")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(orderID)
}
```

The client pings the API on creation — a bad secret or unreachable server returns an error immediately:

```go
_, err := mudrex.NewClient("wrong_secret")
var apiErr *mudrex.MudrexAPIError
if errors.As(err, &apiErr) {
	fmt.Println(apiErr) // [401] Invalid Authentication
}
```

You can also ping anytime to verify connectivity and credentials:

```go
if err := client.Ping(); err != nil {
	log.Fatal(err)
}
```

Or set the API secret via environment variable:

```bash
export MUDREX_API_SECRET="your_api_secret"
```

```go
client, err := mudrex.NewClientWithOptions(mudrex.ClientOptions{})
```

## Place Orders (from [Create new order](https://docs.trade.mudrex.com/docs/post-market-order))

The samples below mirror the official API documentation curl requests, converted to this SDK.  
Pass numeric fields as **strings** for precision (e.g. `"50"`, `"0.01"`).

### Sample 1 — Market order by asset ID (with SL/TP)

Official request:

```bash
curl -X POST "https://trade.mudrex.com/fapi/v1/futures/{asset_id}/order" \
  -H "Content-Type: application/json" \
  -H "X-Authentication: your-secret-key" \
  -d '{
    "leverage": 50,
    "quantity": 0.01,
    "order_price": 12445627,
    "order_type": "LONG",
    "trigger_type": "MARKET",
    "is_takeprofit": true,
    "is_stoploss": true,
    "stoploss_price": 3800,
    "takeprofit_price": 5000,
    "reduce_only": false
  }'
```

Go SDK equivalent:

```go
package main

import (
	"fmt"
	"log"

	mudrex "github.com/DecentralizedJM/mudrex-api-trading-go-sdk"
)

func str(v string) *string { return &v }

func main() {
	client, err := mudrex.NewClient("your-secret-key")
	if err != nil {
		log.Fatal(err)
	}

	resp, err := client.PlaceOrder(mudrex.PlaceOrderRequest{
		AssetID:         "your-asset-uuid", // path: /futures/{asset_id}/order
		Leverage:        "50",
		Quantity:        "0.01",
		OrderPrice:      str("12445627"),
		OrderType:       "LONG",
		TriggerType:     "MARKET",
		IsTakeprofit:    true,
		IsStoploss:      true,
		StoplossPrice:   str("3800"),
		TakeprofitPrice: str("5000"),
		ReduceOnly:      false,
	})
	if err != nil {
		log.Fatal(err)
	}

	orderID, _ := resp.GetString("order_id")
	status, _ := resp.GetString("status")
	fmt.Printf("order_id=%s status=%s\n", orderID, status)
}
```

Expected response fields (from the API docs): `order_id`, `leverage`, `amount`, `quantity`, `price`, `status`, `message`.

### Sample 2 — Symbol-first order (`BTCUSDT` + `is_symbol`)

Official request:

```bash
curl -X POST "https://trade.mudrex.com/fapi/v1/futures/BTCUSDT/order?is_symbol" \
  -H "Content-Type: application/json" \
  -H "X-Authentication: your-secret-key" \
  -d '{
    "leverage": 50,
    "quantity": 0.01,
    "order_price": 12445627,
    "order_type": "LONG",
    "trigger_type": "MARKET",
    "is_takeprofit": true,
    "is_stoploss": true,
    "stoploss_price": 3800,
    "takeprofit_price": 5000,
    "reduce_only": false
  }'
```

Go SDK equivalent (the SDK adds `?is_symbol=true` automatically when you pass `Symbol`):

```go
resp, err := client.PlaceOrder(mudrex.PlaceOrderRequest{
	Symbol:          "BTCUSDT",
	Leverage:        "50",
	Quantity:        "0.01",
	OrderPrice:      str("12445627"),
	OrderType:       "LONG",
	TriggerType:     "MARKET",
	IsTakeprofit:    true,
	IsStoploss:      true,
	StoplossPrice:   str("3800"),
	TakeprofitPrice: str("5000"),
	ReduceOnly:      false,
})
```

### Sample 3 — Simple market long (no SL/TP)

```go
resp, err := client.PlaceOrder(mudrex.PlaceOrderRequest{
	Symbol:      "BTCUSDT",
	Leverage:    "10",
	Quantity:    "0.001",
	OrderType:   "LONG",
	TriggerType: "MARKET",
})
```

### Sample 4 — Limit order

```go
resp, err := client.PlaceOrder(mudrex.PlaceOrderRequest{
	Symbol:       "ETHUSDT",
	Leverage:     "5",
	Quantity:     "0.1",
	OrderType:    "SHORT",
	TriggerType:  "LIMIT",
	OrderPrice:   str("3000"),
})
```

### Sample 5 — Reduce-only close order

If you have an open `LONG` position, use `SHORT` with `ReduceOnly: true`:

```go
resp, err := client.PlaceOrder(mudrex.PlaceOrderRequest{
	Symbol:      "BTCUSDT",
	Leverage:    "10",
	Quantity:    "0.001",
	OrderType:   "SHORT", // opposite of existing position
	TriggerType: "MARKET",
	ReduceOnly:  true,
})
```

### Runnable example

A copy-pasteable program lives at [`examples/place_order/main.go`](examples/place_order/main.go):

```bash
export MUDREX_API_SECRET="your-secret-key"
go run ./examples/place_order
```

### Order placement notes (from API docs)

| Topic | Detail |
|---|---|
| `order_type` | `LONG` or `SHORT` |
| `trigger_type` | `MARKET` or `LIMIT` |
| `order_price` | Required for limit-style pricing; must be within min/max for the asset |
| SL/TP | If `IsStoploss` / `IsTakeprofit` is `true`, the matching price field is required |
| `reduce_only` | When `true`, order type must be opposite of the open position |
| Symbol vs UUID | Pass `Symbol: "BTCUSDT"` (recommended) or `AssetID: "uuid-..."` |

## Configuration

```go
client, err := mudrex.NewClientWithOptions(mudrex.ClientOptions{
	APISecret:     "...",
	TradeCurrency: "USDT", // only USDT supported; this is the default
	Timeout:       10 * time.Second,
	MaxRetries:    3, // retries on network errors only
	LogRequests:   true,
})
```

**Numeric parameters** (quantity, leverage, prices, amount, margin, etc.) should be passed as **strings** (e.g. `"0.001"`, `"10"`) for precision — the API expects string numerics.

## API Reference

### Client

| Method | Description |
|---|---|
| `Ping()` | Verify connectivity and API secret; returns error on failure |

### Futures / Assets

| Method | Description |
|---|---|
| `ListFutures(limit, offset, sort, order)` | List available futures contracts |
| `GetFuture(symbol, assetID)` | Get a single futures contract |
| `GetAvailableFunds(source)` | Get available trading funds |

### Leverage

| Method | Description |
|---|---|
| `GetLeverage(symbol, assetID)` | Get current leverage and margin type |
| `SetLeverage(symbol, assetID, leverage, marginType)` | Set leverage for an asset |

### Orders

| Method | Description |
|---|---|
| `PlaceOrder(PlaceOrderRequest)` | Place a new order — see [Place Orders](#place-orders-from-create-new-order) |
| `GetOrders(limit, offset)` | Get open orders |
| `GetOrder(orderID)` | Get a single order |
| `GetOrderHistory(limit, offset)` | Get order history |
| `AmendOrder(AmendOrderRequest)` | Amend a limit order |
| `CancelOrder(orderID)` | Cancel an open order |

### Positions

| Method | Description |
|---|---|
| `GetPositions(limit, offset)` | Get open positions |
| `GetPositionHistory(limit, offset)` | Get position history |
| `ClosePosition(positionID)` | Close entire position |
| `ClosePositionPartial(positionID, quantity, orderType, limitPrice)` | Partially close |
| `ReversePosition(positionID)` | Reverse a position |
| `PlaceRiskOrder(...)` | Add SL/TP |
| `AmendRiskOrder(AmendRiskOrderRequest)` | Amend SL/TP |
| `AddMargin(positionID, margin)` | Add margin |
| `GetLiquidationPrice(positionID, extMargin)` | Get liquidation price |

### Fees

| Method | Description |
|---|---|
| `GetFeeHistory(limit, offset)` | Get trading fee history |

### Wallet

| Method | Description |
|---|---|
| `GetWalletFunds()` | Get wallet balances |
| `Transfer(fromWallet, toWallet, amount)` | Transfer between wallets |

## Using Symbols vs UUIDs

By default, asset-related methods use trading symbols (e.g. `"BTCUSDT"`). To use a raw asset UUID, pass it as `assetID` and leave `symbol` empty:

```go
client.GetLeverage("BTCUSDT", "")                     // by symbol (recommended)
client.GetLeverage("", "550e8400-e29b-41d4-716-446655440000") // by UUID
```

## Response Format

Methods return a `Response` map or a slice of them. Use `GetString`, `GetBool`, or `Get` to read fields:

```go
resp, _ := client.GetLeverage("BTCUSDT", "")
leverage, _ := resp.GetString("leverage")
marginType, _ := resp.GetString("margin_type")
```

List endpoints return `[]Response`:

```go
orders, _ := client.GetOrders(20, nil)
for _, order := range orders {
	id, _ := order.GetString("id")
	fmt.Println(id)
}
```

Scalar API values are wrapped in a `result` field:

```go
resp, _ := client.GetLiquidationPrice("pid-1", nil)
raw, _ := resp.Result()
```

**ID conventions:** `PlaceOrder` returns `order_id`; list endpoints use `id` on each item.

```go
resp, _ := client.PlaceOrder(...)
orderID, _ := resp.GetString("order_id")
client.CancelOrder(orderID)

orders, _ := client.GetOrders(20, nil)
for _, o := range orders {
	id, _ := o.GetString("id")
	client.CancelOrder(id)
}
```

## Error Handling

```go
_, err := client.PlaceOrder(mudrex.PlaceOrderRequest{...})
var apiErr *mudrex.MudrexAPIError
if errors.As(err, &apiErr) {
	fmt.Printf("API error [%d]: %s\n", apiErr.Code, apiErr.Message)
}

var reqErr *mudrex.MudrexRequestError
if errors.As(err, &reqErr) {
	fmt.Printf("Network error: %s\n", reqErr.Message)
}
```

Common order errors from the API (see [Create new order](https://docs.trade.mudrex.com/docs/post-market-order)):

| HTTP | Message |
|---|---|
| 400 | Params error |
| 400 | invalid trigger type |
| 400 | invalid order type |
| 400 | order price out of permissible range |
| 400 | quantity not a multiple of the quantity step |
| 400 | leverage out of permissible range |

## Rate Limits

Per the [Authentication & Rate Limits](https://docs.trade.mudrex.com/docs/authentication-rate-limits) docs, limits are enforced **per API key**. This SDK does **not** throttle requests. If you exceed a limit, the API returns 429 and the SDK returns `MudrexAPIError`.

### Default limits (most endpoints)

| Duration | Limit |
|---|---|
| Second | 10 |
| Minute | 500 |
| Hour | 30,000 |

### Wallet limits (`/fapi/v1/wallet/*`)

Spot wallet balance and spot↔futures transfer endpoints use **separate** limits:

| Duration | Limit |
|---|---|
| Second | 2 |
| Minute | 50 |
| Hour | 1,000 |

## Testing

```bash
go test ./...
```

## License

MIT — see [LICENSE](LICENSE) for details.

## Disclaimer

**This is an UNOFFICIAL SDK** for educational and informational purposes. Cryptocurrency trading involves significant risk. Always use proper risk management and test thoroughly before trading with real funds.

---

Built and maintained by DecentralizedJM
