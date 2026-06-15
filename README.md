# Mudrex Go SDK

Unofficial Go SDK for the [Mudrex Trading API](https://docs.trade.mudrex.com). It supports the **Futures Trading API** (orders, positions, leverage, wallet, etc.) via a flat `TradeClient`.

**Built and maintained by [DecentralizedJM](https://github.com/DecentralizedJM)**

## Installation

Requires Go 1.21 or higher:

```bash
go get github.com/DecentralizedJM/mudrex-api-trading-go-sdk
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
| `PlaceOrder(PlaceOrderRequest)` | Place a new order |
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

## Rate Limits

The Mudrex API enforces rate limits (2 req/s, 50/min, 1000/hr, 10000/day). This SDK does **not** throttle requests. If you exceed the limit, the API returns 429 and the SDK returns `MudrexAPIError`.

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
