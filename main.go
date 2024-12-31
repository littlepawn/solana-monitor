package main

import (
	"encoding/json"
	"fmt"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/gorilla/websocket"
	"github.com/spf13/cobra"

	"log"
	"meme/core"
	"meme/global"
	"meme/service"
	"os"
	"sync"
	"time"
)

const (
	SolanaWebSocketURL = "wss://api.mainnet-beta.solana.com"
	PingInterval       = 5 * time.Second
	ReconnectInterval  = 5 * time.Second
)

// RPCRequest 表示 RPC 请求格式
type RPCRequest struct {
	Jsonrpc string        `json:"jsonrpc"`
	Id      int           `json:"id"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
}

type Notification struct {
	Jsonrpc string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  struct {
		Result struct {
			Context struct {
				Slot uint64 `json:"slot"`
			} `json:"context"`
			Value struct {
				Signature string      `json:"signature"`
				Logs      []string    `json:"logs"`
				Err       interface{} `json:"err"`
			} `json:"value"`
		} `json:"result"`
		Subscription int `json:"subscription"`
	} `json:"params"`
}

// SafeWebSocket 封装了线程安全的 WebSocket 连接
type SafeWebSocket struct {
	conn    *websocket.Conn
	mu      sync.Mutex
	writeCh chan []byte
	stopCh  chan struct{}
	logger  *log.Logger
}

// NewSafeWebSocket 创建一个新的线程安全 WebSocket 连接
func NewSafeWebSocket(conn *websocket.Conn, logger *log.Logger) *SafeWebSocket {
	ws := &SafeWebSocket{
		conn:    conn,
		writeCh: make(chan []byte, 100),
		stopCh:  make(chan struct{}),
		logger:  logger,
	}
	go ws.writeLoop()
	return ws
}

// writeLoop 持续处理写入操作
func (ws *SafeWebSocket) writeLoop() {
	for {
		select {
		case msg := <-ws.writeCh:
			ws.mu.Lock()
			err := ws.conn.WriteMessage(websocket.TextMessage, msg)
			ws.mu.Unlock()
			if err != nil {
				ws.logger.Printf("WebSocket 写入失败: %v", err)
				return
			}
		case <-ws.stopCh:
			return
		}
	}
}

// SendMessage 发送消息到 WebSocket
func (ws *SafeWebSocket) SendMessage(msg []byte) {
	ws.writeCh <- msg
}

// Close 关闭 WebSocket
func (ws *SafeWebSocket) Close() {
	close(ws.stopCh)
	ws.conn.Close()
}

// connect 建立 WebSocket 连接
func connect() (*websocket.Conn, error) {
	conn, _, err := websocket.DefaultDialer.Dial(SolanaWebSocketURL, nil)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

// createLogger 创建日志记录器
func createLogger(address string) (*log.Logger, *os.File, error) {
	logDir := "logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, nil, fmt.Errorf("创建日志目录失败: %w", err)
	}

	filePath := fmt.Sprintf("%s/%s.log", logDir, address)
	logFile, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, nil, fmt.Errorf("打开日志文件失败: %w", err)
	}

	logger := log.New(logFile, "", log.LstdFlags)
	return logger, logFile, nil
}

// subscribeToLogs 订阅日志
func subscribeToLogs(ws *SafeWebSocket, address string) {
	params := []interface{}{
		map[string]interface{}{
			"mentions": []string{address},
		},
		map[string]interface{}{
			"commitment": "confirmed",
		},
	}

	subscribeReq := RPCRequest{
		Jsonrpc: "2.0",
		Id:      1,
		Method:  "logsSubscribe",
		Params:  params,
	}

	subscribeReqBytes, err := json.Marshal(subscribeReq)
	if err != nil {
		ws.logger.Printf("订阅请求序列化失败: %v", err)
		return
	}

	ws.SendMessage(subscribeReqBytes)
	ws.logger.Printf("订阅请求发送成功: %s", string(subscribeReqBytes))

	handleMessages(ws, address)
}

// handleMessages 处理 WebSocket 消息
func handleMessages(ws *SafeWebSocket, address string) {
	for {
		_, msg, err := ws.conn.ReadMessage()
		if err != nil {
			ws.logger.Printf("读取消息失败: %v", err)
			ws.logger.Printf("尝试重连...")
			reconnect(ws, address)
			return
		}

		ws.logger.Printf("收到消息: %s", string(msg))

		var notification Notification
		err = json.Unmarshal(msg, &notification)
		if err != nil {
			ws.logger.Printf("消息解析失败: %v", err)
			continue
		}

		if notification.Method == "logsNotification" {
			signature := notification.Params.Result.Value.Signature
			ws.logger.Printf("交易签名: %s", signature)

			transactionLogs, err := service.NewTransactionService(ws.logger).GetTransactionLogs(address, signature)
			if err != nil {
				ws.logger.Printf("获取交易日志失败: %v", err)
				continue
			}
			transactionLogsJson, err := json.Marshal(transactionLogs)
			if err != nil {
				ws.logger.Printf("解析后交易原始日志: %v", transactionLogs)
				ws.logger.Printf("解析后交易日志JSON序列化失败: %v", err)
			} else {
				ws.logger.Printf("解析后交易JSON日志: %s", transactionLogsJson)
			}

			if notification.Params.Result.Value.Err != nil {
				ws.logger.Printf("交易失败: %v", notification.Params.Result.Value.Err)
			} else {
				ws.logger.Printf("交易成功")
			}
		}
	}
}

// subscribeToSolanaLogs 订阅多个地址
func subscribeToSolanaLogs(addresses []string) {
	for _, address := range addresses {
		go func(address string) {
			logger, logFile, err := createLogger(address)
			if err != nil {
				fmt.Printf("初始化日志失败: %v", err)
				return
			}
			defer logFile.Close()

			for {
				conn, err := connect()
				if err != nil {
					logger.Printf("WebSocket 连接失败，重试中: %v", err)
					time.Sleep(ReconnectInterval)
					continue
				}

				logger.Printf("WebSocket 连接成功")
				ws := NewSafeWebSocket(conn, logger)
				defer ws.Close()
				go startPing(ws)
				subscribeToLogs(ws, address)
			}
		}(address)
	}
}

func startPing(ws *SafeWebSocket) {
	ticker := time.NewTicker(PingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ws.mu.Lock()
			err := ws.conn.WriteMessage(websocket.PingMessage, nil)
			ws.mu.Unlock()
			if err != nil {
				ws.logger.Printf("发送 Ping 消息失败: %v", err)
				return
			}
		case <-ws.stopCh:
			ws.logger.Println("心跳机制停止")
			return
		}
	}
}

func reconnect(ws *SafeWebSocket, address string) {
	for {
		conn, err := connect()
		if err != nil {
			ws.logger.Printf("WebSocket 重新连接失败，重试中: %v", err)
			time.Sleep(ReconnectInterval)
			continue
		}

		ws.logger.Printf("WebSocket 重新连接成功")
		ws.conn = conn
		go startPing(ws)
		subscribeToLogs(ws, address)
		return
	}
}

func main() {
	global.SystemConfig = core.InitSystemConfig()
	var addresses []string
	selfAddress := global.SystemConfig.SelfAddress
	if selfAddress != "" {
		addresses = append(addresses, solana.MustPublicKeyFromBase58(selfAddress).String())
	}
	monitorAddress := global.SystemConfig.MonitorAddress
	if monitorAddress != "" {
		addresses = append(addresses, solana.MustPublicKeyFromBase58(monitorAddress).String())
	}

	var rootCmd = &cobra.Command{
		Use: "main",
		Run: func(cmd *cobra.Command, args []string) {
			if len(addresses) == 0 {
				fmt.Println("请配置监控地址")
				os.Exit(1)
			}
			// 初始化 Redis
			global.Redis = core.InitRedis()
			fmt.Printf("Redis 连接成功: %v\n", global.Redis)

			// 初始化 Solana RPC 客户端
			global.RpcClient = rpc.New(rpc.MainNetBeta_RPC)
			fmt.Printf("RPC 客户端初始化成功: %v\n", global.RpcClient)

			// 启动 Solana WebSocket 订阅
			fmt.Println("启动 Solana WebSocket 订阅...")
			subscribeToSolanaLogs(addresses)

			// 阻塞主线程
			select {}
		},
	}

	rootCmd.AddCommand(service.BalanceCmd)
	rootCmd.AddCommand(service.TokenCmd)
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
