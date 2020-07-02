/**
 * Auth :   liubo
 * Date :   2020/7/1 13:29
 * Comment:
 */

package main

import (
	"strings"
)

type IDevice interface {
	Close()
}

type IServer interface {
	IDevice
	OpenServer(addr string)

}

type IClient interface {
	IDevice
	OpenClient(serverAddr string)
}

func NewClient(protocol string) IClient {
	protocol = strings.ToLower(protocol)

	return &NetClient{Protocol:protocol, Processor:protocol + ".ltv"}
}

func NewServer(protocol string) IServer {
	protocol = strings.ToLower(protocol)

	return &NetServer{Protocol:protocol, Processor:protocol + ".ltv"}
}

