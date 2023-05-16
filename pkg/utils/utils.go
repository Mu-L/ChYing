package utils

import (
	"fmt"
	"net"
)

/**
  @author: yhy
  @since: 2023/5/11
  @desc: //TODO
**/

func IsPortOccupied(port int) bool {
	address := fmt.Sprintf(":%d", port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return true // 端口已被占用
	}
	listener.Close()
	return false // 端口未被占用
}
