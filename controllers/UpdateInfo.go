package controllers

import (
	// "fmt"
	"encoding/json"
	"github.com/astaxie/beego"
)

type UpdateInfo struct {
	Version  string
	FileList FileChecksumList
	DirList  FileChecksumList //dir
}

func (this *UpdateInfo) Print() {
	beego.Info("升级信息：版本号 " + this.Version)
	if len(this.DirList) > 0 {
		beego.Info("目录列表：")
		this.DirList.Print()
	} else {
		beego.Info("目录列表为空")
	}
	if len(this.FileList) > 0 {
		beego.Info("文件列表：")
		this.FileList.Print()
	} else {
		beego.Info("文件列表为空")
	}
}
func (this *UpdateInfo) ToJson() ([]byte, error) {
	return json.Marshal(this)
}

func NewUpdateInfo(version string, list, dirList FileChecksumList) *UpdateInfo {
	return &UpdateInfo{
		Version:  version,
		FileList: list,
		DirList:  dirList,
	}
}
