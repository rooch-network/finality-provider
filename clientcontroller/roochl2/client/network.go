package client

import "fmt"

type NetworkType string

const (
	NetworkMainnet  NetworkType = "mainnet"
	NetworkTestnet  NetworkType = "testnet"
	NetworkDevnet   NetworkType = "devnet"
	NetworkLocalnet NetworkType = "localnet"
)

//export type NetWorkType = 'mainnet' | 'testnet' | 'devnet' | 'localnet'

//export function getRoochNodeUrl(network: NetWorkType) {
//switch (network) {
//case 'mainnet':
//return 'https://main-seed.rooch.network'
//case 'testnet':
//return 'https://test-seed.rooch.network'
//case 'devnet':
//return 'https://dev-seed.rooch.network'
//case 'localnet':
//return 'http://127.0.0.1:6767'
//default:
//throw new Error(`Unknown network: ${network}`)
//}
//}

func GetRoochNodeUrl(network NetworkType) (string, error) {
	switch network {
	case NetworkMainnet:
		return "https://main-seed.rooch.network", nil
	case NetworkTestnet:
		return "https://test-seed.rooch.network", nil
	case NetworkDevnet:
		return "https://dev-seed.rooch.network", nil
	case NetworkLocalnet:
		return "http://127.0.0.1:6767", nil
	default:
		return "", fmt.Errorf("unknown network")
	}
}
