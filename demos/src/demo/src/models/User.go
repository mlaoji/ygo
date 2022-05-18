package models

//此文件是由 tools/genModel 自动生成, 可按需要修改

import (
	"demo/src/models/dao"
	//"github.com/mlaoji/ygo/models/dao/tx"
)

func User() *UserModel {
	return &UserModel{}
}

type UserModel struct{}

func (this *UserModel) GetUserInfo(id int) map[string]interface{} { // {{{
	return dao.NewDAOUser().GetRecord(id)
} //}}}

func (this *UserModel) AddUser() { // {{{
	//以下代码只是说明如何使用事务
	/*

		tx := tx.TransBegin()
		defer tx.Rollback()

		uid := dao.NewDAOUser(tx).AddRecord(map[string]interface{}{"name": "xx"})

		dao.NewDAOUserInfo(tx).AddRecord(map[string]interface{}{
			"uid":  uid,
			"info": "xxxx",
		})

		tx.Commit()

	*/
} //}}}
