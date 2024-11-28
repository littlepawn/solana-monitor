package global

import (
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/redis/go-redis/v9"
)

var (
	Redis     *redis.Client
	RpcClient *rpc.Client
)
