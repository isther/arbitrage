package binance

import (
	"fmt"
	"net/url"

	"github.com/google/uuid"
)

type AccountStatusParams struct {
	ApiKey    string `json:"apiKey"`
	Timestamp int64  `json:"timestamp"`
	Signature string `json:"signature"`
}

// AccountEvent define account info
type AccountEvent struct {
	MakerCommission            int64           `json:"makerCommission"`
	TakerCommission            int64           `json:"takerCommission"`
	BuyerCommission            int64           `json:"buyerCommission"`
	SellerCommission           int64           `json:"sellerCommission"`
	CanTrade                   bool            `json:"canTrade"`
	CanWithdraw                bool            `json:"canWithdraw"`
	CanDeposit                 bool            `json:"canDeposit"`
	CommissionRates            CommissionRates `json:"commissionRates"`
	Brokered                   bool            `json:"brokered"`
	RequireSelfTradePrevention bool            `json:"requireSelfTradePrevention"`
	UpdateTime                 uint64          `json:"updateTime"`
	AccountType                string          `json:"accountType"`
	Balances                   []Balance       `json:"balances"`
	Permissions                []string        `json:"permissions"`
}

type CommissionRates struct {
	Maker  string `json:"maker"`
	Taker  string `json:"taker"`
	Buyer  string `json:"buyer"`
	Seller string `json:"seller"`
}

// Balance define user balance of your account
type Balance struct {
	Asset  string `json:"asset"`
	Free   string `json:"free"`
	Locked string `json:"locked"`
}

func NewAccountStatus(apiKey, secretKey string) *WsApiRequest {
	var (
		params    = url.Values{}
		timestamp = currentTimestamp() - TimeOffset
	)

	params.Set(string(PARAM_API_KEY), apiKey)
	params.Set(string(PARAM_TIMESTAMP), fmt.Sprintf("%v", timestamp))

	return &WsApiRequest{
		ID:     uuid.New().String(),
		Method: AccountStatus,
		Params: AccountStatusParams{
			ApiKey:    params.Get(string(PARAM_API_KEY)),
			Timestamp: timestamp,
			Signature: signature(params.Encode(), secretKey),
		},
	}
}
