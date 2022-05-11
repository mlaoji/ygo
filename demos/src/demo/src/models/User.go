package models

//此文件是由 tools/genModel 自动生成, 可按需要修改

import (
	"demo/src/models/dao"
)

func User() *UserModel {
	return &UserModel{}
}

type UserModel struct{}

func (this *UserModel) GetUserInfo(id int) map[string]interface{} { // {{{
	return dao.NewDAOUser().GetRecord(id)
} //}}}
