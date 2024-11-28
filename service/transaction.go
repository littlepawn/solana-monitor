package service

import (
	"context"
	"encoding/json"
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

// TransactionService 表示交易服务
type TransactionService struct {
	logger *log.Logger
}

// NewTransactionService 创建一个新的交易服务实例
func NewTransactionService(logger *log.Logger) *TransactionService {
	return &TransactionService{
		logger: logger,
	}
}

func (s *TransactionService) GetTransactionLogs(address, signatureStr string) (TransactionRep, error) {
	client := rpc.New(rpc.MainNetBeta_RPC)
	var transactionRep, _preTransactionRep, _postTransactionRep TransactionRep
	signature, err := solana.SignatureFromBase58(signatureStr)
	if err != nil {
		s.logger.Printf("解析签名交易签名失败: %v", err)
		return transactionRep, err
	}

	txDetails, err := s.fetchTransaction(client, signature)
	if err != nil {
		return transactionRep, err
	}
	if len(txDetails.Meta.PreTokenBalances) == 0 || len(txDetails.Meta.PostTokenBalances) == 0 {
		return transactionRep, fmt.Errorf("非交易记录过滤")
	}

	for _, preTokenBalance := range txDetails.Meta.PreTokenBalances {
		if preTokenBalance.Owner.String() == address && preTokenBalance.ProgramId.String() == SPLTokenProgramID {
			_preTransactionRep = TransactionRep{
				Address: address,
				Amount:  preTokenBalance.UiTokenAmount.UiAmountString,
				Mint:    preTokenBalance.Mint.String(),
			}
			break
		}
	}

	for _, postTokenBalance := range txDetails.Meta.PostTokenBalances {
		if postTokenBalance.Owner.String() == address && postTokenBalance.ProgramId.String() == SPLTokenProgramID {
			_postTransactionRep = TransactionRep{
				Address: address,
				Amount:  postTokenBalance.UiTokenAmount.UiAmountString,
				Mint:    postTokenBalance.Mint.String(),
			}
			break
		}
	}
	s.logger.Printf("交易前数量: %v ========= 交易后数量: %v\n", _preTransactionRep.Amount, _postTransactionRep.Amount)
	if _preTransactionRep.Address == "" && _postTransactionRep.Address != "" {
		transactionRep = _postTransactionRep
		transactionRep.Type = "buy"
		s.logger.Printf("%s 买入数量: %s, mint: %s\n", address, transactionRep.Amount, transactionRep.Mint)
	}
	if _preTransactionRep.Address != "" {
		transactionRep = _preTransactionRep
		transactionRep.Type = "sell"
		if _postTransactionRep.Address != "" {
			preAmount, err := strconv.ParseFloat(_preTransactionRep.Amount, 64)
			if err != nil {
				s.logger.Printf("转化交易前数量失败 %v", err)
				return transactionRep, nil
			}
			postAmount, err := strconv.ParseFloat(_postTransactionRep.Amount, 64)
			if err != nil {
				s.logger.Printf("转化交易后数量失败 %v", err)
				return transactionRep, nil
			}
			if preAmount > postAmount {
				transactionRep.Amount = fmt.Sprintf("%.2f", preAmount-postAmount)
			} else {
				transactionRep.Amount = fmt.Sprintf("%.2f", postAmount-preAmount)
				transactionRep.Type = "buy"
				s.logger.Printf("%s 买入数量: %s, mint: %s\n", address, transactionRep.Amount, transactionRep.Mint)
				return transactionRep, nil
			}
		}
		s.logger.Printf("%s 卖出数量: %s, mint: %s\n", address, transactionRep.Amount, transactionRep.Mint)
	}
	return transactionRep, nil

}

// fetchTransaction fetches the transaction details from the Solana blockchain.
func (s *TransactionService) fetchTransaction(client *rpc.Client, signature solana.Signature) (*rpc.GetTransactionResult, error) {
	ctx := context.Background()
	s.logger.Printf("开始获取交易详情...")

	var tx *rpc.GetTransactionResult
	var err error
	maxRetries := 15             // 最大重试次数
	baseDelay := time.Second * 2 // 初始延迟
	maxDelay := 30 * time.Second

	for i := 0; i < maxRetries; i++ {
		tx, err = client.GetTransaction(ctx, signature, nil)
		if err == nil {
			// 请求成功，退出循环
			txLogsJson, err := json.Marshal(tx)
			if err != nil {
				s.logger.Printf("交易raw日志: %v", tx)
				s.logger.Printf("交易日志JSON序列化失败: %v", err)
			} else {
				s.logger.Printf("交易JSON日志: %s", txLogsJson)
			}
			break
		}

		if strings.Contains(err.Error(), "not found") {
			s.logger.Printf("交易未找到，重试中...")
		} else if strings.Contains(err.Error(), "Too many requests") {
			s.logger.Printf("请求速率限制。%d 秒后重试...", 1<<i) // 2^i 秒
		} else {
			s.logger.Printf("获取交易详情时发生错误: %v", err)
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
		s.logger.Printf("重试后交易中未找到日志")
		return nil, fmt.Errorf("no logs found in the transaction after retries")
	}

	return tx, nil
}
