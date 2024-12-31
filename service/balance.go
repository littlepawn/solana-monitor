package service

import (
	"context"
	"fmt"
	"log"
	"math/big"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/spf13/cobra"
	"meme/global"
)

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

// NewBalanceService 创建一个新的余额服务实例
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
		fmt.Printf("代币账户地址: %s\n", tokenAccount.Pubkey)
		fmt.Printf("tokenAccount.Account: %+v\n", tokenAccount.Account)
		accountData := tokenAccount.Account.Data.GetBinary()
		parseTokenAccountData(client, accountData)
	}
}

// 解析代币账户数据
func parseTokenAccountData(client *rpc.Client, accountData []byte) {
	// 检查账户数据长度是否符合 SPL Token 数据结构
	if len(accountData) < 165 {
		fmt.Println("账户数据长度不正确，可能不是一个有效的 SPL 代币账户")
		return
	}
	//fmt.Printf("accountData: %+v\n", accountData)

	// 提取余额数据
	amountBytes := accountData[64:72] // SPL Token 余额存储在字节 [64:72]
	reverseBytes(amountBytes)         // 转换为小端字节序
	amount := new(big.Int).SetBytes(amountBytes)
	if amount.Cmp(big.NewInt(1e6)) < 0 {
		return
	}

	// 根据精度计算余额
	decimals := getTokenDecimals(client, solana.PublicKeyFromBytes(accountData[0:32]))
	if decimals > 0 {
		amount = amount.Div(amount, big.NewInt(10).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil))
	}

	// 提取 Mint 地址（代币的唯一标识）
	mint := solana.PublicKeyFromBytes(accountData[0:32])

	fmt.Printf("代币地址 (Mint): %s\n", mint)
	fmt.Printf("余额: %s\n", amount)
	fmt.Println("--------------------------------------")
}

func reverseBytes(b []byte) {
	for i, j := 0, len(b)-1; i < j; i, j = i+1, j-1 {
		b[i], b[j] = b[j], b[i]
	}
}

// 获取代币 Decimals
func getTokenDecimals(client *rpc.Client, mint solana.PublicKey) int {
	accountInfo, err := client.GetAccountInfo(context.TODO(), mint)
	if err != nil {
		fmt.Printf("获取 Mint 信息失败: %v\n", err)
		return -1
	}

	data := accountInfo.Value.Data.GetBinary()
	// Mint 数据长度至少为 82 字节
	if len(data) < 82 {
		fmt.Println("Mint 数据长度不足，无法解析 Decimals")
		return -1
	}

	// Decimals 位于第 44 字节
	return int(data[44])
}
