/**
 * Auth :   liubo
 * Date :   2020/7/1 9:52
 * Comment:
 */

package main

import (
	"github.com/davyxu/cellnet"
	"github.com/davyxu/cellnet/codec"
	// 使用binary协议，因此匿名引用这个包，底层会自动注册
	_ "github.com/davyxu/cellnet/codec/binary"
	"github.com/davyxu/cellnet/util"
	"reflect"
)

type PtAck struct {
	Id int32
	Time int64
	Stuffing []int32
}

func init() {

	cellnet.RegisterMessageMeta(&cellnet.MessageMeta{
		Codec: codec.MustGetCodec("binary"),
		Type:  reflect.TypeOf((*PtAck)(nil)).Elem(),
		ID:    int(util.StringHash("PtAck")),
	})
}
