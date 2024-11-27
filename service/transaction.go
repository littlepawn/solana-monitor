package service

import (
	"context"
	"fmt"
	"github.com/gagliardetto/solana-go"
	"log"
	"strings"
	"time"

	"github.com/gagliardetto/solana-go/rpc"
)

const (
	SPLTokenProgramID = "TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA"
	SystemProgramID   = "11111111111111111111111111111111"
)

type CategorizedLogs struct {
	SPLTokenLogs []string
	SystemLogs   []string
	OtherLogs    []string
}

type TransactionService struct{}

func NewTransactionService() *TransactionService {
	return &TransactionService{}
}

func (s *TransactionService) GetTransactionLogs(signatureStr string) error {
	client := rpc.New(rpc.MainNetBeta_RPC)
	signature, err := solana.SignatureFromBase58(signatureStr)
	if err != nil {
		log.Fatalf("Failed to parse signature: %v", err)
		return err
	}

	categorizedLogs, err := categorizeTransactionLogs(client, signature)
	if err != nil {
		log.Fatalf("Failed to categorize logs: %v", err)
		return err
	}

	printLogs(categorizedLogs)
	return nil
}

// categorizeTransactionLogs categorizes transaction logs based on program IDs.
func categorizeTransactionLogs(client *rpc.Client, signature solana.Signature) (CategorizedLogs, error) {
	tx, err := fetchTransaction(client, signature)
	if err != nil {
		return CategorizedLogs{}, err
	}

	return categorizeLogs(tx.Meta.LogMessages), nil
}

// fetchTransaction fetches the transaction details from the Solana blockchain.
func fetchTransaction(client *rpc.Client, signature solana.Signature) (*rpc.GetTransactionResult, error) {
	ctx := context.Background()
	fmt.Println("Fetching transaction details...")

	var tx *rpc.GetTransactionResult
	var err error
	maxRetries := 10         // 最大重试次数
	baseDelay := time.Second // 初始延迟
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

// categorizeLogs categorizes logs based on predefined program IDs.
func categorizeLogs(logMessages []string) CategorizedLogs {
	categorizedLogs := CategorizedLogs{}
	for _, logMsg := range logMessages {
		switch {
		case strings.Contains(logMsg, SPLTokenProgramID):
			categorizedLogs.SPLTokenLogs = append(categorizedLogs.SPLTokenLogs, logMsg)
		case strings.Contains(logMsg, SystemProgramID):
			categorizedLogs.SystemLogs = append(categorizedLogs.SystemLogs, logMsg)
		default:
			categorizedLogs.OtherLogs = append(categorizedLogs.OtherLogs, logMsg)
		}
	}
	return categorizedLogs
}

// printLogs prints the categorized logs to the console.
func printLogs(categorizedLogs CategorizedLogs) {
	fmt.Println("SPL Token Logs:")
	for _, splTokenLog := range categorizedLogs.SPLTokenLogs {
		fmt.Println(splTokenLog)
	}

	fmt.Println("\nSystem Logs:")
	for _, systemLog := range categorizedLogs.SystemLogs {
		fmt.Println(systemLog)
	}

	fmt.Println("\nOther Logs:")
	for _, otherLog := range categorizedLogs.OtherLogs {
		fmt.Println(otherLog)
	}
}
