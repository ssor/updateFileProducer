package controllers

import (
	"bufio"
	"crypto/sha256"
	"fmt"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/config"
	"github.com/codegangsta/cli"
	"io"
	"os"
	"path/filepath"
	// "strconv"
	"path"
	"strings"
)

var G_updateInfo *UpdateInfo

// var Version = "1.0"

// var G_FileList = FileChecksumList{}
var IgnoreFileNameList = []string{}
var G_ignoreFolderNameList = []string{}

// var G_printLog = true
// var DebugLevel = 4
var iniconf config.ConfigContainer = nil
var G_dirSrc = "./"
var G_dirDest = "./output/"
var G_appVersion = ""
var G_versionInfoFileName = "VersionInfo.md"

func init() {
	G_updateInfo = NewUpdateInfo("1.0", FileChecksumList{}, FileChecksumList{})
	initConfig()
	go initCli()
}

type MainController struct {
	beego.Controller
}

func (this *MainController) Get() {
	this.Data["Website"] = "beego.me"
	this.Data["Email"] = "astaxie@gmail.com"
	this.TplNames = "index.tpl"
}

func OutputVersionFile() {
	destVerFile := G_dirDest + G_versionInfoFileName
	if Exist(destVerFile) == true { //将所有文件输出完成后，才能创建版本信息文件作为完成的标识，所以一开始不能存在该文件
		if err := os.Remove(destVerFile); err != nil {
			beego.Warn("删除输出目录的版本信息文件" + destVerFile + "出错：" + err.Error())
			return
		}
	}
	dirList, fileList, err := CreateChecksumPathList(G_dirSrc)
	if err != nil {
		beego.Error("创建源目录文件列表出错：" + err.Error())
		return
	}
	G_updateInfo.FileList = fileList
	G_updateInfo.DirList = dirList

	err = copyFileToOutputDir()
	if err != nil {
		beego.Error("向输出目录拷贝文件时出错：" + err.Error())
		return
	}

	if err := createVersionInfoFile(destVerFile); err != nil {
		beego.Error("在输出目录创建版本信息文件出错：" + err.Error())
		return
	}
}
func copyFileToOutputDir() error {
	//对比对应Bin目录的文件，进行同步

	//同步目录
	// outputBinDirList := []string{}
	for _, file := range G_updateInfo.DirList {
		destDir := strings.Replace(file.Path, G_dirSrc, G_dirDest+"Bin/", 1)
		if Exist(destDir) == false {
			if err := os.MkdirAll(destDir, os.ModePerm); err != nil {
				return err
			}
			beego.Info("创建了目录：" + destDir)
		}
	}

	// 同步文件
	for _, file := range G_updateInfo.FileList {
		destDir := strings.Replace(file.Path, G_dirSrc, G_dirDest+"Bin/", 1)
		if Exist(destDir) == false {
			if err := CopyFile(destDir, file.Path); err != nil {
				return err
			}
			beego.Info("复制了文件：" + destDir)
		} else {
			if checksum, err := createChecksumForFile(destDir); err != nil {
				return err
			} else {
				if checksum == file.Checksum {
					beego.Info("源文件与目标文件相同，不需要复制 " + file.Path)
				} else {
					beego.Info("源文件存在，但是文件发生了变化，需要首先删除源文件")
					if err := os.Remove(destDir); err != nil {
						return err
					} else {
						if err := CopyFile(destDir, file.Path); err != nil {
							return err
						}
						beego.Info("复制了文件：" + destDir)
					}
				}
			}
		}
	}

	return nil
}
func createVersionInfoFile(versionInfoFileFullPath string) error {
	ui := G_updateInfo
	ui.Print()

	bytes, err := ui.ToJson()
	if err != nil {
		return err
	}
	if Exist(versionInfoFileFullPath) == true {
		if err := os.Remove(versionInfoFileFullPath); err != nil {
			return err
		}
	}
	if fd, err := os.OpenFile(versionInfoFileFullPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755); err != nil {
		return err
	} else {
		defer fd.Close()
		if _, err := fd.Write(bytes); err != nil {
			return err
		}
	}
	return nil
}

//计算指定目录的文件校验值，返回文件路径及其校验值列表
func CreateChecksumPathList(root string) (FileChecksumList, FileChecksumList, error) {
	listFile := FileChecksumList{}
	listDir := FileChecksumList{}

	walkFn := func(fullPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() == true {
			if isIgnoredDir(fullPath) == true {
				beego.Debug(fmt.Sprintf("目录(忽略)：%s ", fullPath))
				return nil
			} else {
				beego.Debug(fmt.Sprintf("目录：%s", fullPath))
				listDir = listDir.Add(NewFileChecksum(fullPath, ""))
			}
		} else {
			fileName := path.Base(fullPath)
			if isIgnoredFile(fileName) == true {
				beego.Debug(fmt.Sprintf("文件(忽略)：%s  指定忽略该文件名", fullPath))
				return nil
			}
			if isIgnoredDir(fullPath) == true {
				beego.Debug(fmt.Sprintf("文件(忽略)：%s  在忽略的文件夹中", fullPath))
				return nil
			}
			beego.Debug(fmt.Sprintf("文件：%s", fullPath))

			if checksum, err := createChecksumForFile(fullPath); err != nil {
				return err
			} else {
				listFile = listFile.Add(NewFileChecksum(fullPath, checksum))
			}
		}
		return nil
	}
	if err := filepath.Walk(root, walkFn); err != nil {
		return nil, nil, err
	}
	return listDir, listFile, nil
}

// func createFileChecksumList(root string) FileChecksumList{
// 	beego.Info(fmt.Sprintf("升级的应用目录：%s", root))
// 	if list, err := checksumPath(root); err != nil {
// 		beego.Error(fmt.Sprintf("%s", err))
// 	} else {
// 		// list.Print()
// 		G_FileList = list
// 	}
// }

func initCli() {
	cliApp := cli.NewApp()
	cliApp.Name = ""
	cliApp.Usage = "设置系统运行参数"
	cliApp.Version = "1.0.1"
	cliApp.Email = "ssor@qq.com"
	cliApp.Commands = []cli.Command{
		{
			Name:        "setversion",
			ShortName:   "sv",
			Usage:       "设置要生成的升级文档的版本号",
			Description: "根据实际应用升级情况填写",
			Action: func(c *cli.Context) {
				// fmt.Println(fmt.Sprintf("%#v", c.Command))
				// fmt.Println("-----------------------------")
				value := strings.ToLower(c.Args().First())
				if len(value) > 0 {
					beego.Info(fmt.Sprintf("设置版本号：%s", value))
					G_updateInfo.Version = value
				}
			},
		}, {
			// 	Name:        "appdir",
			// 	ShortName:   "ad",
			// 	Usage:       "设置升级应用的目录",
			// 	Description: "如果为空，则认为是当前目录的Bin文件夹",
			// 	Action: func(c *cli.Context) {
			// 		// fmt.Println(fmt.Sprintf("%#v", c.Command))
			// 		// fmt.Println("-----------------------------")
			// 		root := strings.ToLower(c.Args().First())
			// 		if len(root) <= 0 {
			// 			root = "Bin"
			// 		}
			// 		createFileChecksumList(root)
			// 	},
			// }, {
			Name:        "outputUpdateInfo",
			ShortName:   "ou",
			Usage:       "输出升级信息md文件",
			Description: "不需要更改信息时再输出文件，会覆盖之前的文件，文件名为 " + G_versionInfoFileName,
			Action: func(c *cli.Context) {
				// createVersionInfoFile()
				OutputVersionFile()
			},
		},
	}
	go func() {
		reader := bufio.NewReader(os.Stdin)
		for {
			fmt.Println("等待输入。。。")

			data, _, _ := reader.ReadLine()
			command := string(data)
			cliApp.Run(strings.Split(command, " "))
		}
	}()
	// app.Run(os.Args)
}

func initConfig() {
	var err error
	iniconf, err = config.NewConfig("ini", "conf/app.conf")
	if err != nil {
		beego.Error(err.Error())
	} else {
		//忽略特定文件
		ignoreFiles := iniconf.Strings("ignoredFiles")

		temp := []string{}
		for _, file := range ignoreFiles {
			if len(file) > 0 {
				temp = append(temp, file)
			}
		}
		ignoreFiles = temp
		if len(ignoreFiles) > 0 {
			IgnoreFileNameList = append(IgnoreFileNameList, ignoreFiles...)
			beego.Info(fmt.Sprintf("现有 %d 个忽略的文件", len(IgnoreFileNameList)))
			beego.Info("过滤文件名称如下：")
			for _, keyword := range ignoreFiles {
				beego.Debug(keyword)
			}
		} else {
			beego.Info("没有需要忽略的文件")
		}

		ignoreFolders := iniconf.Strings("ignoredFolders")
		temp = []string{}
		for _, folder := range ignoreFolders {
			if len(folder) > 0 {
				temp = append(temp, folder)
			}
		}
		ignoreFolders = temp
		if len(ignoreFolders) > 0 {
			G_ignoreFolderNameList = append(G_ignoreFolderNameList, ignoreFolders...)
			beego.Info(fmt.Sprintf("现有 %d 个忽略的文件夹", len(G_ignoreFolderNameList)))
			beego.Info("过滤的文件夹如下：")
			for _, keyword := range ignoreFolders {
				beego.Debug(keyword)
			}
		} else {
			beego.Info("没有需要忽略的文件夹")
		}

		dirSrc := iniconf.String("srcDir")
		if len(dirSrc) > 0 {
			G_dirSrc = dirSrc
		}
		beego.Info("源目录：" + G_dirSrc)

		dirDest := iniconf.String("destDir")
		if len(dirDest) > 0 {
			G_dirDest = dirDest
		}
		beego.Info("输出目录: " + G_dirDest)

		appVersion := iniconf.String("appVersion")
		if len(appVersion) > 0 {
			G_appVersion = appVersion
		}
		beego.Info("应用版本号: " + G_appVersion)

		// if locationCountTemp, err := iniconf.Int("locationCount"); err != nil {
		// 	DebugMust(err.Error() + GetFileLocation())

		// } else {
		// 	if locationCountTemp > 0 {
		// 		DebugInfo(fmt.Sprintf("获取货位数量：%d", locationCountTemp) + GetFileLocation())
		// 		locationCount = locationCountTemp
		// 	} else {
		// 		DebugInfo(fmt.Sprintf("设置货位数量为默认值：%d", locationCount) + GetFileLocation())

		// 	}
		// }

		// if err := iniconf.Set("locationCount", "23"); err != nil {
		// 	beego.Warn(err.Error())
		// }
		// if err := iniconf.SaveConfigFile("conf/app.conf"); err != nil {
		// 	beego.Warn(err.Error())
		// }
	}

}
func isIgnoredFile(name string) bool {
	for _, fileName := range IgnoreFileNameList {
		if fileName == name {
			return true
		}
	}

	return false
}
func isIgnoredDir(name string) bool {
	for _, dirName := range G_ignoreFolderNameList {
		if strings.Contains(name, G_dirSrc+dirName) {
			return true
		}
		// if name == G_dirSrc+dirName {
		// 	return true
		// }
	}
	return false
}

// createChecksumForFile returns the sha256 checksum for the given file
func createChecksumForFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	if bytes, err := CreateChecksumForReader(f); err != nil {
		return "", err
	} else {
		return fmt.Sprintf("%x", bytes), nil
	}
}

// CreateChecksumForReader returns the sha256 checksum for the entire
// contents of the given reader.
func CreateChecksumForReader(rd io.Reader) ([]byte, error) {
	h := sha256.New()
	if _, err := io.Copy(h, rd); err != nil {
		return nil, err
	}
	return h.Sum(nil), nil
}

// 检查文件或目录是否存在
// 如果由 filename 指定的文件或目录存在则返回 true，否则返回 false
func Exist(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil || os.IsExist(err)
}
func CopyFile(dstName, srcName string) error {
	src, err := os.Open(srcName)
	if err != nil {
		return err
	}
	defer src.Close()
	dst, err := os.OpenFile(dstName, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer dst.Close()
	if _, err := io.Copy(dst, src); err != nil {
		return err
	}
	return nil
}
