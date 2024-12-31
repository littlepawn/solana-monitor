package service

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/spf13/cobra"
	"log"
	"meme/global"
	"net/http"
)

var TokenCmd = &cobra.Command{
	Use:   "token",
	Short: "Get token metadata",
	Run: func(cmd *cobra.Command, args []string) {
		client := rpc.New(rpc.MainNetBeta_RPC)
		selfAddress := global.SystemConfig.SelfAddress
		address := solana.MustPublicKeyFromBase58(selfAddress)
		GetTokenMetadata(client, address)
	},
}

type TokenMetadata struct {
	Name string
	// 可扩展为包含更多元数据字段，如符号、URI 等
}

type TokenService struct {
	logger *log.Logger
}

func NewTokenService(logger *log.Logger) *TokenService {
	return &TokenService{
		logger: logger,
	}
}

func GetTokenMetadata(client *rpc.Client, mint solana.PublicKey) string {
	// Metadata program ID
	metadataProgramID := solana.MustPublicKeyFromBase58("metaqbxxUerdq28cj1RbAWkYQm3ybzjb6a8bt518x1s")

	// Derive metadata address
	metadataAddress, _, err := solana.FindProgramAddress(
		[][]byte{
			[]byte("metadata"),
			metadataProgramID.Bytes(),
			mint.Bytes(),
		},
		metadataProgramID,
	)
	if err != nil {
		fmt.Printf("无法推导元数据地址: %v\n", err)
		return "Unknown"
	}

	// Fetch account data
	accountInfo, err := client.GetAccountInfo(context.TODO(), metadataAddress)
	if err != nil || accountInfo == nil || accountInfo.Value == nil {
		fmt.Printf("获取元数据失败: %v\n", err)
		return "Unknown"
	}

	// Decode metadata (depends on token metadata structure)
	data := accountInfo.Value.Data.GetBinary()
	decodedMetadata := parseTokenMetadata(data)
	return decodedMetadata.Name
}

func GetTokenPrice(mint solana.PublicKey) (float64, error) {
	// 调用外部 API 获取代币价格，例如 CoinGecko 或 Serum 数据源
	// 这是伪代码，需要替换为实际 API 调用
	priceAPI := "https://api.coingecko.com/api/v3/simple/token_price/solana?contract_addresses=" + mint.String() + "&vs_currencies=usd"
	resp, err := http.Get(priceAPI)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var result map[string]map[string]float64
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}

	price, ok := result[mint.String()]["usd"]
	if !ok {
		return 0, fmt.Errorf("未找到价格信息")
	}
	return price, nil
}

func parseTokenMetadata(data []byte) TokenMetadata {
	// 解析 token metadata 字节，根据 SPL Token Metadata Program 的规范
	fmt.Printf("元数据字节: %v\n", data)
	fmt.Println("元数据:", string(data))
	// 假设名字字段在固定位置并以 NULL 结尾
	name := string(data[33:65]) // 字节 33 到 65 是名称字段（根据 Token Metadata Program 的规范）
	return TokenMetadata{Name: name}
}
