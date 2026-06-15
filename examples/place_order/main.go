// Example: place a market order using the Mudrex Go SDK.
// Mirrors https://docs.trade.mudrex.com/docs/post-market-order
//
// Run:
//
//	export MUDREX_API_SECRET="your-secret-key"
//	go run ./examples/place_order
package main

import (
	"fmt"
	"log"
	"os"

	mudrex "github.com/DecentralizedJM/mudrex-api-trading-go-sdk"
)

func str(v string) *string { return &v }

func main() {
	secret := os.Getenv("MUDREX_API_SECRET")
	if secret == "" {
		log.Fatal("set MUDREX_API_SECRET")
	}

	client, err := mudrex.NewClient(secret)
	if err != nil {
		log.Fatal(err)
	}

	// Symbol-first order (equivalent to POST /futures/BTCUSDT/order?is_symbol)
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
	})
	if err != nil {
		log.Fatal(err)
	}

	orderID, _ := resp.GetString("order_id")
	status, _ := resp.GetString("status")
	fmt.Printf("order_id=%s status=%s\n", orderID, status)
}
