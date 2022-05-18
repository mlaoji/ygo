package models

//此文件是由 tools/genRpcModel 自动生成, 可按需要修改

import (
	"fmt"
	"github.com/mlaoji/yclient"
	"github.com/mlaoji/ygo/x"
)

func Message() *MessageModel {
	return &MessageModel{}
}

type MessageModel struct {}

func (this *MessageModel) getClient() (*yclient.YClient, error) { // {{{
	conf := x.Conf.GetMap("rpc_client_message") 
	return yclient.NewYClient(conf["host"], conf["appid"], conf["secret"])
} //}}}

func (this *MessageModel) SendMessage() (x.MAP, error) { // {{{
   c, err := this.getClient()
   if nil != err {
      return nil, err
   }

   res, err := c.Request("message/sendMessage", x.MAP{})
   if err != nil {
      return nil, err
   }

   if res.GetCode() > 0 {
      return nil, fmt.Errorf("rpc client return err: %s", res.GetMsg())
   }

   return res.GetData(), nil
} // }}}
