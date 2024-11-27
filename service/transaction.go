package service

import (
	"context"
	"fmt"
	"github.com/gagliardetto/solana-go"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/gagliardetto/solana-go/rpc"
)

const (
	SPLTokenProgramID = "TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA"
	SystemProgramID   = "11111111111111111111111111111111"
)

type TransactionRep struct {
	Address string
	Amount  string
	Mint    string
	Type    string
}

type TransactionService struct{}

func NewTransactionService() *TransactionService {
	return &TransactionService{}
}

func (s *TransactionService) GetTransactionLogs(address, signatureStr string) (TransactionRep, error) {
	client := rpc.New(rpc.MainNetBeta_RPC)
	var transactionRep, _preTransactionRep, _postTransactionRep TransactionRep
	signature, err := solana.SignatureFromBase58(signatureStr)
	if err != nil {
		log.Fatalf("Failed to parse signature: %v", err)
		return transactionRep, err
	}

	txDetails, err := fetchTransaction(client, signature)
	if err != nil {
		return transactionRep, err
	}
	for _, tokenBalance := range txDetails.Meta.PreTokenBalances {
		if tokenBalance.Owner.String() == address && tokenBalance.ProgramId.String() == SPLTokenProgramID {
			fmt.Printf("%s 买入数量: %s, mint: %s\n", address, tokenBalance.UiTokenAmount.UiAmountString, tokenBalance.Mint.String())
			_preTransactionRep = TransactionRep{
				Address: address,
				Amount:  tokenBalance.UiTokenAmount.UiAmountString,
				Mint:    tokenBalance.Mint.String(),
			}
			break
		}
	}

	for _, tokenBalance := range txDetails.Meta.PostTokenBalances {
		if tokenBalance.Owner.String() == address && tokenBalance.ProgramId.String() == SPLTokenProgramID {
			_postTransactionRep = TransactionRep{
				Address: address,
				Amount:  tokenBalance.UiTokenAmount.UiAmountString,
				Mint:    tokenBalance.Mint.String(),
			}
			break
		}
	}
	if _preTransactionRep.Address == "" && _postTransactionRep.Address != "" {
		transactionRep = _postTransactionRep
		transactionRep.Type = "buy"
		fmt.Printf("%s 买入数量: %s, mint: %s\n", address, transactionRep.Amount, transactionRep.Mint)
	}
	if _preTransactionRep.Address != "" {
		transactionRep = _preTransactionRep
		transactionRep.Type = "sell"
		if _postTransactionRep.Address != "" {
			preAmount, err := strconv.ParseFloat(_preTransactionRep.Amount, 64)
			if err != nil {
				fmt.Printf("Failed to parse pre-transaction amount: %v", err)
			}
			postAmount, err := strconv.ParseFloat(_postTransactionRep.Amount, 64)
			if err != nil {
				fmt.Printf("Failed to parse post-transaction amount: %v", err)
			}
			if preAmount > postAmount {
				transactionRep.Amount = fmt.Sprintf("%.2f", preAmount-postAmount)
			} else {
				transactionRep.Amount = fmt.Sprintf("%.2f", postAmount-preAmount)
				transactionRep.Type = "buy"
				fmt.Printf("%s 买入数量: %s, mint: %s\n", address, transactionRep.Amount, transactionRep.Mint)
				return transactionRep, nil
			}
		}
		fmt.Printf("%s 卖出数量: %s, mint: %s\n", address, transactionRep.Amount, transactionRep.Mint)
	}
	return transactionRep, nil

}

// fetchTransaction fetches the transaction details from the Solana blockchain.
func fetchTransaction(client *rpc.Client, signature solana.Signature) (*rpc.GetTransactionResult, error) {
	ctx := context.Background()
	fmt.Println("Fetching transaction details...")

	var tx *rpc.GetTransactionResult
	var err error
	maxRetries := 15             // 最大重试次数
	baseDelay := time.Second * 2 // 初始延迟
	maxDelay := 30 * time.Second

	for i := 0; i < maxRetries; i++ {
		tx, err = client.GetTransaction(ctx, signature, nil)
		if err == nil {
			// 请求成功，退出循环
			break
		}

		if strings.Contains(err.Error(), "not found") {
			fmt.Println("Transaction not found, retrying...")
		} else if strings.Contains(err.Error(), "Too many requests") {
			fmt.Printf("Rate limit hit. Retrying after %d seconds...\n", 1<<i) // 2^i 秒
		} else {
			return nil, fmt.Errorf("failed to fetch transaction: %w", err)
		}

		// 计算动态延迟
		delay := baseDelay * time.Duration(1<<i)
		if delay > maxDelay {
			delay = maxDelay
		}
		time.Sleep(delay)
	}

	// 检查最终结果是否有效
	if tx == nil || tx.Meta == nil || tx.Meta.LogMessages == nil {
		return nil, fmt.Errorf("no logs found in the transaction after retries")
	}

	return tx, nil
}
