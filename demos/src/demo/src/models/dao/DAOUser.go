package dao

//此文件是由 ./tools/genModel 自动生成

import (
	"github.com/mlaoji/ygo/models/dao"
	"github.com/mlaoji/ygo/x"
)

func NewDAOUser(tx ...x.DBClient) *DAOUser {
	ins := &DAOUser{}
	ins.Init(tx...)
	return ins
}

type DAOUser struct {
	dao.DAOProxy
}

func (this *DAOUser) Init(tx ...x.DBClient) {
	if len(tx) > 0 {
		this.DAOProxy.InitTx(tx[0])
	} else {
		this.DAOProxy.Init()
	}
	this.SetTable("user")
	this.SetPrimary("uid")
}
