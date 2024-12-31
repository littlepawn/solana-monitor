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
		//fmt.Printf("代币账户地址: %s\n", tokenAccount.Pubkey)
		//fmt.Printf("tokenAccount.Account: %+v\n", tokenAccount.Account)
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
	// 提取 Mint 地址（代币的唯一标识）
	mint := solana.PublicKeyFromBytes(accountData[0:32])

	metadata, _ := GetTokenMetadata(client, mint)

	// 根据精度计算余额
	decimals := getTokenDecimals(client, solana.PublicKeyFromBytes(accountData[0:32]))
	if decimals > 0 {
		// 余额 = 余额 / 10^decimals，保留 8 位小数点
		divisor := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil))
		amountFloat := new(big.Float).Quo(new(big.Float).SetInt(amount), divisor)

		// 格式化余额为保留 8 位小数
		amountFormatted := fmt.Sprintf("%.8f", amountFloat)
		fmt.Printf("余额(已处理精度): %s\n", amountFormatted)
	} else {

		fmt.Printf("余额(未处理精度): %s\n", amount)
	}
	//fmt.Printf("代币精度: %d\n", decimals)
	fmt.Printf("代币地址 (Mint): %s\n", mint)
	fmt.Printf("代币符号: %s\n", metadata.Symbol)
	fmt.Println("--------------------------------------")
}

func reverseBytes(b []byte) {
	for i, j := 0, len(b)-1; i < j; i, j = i+1, j-1 {
		b[i], b[j] = b[j], b[i]
	}
}

// 获取代币 Decimals
func getTokenDecimals(client *rpc.Client, mint solana.PublicKey) int {
	// 调用 getTokenSupply 获取精度
	response, err := client.GetTokenSupply(context.TODO(), mint, rpc.CommitmentConfirmed)
	if err != nil || response.Value.Decimals == 0 {
		fmt.Printf("获取精度时出错: %v\n", err)
		return 0
	}
	//spew.Dump(response)
	return int(response.Value.Decimals)
}
