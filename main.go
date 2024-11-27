package main

import (
	"encoding/json"
	"fmt"
	"github.com/gagliardetto/solana-go"
	"github.com/gorilla/websocket"
	"log"
	"os"
	"sync"
	"time"
)

const (
	SolanaWebSocketURL = "wss://api.mainnet-beta.solana.com"
	PingInterval       = 15 * time.Second // 每 15 秒发送一次 Ping
	ReconnectInterval  = 5 * time.Second  // 断开后 5 秒后重连
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

// 定义 WebSocket 连接的状态
type WSConnection struct {
	conn              *websocket.Conn
	subscribeResponse bool
	stopPing          chan struct{}
	mu                sync.Mutex // 用于保护订阅响应状态
	logger            *log.Logger
	logFile           *os.File
}

// 建立 WebSocket 连接
func connect() (*websocket.Conn, error) {
	conn, _, err := websocket.DefaultDialer.Dial(SolanaWebSocketURL, nil)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

// 创建日志文件和对应的日志记录器
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

// 订阅日志
func subscribeToLogs(wsConn *WSConnection, address string, wg *sync.WaitGroup) {
	defer wg.Done() // 结束时通知 waitgroup

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
		wsConn.logger.Printf("订阅请求序列化失败: %v", err)
		return
	}

	err = wsConn.conn.WriteMessage(websocket.TextMessage, subscribeReqBytes)
	if err != nil {
		wsConn.logger.Printf("订阅请求发送失败: %v", err)
		return
	}

	wsConn.logger.Printf("订阅请求发送成功: %s", string(subscribeReqBytes))
	handleMessages(wsConn, address)
}

// 处理 WebSocket 消息
func handleMessages(wsConn *WSConnection, address string) {
	for {
		_, msg, err := wsConn.conn.ReadMessage()
		if err != nil {
			wsConn.logger.Printf("读取消息失败: %v", err)
			reconnect(wsConn, address)
			return
		}

		wsConn.logger.Printf("收到消息: %s", string(msg))

		var notification Notification
		err = json.Unmarshal(msg, &notification)
		if err != nil {
			wsConn.logger.Printf("消息解析失败: %v", err)
			continue
		}

		if notification.Method == "logsNotification" {
			signature := notification.Params.Result.Value.Signature
			wsConn.logger.Printf("交易签名: %s", signature)

			if notification.Params.Result.Value.Err != nil {
				wsConn.logger.Printf("交易失败: %v", notification.Params.Result.Value.Err)
			} else {
				wsConn.logger.Printf("交易成功")
			}
		}
	}
}

func reconnect(wsConn *WSConnection, address string) {
	for {
		conn, err := connect()
		if err != nil {
			wsConn.logger.Printf("WebSocket 重新连接失败，重试中: %v", err)
			time.Sleep(ReconnectInterval)
			continue
		}

		wsConn.logger.Printf("WebSocket 重新连接成功")
		wsConn.conn = conn
		wsConn.subscribeResponse = false
		go startPing(wsConn)
		subscribeToLogs(wsConn, address, &sync.WaitGroup{})
		return
	}
}

func startPing(wsConn *WSConnection) {
	ticker := time.NewTicker(PingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := wsConn.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				wsConn.logger.Printf("发送 Ping 消息失败: %v", err)
				return
			}
		case <-wsConn.stopPing:
			wsConn.logger.Println("心跳机制停止")
			return
		}
	}
}

func subscribeToSolanaLogs() {
	addresses := []string{
		solana.MustPublicKeyFromBase58("BkkrwtxMNkA66hfPbXcp1F3cxQihbG9ysYkHySsXhwyp").String(),
		solana.MustPublicKeyFromBase58("HXvUJoQuDvpZ4oNNFF5itafDfwMUCAFijLnjCwKVJ5rg").String(),
	}

	var wg sync.WaitGroup

	for _, address := range addresses {
		wg.Add(1)
		go func(address string) {
			logger, logFile, err := createLogger(address)
			if err != nil {
				fmt.Printf("初始化日志失败: %v", err)
				wg.Done()
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
				wsConn := &WSConnection{
					conn:              conn,
					stopPing:          make(chan struct{}),
					subscribeResponse: false,
					logger:            logger,
					logFile:           logFile,
				}

				go startPing(wsConn)
				subscribeToLogs(wsConn, address, &wg)
				wg.Wait()
			}
		}(address)
	}
	fmt.Println("websockets 监听启动...")
	wg.Wait()
}

func main() {
	subscribeToSolanaLogs()
}
