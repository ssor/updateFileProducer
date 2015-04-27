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
	this.FileList.Print()
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
