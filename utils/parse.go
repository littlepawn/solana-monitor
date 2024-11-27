package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

func ParseTransactionLogs() {
	// 初始化 RPC 客户端
	client := rpc.New("https://api.mainnet-beta.solana.com")

	// 替换为实际的交易签名
	signatureStr := "5DcYSSqgrgivaWsWA4i3PNrNYkdFTnaU6EC2i3afiBun2kpNdFKSPKf14FWuhRSDCqiB7p9MeJQtf3oSqCK9LmFx"
	address := "HXvUJoQuDvpZ4oNNFF5itafDfwMUCAFijLnjCwKVJ5rg"

	ctx := context.Background()
	signature, err := solana.SignatureFromBase58(signatureStr)
	if err != nil {
		fmt.Printf("Failed to parse signature: %v\n", err)
		return
	}

	// 获取交易详情
	txDetails, err := client.GetTransaction(ctx, signature, nil)
	if err != nil {
		fmt.Printf("查询交易失败: %v\n", err)
		return
	}
	txDetailsJSON, err := json.MarshalIndent(txDetails, "", "  ")
	if err != nil {
		fmt.Printf("Failed to marshal transaction details: %v\n", err)
		return
	}
	fmt.Printf("交易详情:%s\n", txDetailsJSON)
	for _, tokenBalance := range txDetails.Meta.PostTokenBalances {
		if tokenBalance.Owner.String() == address && tokenBalance.ProgramId.String() == "TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA" {
			fmt.Printf("%s 买入数量: %s, mint: %s\n", address, tokenBalance.UiTokenAmount.UiAmountString, tokenBalance.Mint.String())
		}
	}
	
}
