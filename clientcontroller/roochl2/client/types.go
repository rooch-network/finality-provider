package client

type GetStatesParams struct {
	AccessPath  string
	StateOption *StateOption
}

type StateOption struct {
	Decode      bool
	ShowDisplay bool
}

type ListStatesParams struct {
	AccessPath  string
	Cursor      string
	Limit       string
	StateOption *StateOption
}

type GetModuleABIParams struct {
	ModuleAddr string
	ModuleName string
}

type GetEventsByEventHandleParams struct {
	EventHandleType string
	Cursor          string
	Limit           string
	DescendingOrder bool
	EventOptions    map[string]interface{}
}

type QueryEventsParams struct {
	Filter      map[string]interface{}
	Cursor      string
	Limit       string
	QueryOption map[string]interface{}
}

type QueryInscriptionsParams struct {
	Filter          map[string]interface{}
	Cursor          string
	Limit           string
	DescendingOrder bool
}

type QueryUTXOsParams struct {
	Filter          map[string]interface{}
	Cursor          string
	Limit           string
	DescendingOrder bool
}

type BroadcastTXParams struct {
	Hex           string
	MaxFeeRate    float64
	MaxBurnAmount float64
}

type QueryObjectStatesParams struct {
	Filter      map[string]interface{}
	Cursor      string
	Limit       string
	QueryOption map[string]interface{}
}

type QueryTransactionsParams struct {
	Filter      map[string]interface{}
	Cursor      string
	Limit       string
	QueryOption map[string]interface{}
}

type GetBalanceParams struct {
	Owner    string
	CoinType string
}

type GetBalancesParams struct {
	Owner  string
	Cursor string
	Limit  string
}
