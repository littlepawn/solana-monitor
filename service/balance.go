package service

import (
	"context"
	"fmt"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/spf13/cobra"
	"log"

	"meme/global"
)

type TokenAmount struct {
	Amount         string `json:"amount"`
	Decimals       uint8  `json:"decimals"`
	UiAmountString string `json:"uiAmountString"`
}

type TokenAccountData struct {
	Parsed struct {
		Info struct {
			TokenAmount TokenAmount `json:"tokenAmount"`
		} `json:"info"`
	} `json:"parsed"`
}

var BalanceCmd = &cobra.Command{
	Use:   "balance",
	Short: "Get SOL and token balances",
	Run: func(cmd *cobra.Command, args []string) {
		client := rpc.New(rpc.MainNetBeta_RPC)
		selfAddress := global.SystemConfig.SelfAddress
		address := solana.MustPublicKeyFromBase58(selfAddress)

		getSolBalance(client, address)
		getTokenBalances(client, address)
	},
}

type BalanceService struct {
	logger *log.Logger
}

// NewTransactionService 创建一个新的交易服务实例
func NewBalanceService(logger *log.Logger) *BalanceService {
	return &BalanceService{
		logger: logger,
	}
}

// 获取 SOL 余额
func getSolBalance(client *rpc.Client, address solana.PublicKey) {
	balance, err := client.GetBalance(context.TODO(), address, rpc.CommitmentConfirmed)
	if err != nil {
		log.Fatalf("获取 SOL 余额失败: %v", err)
	}
	fmt.Printf("账户: %s 的 SOL 余额: %.9f SOL\n", address.String(), float64(balance.Value)/1e9)
}

// 获取代币账户及余额
func getTokenBalances(client *rpc.Client, address solana.PublicKey) {
	// 查询账户代币持有情况
	response, err := client.GetTokenAccountsByOwner(
		context.TODO(),
		address,
		&rpc.GetTokenAccountsConfig{
			ProgramId: &solana.TokenProgramID,
		},
		&rpc.GetTokenAccountsOpts{Commitment: rpc.CommitmentConfirmed},
	)
	if err != nil {
		log.Fatalf("获取代币账户失败: %v", err)
	}

	if len(response.Value) == 0 {
		fmt.Printf("账户 %s 未持有任何 SPL 代币\n", address.String())
		return
	}

	fmt.Printf("账户 %s 持有的 SPL 代币列表:\n", address.String())
	for _, tokenAccount := range response.Value {
		// 获取代币账户地址和余额
		fmt.Printf("tokenAccount.Pubkey: %+v\n", tokenAccount.Pubkey)
		fmt.Printf("tokenAccount.Account: %+v\n", tokenAccount.Account)
		fmt.Printf("tokenAccount.Account.Data: %+v\n", tokenAccount.Account.Data)
		data, err := tokenAccount.Account.Data.MarshalJSON()
		if err != nil {
			fmt.Println("解析代币账户数据失败:", err)
			continue
		}
		//fmt.Printf("data: %+v\n", data)
		fmt.Println("data: ", string(data))
		fmt.Println("--------------------------------------")
	}
}
