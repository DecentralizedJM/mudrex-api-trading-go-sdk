package mudrex

import (
	"encoding/json"
	"time"
)

// Enums
type OrderType string
type TriggerType string
type MarginType string
type OrderStatus string
type PositionStatus string
type WalletType string

const (
	// Order types
	OrderTypeLong  OrderType = "LONG"
	OrderTypeShort OrderType = "SHORT"
	
	// Trigger types
	TriggerTypeMarket TriggerType = "MARKET"
	TriggerTypeLimit  TriggerType = "LIMIT"
	
	// Margin types
	MarginTypeIsolated MarginType = "ISOLATED"
	
	// Order statuses
	OrderStatusOpen             OrderStatus = "OPEN"
	OrderStatusFilled           OrderStatus = "FILLED"
	OrderStatusPartiallyFilled  OrderStatus = "PARTIALLY_FILLED"
	OrderStatusCancelled        OrderStatus = "CANCELLED"
	OrderStatusExpired          OrderStatus = "EXPIRED"
	
	// Position statuses
	PositionStatusOpen       PositionStatus = "OPEN"
	PositionStatusClosed     PositionStatus = "CLOSED"
	PositionStatusLiquidated PositionStatus = "LIQUIDATED"
	
	// Wallet types
	WalletTypeSpot    WalletType = "SPOT"
	WalletTypeFutures WalletType = "FUTURES"
)

// Wallet Models
type WalletBalance struct {
	Total             string `json:"total"`
	Withdrawable      string `json:"withdrawable"`
	Invested          string `json:"invested"`
	Rewards           string `json:"rewards"`
	CoinInvestable    string `json:"coin_investable"`
	CoinsetInvestable string `json:"coinset_investable"`
	VaultInvestable   string `json:"vault_investable"`
}

type FuturesBalance struct {
	Balance        string `json:"balance"`
	LockedAmount   string `json:"locked_amount"`
	FirstTimeUser  bool   `json:"first_time_user"`
}

type TransferResult struct {
	TransactionID string `json:"transaction_id"`
	Success       bool   `json:"success"`
}

// Asset Models
type Asset struct {
	AssetID       string `json:"asset_id"`
	Symbol        string `json:"symbol"`
	BaseCurrency  string `json:"base_currency"`
	QuoteCurrency string `json:"quote_currency"`
	MinQuantity   string `json:"min_quantity"`
	MaxQuantity   string `json:"max_quantity"`
	QuantityStep  string `json:"quantity_step"`
	MinLeverage   string `json:"min_leverage"`
	MaxLeverage   string `json:"max_leverage"`
	MakerFee      string `json:"maker_fee"`
	TakerFee      string `json:"taker_fee"`
	IsActive      bool   `json:"is_active"`
}

type AssetListResponse struct {
	Assets     []Asset `json:"assets"`
	Page       int     `json:"page"`
	PerPage    int     `json:"per_page"`
	Total      int     `json:"total"`
	TotalPages int     `json:"total_pages"`
}

// Leverage Models
type Leverage struct {
	AssetID    string     `json:"asset_id"`
	Leverage   string     `json:"leverage"`
	MarginType MarginType `json:"margin_type"`
}

// Order Models
type OrderRequest struct {
	Leverage         string      `json:"leverage"`
	Quantity         string      `json:"quantity"`
	OrderType        OrderType   `json:"order_type"`
	TriggerType      TriggerType `json:"trigger_type"`
	Price            *string     `json:"price,omitempty"`
	StopLossPrice    *string     `json:"stoploss_price,omitempty"`
	TakeProfitPrice  *string     `json:"takeprofit_price,omitempty"`
	ReduceOnly       bool        `json:"reduce_only,omitempty"`
}

type Order struct {
	OrderID         string        `json:"order_id"`
	Symbol          string        `json:"symbol"`
	AssetID         string        `json:"asset_id"`
	OrderType       OrderType     `json:"order_type"`
	TriggerType     TriggerType   `json:"trigger_type"`
	Price           string        `json:"price"`
	Quantity        string        `json:"quantity"`
	FilledQuantity  string        `json:"filled_quantity"`
	AvgFilledPrice  string        `json:"avg_filled_price"`
	Status          OrderStatus   `json:"status"`
	Leverage        string        `json:"leverage"`
	StopLossPrice   *string       `json:"stoploss_price,omitempty"`
	TakeProfitPrice *string       `json:"takeprofit_price,omitempty"`
	CreatedAt       time.Time     `json:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at"`
	ReduceOnly      bool          `json:"reduce_only"`
}

// Position Models
type Position struct {
	PositionID      string         `json:"position_id"`
	Symbol          string         `json:"symbol"`
	AssetID         string         `json:"asset_id"`
	EntryPrice      string         `json:"entry_price"`
	Quantity        string         `json:"quantity"`
	Side            OrderType      `json:"side"`
	Status          PositionStatus `json:"status"`
	Leverage        string         `json:"leverage"`
	UnrealizedPnL   string         `json:"unrealized_pnl"`
	RealizedPnL     string         `json:"realized_pnl"`
	Margin          string         `json:"margin"`
	MarginRatio     string         `json:"margin_ratio"`
	MarkPrice       string         `json:"mark_price"`
	StopLoss        *string        `json:"stop_loss,omitempty"`
	TakeProfit      *string        `json:"take_profit,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

// PnLPercentage calculates the P&L percentage
func (p *Position) PnLPercentage() (string, error) {
	// This is a placeholder - actual implementation would parse and calculate
	return "0", nil
}

// RiskOrder represents a stop loss or take profit order
type RiskOrder struct {
	OrderID         string    `json:"order_id"`
	PositionID      string    `json:"position_id"`
	OrderType       string    `json:"order_type"` // "STOP_LOSS" or "TAKE_PROFIT"
	TriggerPrice    string    `json:"trigger_price"`
	ExecutionPrice  *string   `json:"execution_price,omitempty"`
	Status          string    `json:"status"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// Fee Models
type FeeRecord struct {
	AssetID    string    `json:"asset_id"`
	Symbol     string    `json:"symbol"`
	FeeAmount  string    `json:"fee_amount"`
	FeeRate    string    `json:"fee_rate"`
	TradeType  string    `json:"trade_type"` // "MAKER" or "TAKER"
	OrderID    string    `json:"order_id"`
	CreatedAt  time.Time `json:"created_at"`
}

// APIResponse wraps API responses
type APIResponse struct {
	Success bool            `json:"success"`
	Message string          `json:"message,omitempty"`
	Data    json.RawMessage `json:"data,omitempty"`
	Error   *APIError       `json:"error,omitempty"`
}

// APIError represents an API error
type APIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}
