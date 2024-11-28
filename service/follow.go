package service

import (
	"context"
	"fmt"
	"log"
	"meme/global"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/token"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/gagliardetto/solana-go/rpc/jsonrpc"
)

type FollowTransactionService struct {
	client *rpc.Client
	logger *log.Logger
}

func NewFollowTransactionService(client *rpc.Client, logger *log.Logger) *FollowTransactionService {
	return &FollowTransactionService{client: client, logger: logger}
}

// FollowAndSend 根据指定 mint 筛选交易并跟单
func (fts *FollowTransactionService) FollowAndSend(address solana.PublicKey, signatureStr string, followMint solana.PublicKey, destination solana.PublicKey) error {
	signature, err := solana.SignatureFromBase58(signatureStr)
	if err != nil {
		return fmt.Errorf("解析签名失败: %v", err)
	}

	// 获取交易详情
	txDetails, err := fts.client.GetTransaction(context.TODO(), signature, nil)
	if err != nil {
		if rpcErr, ok := err.(*jsonrpc.RPCError); ok {
			fts.logger.Printf("RPC 错误: %s", rpcErr.Message)
		}
		return fmt.Errorf("获取交易详情失败: %v", err)
	}

	// 筛选出与 followMint 相关的交易
	var tokenAmount string
	for _, postTokenBalance := range txDetails.Meta.PostTokenBalances {
		if postTokenBalance.Mint == followMint {
			tokenAmount = postTokenBalance.UiTokenAmount.UiAmountString
			break
		}
	}

	if tokenAmount == "" {
		fts.logger.Println("未找到符合条件的交易记录")
		return nil
	}

	// 构造交易（转移 Token）
	return fts.createAndSendTransaction(followMint, address, destination, tokenAmount)
}

func (fts *FollowTransactionService) createAndSendTransaction(mint solana.PublicKey, source solana.PublicKey, destination solana.PublicKey, amount string) error {
	// 加载钱包密钥对
	wallet, err := solana.PrivateKeyFromBase58("<你的钱包私钥>")
	if err != nil {
		return fmt.Errorf("加载钱包失败: %v", err)
	}

	// 构造交易
	tx, err := solana.NewTransaction(
		[]solana.Instruction{
			token.NewTransferCheckedInstruction(
				uint64(1000), // 转移的数量（按最小单位）; 可根据需要转换 `amount` 为 uint64
				uint8(3),

				source,
				mint, // Token 的 mint 地址
				destination,
				wallet.PublicKey(),
				[]solana.PublicKey{wallet.PublicKey()},
			).Build(),
		},
		solana.Hash(solana.MustPublicKeyFromBase58("<最近的区块哈希>")),
	)
	if err != nil {
		return fmt.Errorf("构造交易失败: %v", err)
	}

	// 签名并发送交易
	_, err = fts.client.SendTransactionWithOpts(context.TODO(), tx, rpc.TransactionOpts{})
	if err != nil {
		return fmt.Errorf("发送交易失败: %v", err)
	}

	fts.logger.Printf("成功发送跟单交易: %s -> %s (数量: %s)", source, destination, amount)
	return nil
}

func test() {
	logger := log.Default()
	client := global.RpcClient
	service := NewFollowTransactionService(client, logger)

	err := service.FollowAndSend(
		solana.MustPublicKeyFromBase58("<钱包地址>"),
		"<目标签名>",
		solana.MustPublicKeyFromBase58("<目标 Mint 地址>"),
		solana.MustPublicKeyFromBase58("<接收地址>"),
	)
	if err != nil {
		logger.Fatalf("跟单交易失败: %v", err)
	}
}
