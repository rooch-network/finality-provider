package types

//// QueryInscriptionsParams represents parameters for querying inscriptions
//type QueryInscriptionsParams struct {
//	Filter          InscriptionFilterView `json:"filter"`
//	Cursor          *IndexerStateIDView   `json:"cursor,omitempty"`
//	Limit           *string               `json:"limit,omitempty"`
//	DescendingOrder *bool                 `json:"descendingOrder,omitempty"`
//}

//cursor: Option<StrView<u128>>,
//limit: Option<StrView<u64>>,
//descending_order: Option<bool>,

// QueryInscriptionsParams represents parameters for querying inscriptions
type GetBlocksParams struct {
	BlockType       string `json:"blockType,omitempty"`
	Cursor          string `json:"cursor,omitempty"`
	Limit           string `json:"limit,omitempty"`
	DescendingOrder bool   `json:"descendingOrder,omitempty"`
}

type BlockType string

const (
	BlockTypeAll       BlockType = "all"
	BlockTypeFinalized BlockType = "finalized"
)
