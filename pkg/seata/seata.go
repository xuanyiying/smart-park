package seata

import (
	"log"

	"github.com/seata/seata-go/pkg/client"
)

func Init() error {
	// 初始化 Seata 客户端
	client.InitPath("configs/seata.yaml")

	log.Println("Seata initialized successfully")
	return nil
}
