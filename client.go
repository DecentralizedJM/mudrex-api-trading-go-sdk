package mudrex

import (
	"fmt"
	"net/http"
	"time"
)

// ClientOptions configures a TradeClient.
type ClientOptions struct {
	APISecret     string
	TradeCurrency string
	Timeout       time.Duration
	MaxRetries    int
	LogRequests   bool
	BaseURL       string
	HTTPClient    *http.Client
	SkipPing      bool
}

// TradeClient is the Mudrex Futures Trading API client.
type TradeClient struct {
	*httpClient
	tradeCurrency string
}

// NewClient creates a TradeClient with default options.
func NewClient(apiSecret string) (*TradeClient, error) {
	return NewClientWithOptions(ClientOptions{APISecret: apiSecret})
}

// NewClientWithOptions creates a TradeClient with custom configuration.
func NewClientWithOptions(opts ClientOptions) (*TradeClient, error) {
	tradeCurrency := opts.TradeCurrency
	if tradeCurrency == "" {
		tradeCurrency = "USDT"
	}
	if tradeCurrency != "USDT" {
		return nil, fmt.Errorf("Only USDT is supported as trade currency. Use TradeCurrency=\"USDT\" (default)")
	}

	httpClient, err := newHTTPClient(clientConfig{
		apiSecret:   opts.APISecret,
		tradeCurr:   tradeCurrency,
		timeout:     opts.Timeout,
		maxRetries:  opts.MaxRetries,
		logRequests: opts.LogRequests,
		baseURL:     opts.BaseURL,
		httpClient:  opts.HTTPClient,
		skipPing:    opts.SkipPing,
	})
	if err != nil {
		return nil, err
	}

	return &TradeClient{
		httpClient:    httpClient,
		tradeCurrency: tradeCurrency,
	}, nil
}

// Ping verifies connectivity and authentication.
func (c *TradeClient) Ping() error {
	return c.ping()
}

type assetRef struct {
	identifier string
	params     map[string]any
}

func resolveAsset(symbol, assetID string) (assetRef, error) {
	if symbol != "" && assetID != "" {
		return assetRef{}, fmt.Errorf("provide either symbol or assetID, not both")
	}
	if symbol == "" && assetID == "" {
		return assetRef{}, fmt.Errorf("either symbol or assetID is required")
	}
	if symbol != "" {
		return assetRef{
			identifier: symbol,
			params:     map[string]any{"is_symbol": "true"},
		}, nil
	}
	return assetRef{identifier: assetID, params: nil}, nil
}

func asResponse(value any) (Response, error) {
	resp, ok := value.(Response)
	if !ok {
		return nil, fmt.Errorf("unexpected response type %T", value)
	}
	return resp, nil
}

func asResponseList(value any) ([]Response, error) {
	items, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("unexpected response type %T", value)
	}
	out := make([]Response, 0, len(items))
	for _, item := range items {
		resp, ok := item.(Response)
		if !ok {
			return nil, fmt.Errorf("unexpected list item type %T", item)
		}
		out = append(out, resp)
	}
	return out, nil
}

// ListFutures lists available futures contracts.
func (c *TradeClient) ListFutures(limit, offset int, sort, order string) ([]Response, error) {
	if limit == 0 {
		limit = 10
	}
	value, err := c.get("/futures", map[string]any{
		"limit":  limit,
		"offset": offset,
		"sort":   sort,
		"order":  order,
	})
	if err != nil {
		return nil, err
	}
	return asResponseList(value)
}

// GetFuture returns details for a single futures contract.
func (c *TradeClient) GetFuture(symbol, assetID string) (Response, error) {
	ref, err := resolveAsset(symbol, assetID)
	if err != nil {
		return nil, err
	}
	value, err := c.get("/futures/"+ref.identifier, ref.params)
	if err != nil {
		return nil, err
	}
	return asResponse(value)
}

// GetAvailableFunds returns available funds for futures trading.
func (c *TradeClient) GetAvailableFunds(source string) (Response, error) {
	value, err := c.get("/futures/funds", map[string]any{
		"source":          source,
		"trade_currency":  c.tradeCurrency,
	})
	if err != nil {
		return nil, err
	}
	return asResponse(value)
}

// GetLeverage returns leverage settings for an asset.
func (c *TradeClient) GetLeverage(symbol, assetID string) (Response, error) {
	ref, err := resolveAsset(symbol, assetID)
	if err != nil {
		return nil, err
	}
	params := ref.params
	if params == nil {
		params = map[string]any{}
	}
	params["trade_currency"] = c.tradeCurrency
	value, err := c.get("/futures/"+ref.identifier+"/leverage", params)
	if err != nil {
		return nil, err
	}
	return asResponse(value)
}

// SetLeverage sets leverage for an asset.
func (c *TradeClient) SetLeverage(symbol, assetID, leverage, marginType string) (Response, error) {
	ref, err := resolveAsset(symbol, assetID)
	if err != nil {
		return nil, err
	}
	if marginType == "" {
		marginType = "ISOLATED"
	}
	value, err := c.post("/futures/"+ref.identifier+"/leverage", ref.params, map[string]any{
		"leverage":       leverage,
		"margin_type":    marginType,
		"trade_currency": c.tradeCurrency,
	})
	if err != nil {
		return nil, err
	}
	return asResponse(value)
}

// PlaceOrderRequest holds order placement parameters.
type PlaceOrderRequest struct {
	Symbol           string
	AssetID          string
	Leverage         string
	Quantity         string
	OrderType        string
	TriggerType      string
	OrderPrice       *string
	IsStoploss       bool
	IsTakeprofit     bool
	StoplossPrice    *string
	TakeprofitPrice  *string
	ReduceOnly       bool
}

// PlaceOrder creates a new futures order.
func (c *TradeClient) PlaceOrder(req PlaceOrderRequest) (Response, error) {
	ref, err := resolveAsset(req.Symbol, req.AssetID)
	if err != nil {
		return nil, err
	}
	body := map[string]any{
		"leverage":       req.Leverage,
		"quantity":       req.Quantity,
		"order_type":     req.OrderType,
		"trigger_type":   req.TriggerType,
		"is_stoploss":    req.IsStoploss,
		"is_takeprofit":  req.IsTakeprofit,
		"reduce_only":    req.ReduceOnly,
		"trade_currency": c.tradeCurrency,
		"order_price":    req.OrderPrice,
		"stoploss_price": req.StoplossPrice,
		"takeprofit_price": req.TakeprofitPrice,
	}
	value, err := c.post("/futures/"+ref.identifier+"/order", ref.params, body)
	if err != nil {
		return nil, err
	}
	return asResponse(value)
}

// GetOrders returns open orders.
func (c *TradeClient) GetOrders(limit int, offset *int64) ([]Response, error) {
	if limit == 0 {
		limit = 20
	}
	params := map[string]any{
		"limit":          limit,
		"trade_currency": c.tradeCurrency,
	}
	if offset != nil {
		params["offset"] = *offset
	}
	value, err := c.get("/futures/orders", params)
	if err != nil {
		return nil, err
	}
	return asResponseList(value)
}

// GetOrder returns a single order by ID.
func (c *TradeClient) GetOrder(orderID string) (Response, error) {
	value, err := c.get("/futures/orders/"+orderID, nil)
	if err != nil {
		return nil, err
	}
	return asResponse(value)
}

// GetOrderHistory returns order history.
func (c *TradeClient) GetOrderHistory(limit int, offset *int64) ([]Response, error) {
	if limit == 0 {
		limit = 20
	}
	params := map[string]any{
		"limit":          limit,
		"trade_currency": c.tradeCurrency,
	}
	if offset != nil {
		params["offset"] = *offset
	}
	value, err := c.get("/futures/orders/history", params)
	if err != nil {
		return nil, err
	}
	return asResponseList(value)
}

// AmendOrderRequest holds order amendment parameters.
type AmendOrderRequest struct {
	OrderID            string
	OrderPrice         *string
	IsStoploss         *bool
	IsTakeprofit       *bool
	StoplossPrice      *string
	TakeprofitPrice    *string
	StoplossOrderID    *string
	TakeprofitOrderID  *string
}

// AmendOrder amends an existing limit order.
func (c *TradeClient) AmendOrder(req AmendOrderRequest) (Response, error) {
	value, err := c.patch("/futures/orders/"+req.OrderID, map[string]any{
		"order_price":          req.OrderPrice,
		"is_stoploss":          req.IsStoploss,
		"is_takeprofit":        req.IsTakeprofit,
		"stoploss_price":       req.StoplossPrice,
		"takeprofit_price":     req.TakeprofitPrice,
		"stoploss_order_id":    req.StoplossOrderID,
		"takeprofit_order_id":  req.TakeprofitOrderID,
	})
	if err != nil {
		return nil, err
	}
	return asResponse(value)
}

// CancelOrder cancels an open order.
func (c *TradeClient) CancelOrder(orderID string) (any, error) {
	return c.delete("/futures/orders/" + orderID)
}

// GetPositions returns open positions.
func (c *TradeClient) GetPositions(limit int, offset *int64) ([]Response, error) {
	if limit == 0 {
		limit = 20
	}
	params := map[string]any{
		"limit":          limit,
		"trade_currency": c.tradeCurrency,
	}
	if offset != nil {
		params["offset"] = *offset
	}
	value, err := c.get("/futures/positions", params)
	if err != nil {
		return nil, err
	}
	return asResponseList(value)
}

// GetPositionHistory returns closed position history.
func (c *TradeClient) GetPositionHistory(limit int, offset *int64) ([]Response, error) {
	if limit == 0 {
		limit = 20
	}
	params := map[string]any{
		"limit":          limit,
		"trade_currency": c.tradeCurrency,
	}
	if offset != nil {
		params["offset"] = *offset
	}
	value, err := c.get("/futures/positions/history", params)
	if err != nil {
		return nil, err
	}
	return asResponseList(value)
}

// ClosePosition closes an entire position.
func (c *TradeClient) ClosePosition(positionID string) (any, error) {
	return c.post("/futures/positions/"+positionID+"/close", nil, nil)
}

// ClosePositionPartial partially closes a position.
func (c *TradeClient) ClosePositionPartial(positionID, quantity, orderType string, limitPrice *string) (any, error) {
	return c.post("/futures/positions/"+positionID+"/close/partial", nil, map[string]any{
		"quantity":    quantity,
		"order_type":  orderType,
		"limit_price": limitPrice,
	})
}

// ReversePosition reverses a position.
func (c *TradeClient) ReversePosition(positionID string) (any, error) {
	return c.post("/futures/positions/"+positionID+"/reverse", nil, nil)
}

// PlaceRiskOrder places stop-loss and/or take-profit on a position.
func (c *TradeClient) PlaceRiskOrder(positionID string, isStoploss, isTakeprofit bool, stoplossPrice, takeprofitPrice *string) (Response, error) {
	value, err := c.post("/futures/positions/"+positionID+"/riskorder", nil, map[string]any{
		"is_stoploss":      isStoploss,
		"is_takeprofit":    isTakeprofit,
		"stoploss_price":   stoplossPrice,
		"takeprofit_price": takeprofitPrice,
	})
	if err != nil {
		return nil, err
	}
	return asResponse(value)
}

// AmendRiskOrderRequest holds risk order amendment parameters.
type AmendRiskOrderRequest struct {
	PositionID         string
	IsStoploss         *bool
	IsTakeprofit       *bool
	StoplossOrderID    *string
	TakeprofitOrderID  *string
	StoplossPrice      *string
	TakeprofitPrice    *string
}

// AmendRiskOrder amends stop-loss and/or take-profit on a position.
func (c *TradeClient) AmendRiskOrder(req AmendRiskOrderRequest) (Response, error) {
	value, err := c.patch("/futures/positions/"+req.PositionID+"/riskorder", map[string]any{
		"is_stoploss":          req.IsStoploss,
		"is_takeprofit":        req.IsTakeprofit,
		"stoploss_order_id":    req.StoplossOrderID,
		"takeprofit_order_id":  req.TakeprofitOrderID,
		"stoploss_price":       req.StoplossPrice,
		"takeprofit_price":     req.TakeprofitPrice,
	})
	if err != nil {
		return nil, err
	}
	return asResponse(value)
}

// AddMargin adds margin to an open position.
func (c *TradeClient) AddMargin(positionID, margin string) (Response, error) {
	value, err := c.post("/futures/positions/"+positionID+"/add-margin", nil, map[string]any{
		"margin": margin,
	})
	if err != nil {
		return nil, err
	}
	return asResponse(value)
}

// GetLiquidationPrice returns the liquidation price for a position.
func (c *TradeClient) GetLiquidationPrice(positionID string, extMargin *string) (Response, error) {
	value, err := c.get("/futures/positions/"+positionID+"/liq-price", map[string]any{
		"ext_margin":     extMargin,
		"trade_currency": c.tradeCurrency,
	})
	if err != nil {
		return nil, err
	}
	return asResponse(value)
}

// GetFeeHistory returns trading fee history.
func (c *TradeClient) GetFeeHistory(limit int, offset *int64) ([]Response, error) {
	if limit == 0 {
		limit = 10
	}
	params := map[string]any{
		"limit":          limit,
		"trade_currency": c.tradeCurrency,
	}
	if offset != nil {
		params["offset"] = *offset
	}
	value, err := c.get("/futures/fee/history", params)
	if err != nil {
		return nil, err
	}
	return asResponseList(value)
}

// GetWalletFunds returns spot wallet balances.
func (c *TradeClient) GetWalletFunds() (Response, error) {
	value, err := c.get("/wallet/funds", nil)
	if err != nil {
		return nil, err
	}
	return asResponse(value)
}

// Transfer moves funds between wallets.
func (c *TradeClient) Transfer(fromWallet, toWallet, amount string) (Response, error) {
	value, err := c.post("/wallet/futures/transfer", nil, map[string]any{
		"from_wallet_type": fromWallet,
		"to_wallet_type":   toWallet,
		"amount":           amount,
	})
	if err != nil {
		return nil, err
	}
	return asResponse(value)
}
