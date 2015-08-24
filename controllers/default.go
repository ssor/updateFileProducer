package controllers

import (
	"bufio"
	"crypto/sha256"
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/astaxie/beego"
	"github.com/codegangsta/cli"
	"io"
	"os"
	"path/filepath"
	// "strconv"
	"io/ioutil"
	"path"
	"strings"
)

var (
	G_updateInfo          *UpdateInfo
	G_versionInfoFileName = "VersionInfo.md"
	// IgnoreFileNameList    = []string{}
	// G_ignoreFolderNameList = []string{}
	// G_dirSrc               = "./"
	// G_dirDest              = "./output/"
	// G_appVersion           = ""
	// iniconf                config.ConfigContainer = nil
)

//系统配置项
type Config struct {
	IgnoreFolderNameList, IgnoreFileNameList []string
	DirSrc, DirDest, AppVersion              string
}

func (this *Config) ListName() string {
	return "系统配置"
}
func (this *Config) InfoList() []string {
	list := []string{
		fmt.Sprintf("应用版本:     %s", this.AppVersion),
		fmt.Sprintf("应用源目录:   %s", this.DirSrc),
		fmt.Sprintf("应用输出目录: %s", this.DirDest),
		fmt.Sprintf("忽略的文件夹: %s", this.IgnoreFolderNameList),
		fmt.Sprintf("忽略的文件名: %s", this.IgnoreFileNameList),
	}
	return list
}

var (
	G_conf Config
)

func init() {
	initConfig()
	G_updateInfo = NewUpdateInfo(G_conf.AppVersion, FileChecksumList{}, FileChecksumList{})
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
	destVerFile := G_conf.DirDest + G_versionInfoFileName
	if Exist(destVerFile) == true { //将所有文件输出完成后，才能创建版本信息文件作为完成的标识，所以一开始不能存在该文件
		if err := os.Remove(destVerFile); err != nil {
			DebugSysF("删除输出目录的版本信息文件" + destVerFile + "出错：" + err.Error())
			return
		}
	}
	err := prepareVersionFileContent()
	if err != nil {
		DebugMustF("创建源目录文件列表出错：" + err.Error())
		return
	}
	err = copyFileToOutputDir()
	if err != nil {
		DebugMustF("向输出目录拷贝文件时出错：" + err.Error())
		return
	}

	if err := createVersionInfoFile(destVerFile); err != nil {
		DebugMustF("在输出目录创建版本信息文件出错：" + err.Error())
		return
	}
	beego.Warn("升级文件输出成功")
}
func prepareVersionFileContent() error {
	dirList, fileList, err := CreateChecksumPathList(G_conf.DirSrc)
	if err != nil {
		return err
	}
	G_updateInfo.FileList = fileList
	G_updateInfo.DirList = dirList
	return nil
}
func copyFileToOutputDir() error {
	//对比对应Bin目录的文件，进行同步

	//同步目录
	// outputBinDirList := []string{}
	count := 0
	for _, file := range G_updateInfo.DirList {
		// destDir := strings.Replace(file.Path, G_dirSrc, G_dirDest+"Bin/", 1)
		destDir := G_conf.DirDest + "Bin/" + file.Path
		if Exist(destDir) == false {
			if err := os.MkdirAll(destDir, os.ModePerm); err != nil {
				return err
			}
			beego.Info("创建了目录：" + destDir)
			count += 1
		}
	}
	beego.Warn(fmt.Sprintf("共创建了 %d 个新目录", count))

	copiedFileNameList := []string{}
	// 同步文件
	for _, file := range G_updateInfo.FileList {
		// destDir := strings.Replace(file.Path, G_dirSrc, G_dirDest+"Bin/", 1)
		destDir := G_conf.DirDest + "Bin/" + file.Path
		dirSrc := G_conf.DirSrc + file.Path
		if Exist(destDir) == false {
			if err := CopyFile(destDir, dirSrc); err != nil {
				return err
			}
			beego.Info("复制了文件：" + destDir)
			copiedFileNameList = append(copiedFileNameList, destDir)
		} else {
			if checksum, err := createChecksumForFile(destDir); err != nil {
				return err
			} else {
				if checksum == file.Checksum {
					// beego.Trace("源文件与目标文件相同，不需要复制 " + file.Path)
				} else {
					beego.Info("源文件存在，但是文件发生了变化，需要首先删除源文件")
					if err := os.Remove(destDir); err != nil {
						return err
					} else {
						if err := CopyFile(destDir, dirSrc); err != nil {
							return err
						}
						beego.Info("复制了文件：" + destDir)
						copiedFileNameList = append(copiedFileNameList, destDir)
					}
				}
			}
		}
	}
	beego.Warn(fmt.Sprintf("共复制了 %d 个文件", len(copiedFileNameList)))
	for _, file := range copiedFileNameList {
		beego.Debug(file)

	}

	return nil
}
func createVersionInfoFile(versionInfoFileFullPath string) error {
	ui := G_updateInfo
	// ui.Print()

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
				// beego.Debug(fmt.Sprintf("目录(忽略)：%s ", fullPath))
				return nil
			} else {
				// beego.Debug(fmt.Sprintf("目录：%s", fullPath))
				pathTrimed := strings.Replace(fullPath, G_conf.DirSrc, "", 1)
				listDir = listDir.Add(NewFileChecksum(pathTrimed, ""))
			}
		} else {
			fileName := path.Base(fullPath)
			if isIgnoredFile(fileName) == true {
				beego.Debug(fmt.Sprintf("文件(忽略)：%s  指定忽略该文件名", fullPath))
				return nil
			}
			if isIgnoredDir(fullPath) == true {
				// beego.Debug(fmt.Sprintf("文件(忽略)：%s  在忽略的文件夹中", fullPath))
				return nil
			}
			// beego.Debug(fmt.Sprintf("文件：%s", fullPath))

			if checksum, err := createChecksumForFile(fullPath); err != nil {
				return err
			} else {
				pathTrimed := strings.Replace(fullPath, G_conf.DirSrc, "", 1)
				listFile = listFile.Add(NewFileChecksum(pathTrimed, checksum))
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
// 		DebugMustF(fmt.Sprintf("%s", err))
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
			Name:        "try",
			ShortName:   "try",
			Usage:       "查看将要输出的目录和文件",
			Description: "不会真正创建和复制文件",
			Action: func(c *cli.Context) {
				err := prepareVersionFileContent()
				if err != nil {
					DebugMustF("创建源目录文件列表出错：" + err.Error())
					return
				}
				G_updateInfo.Print()
			},
		}, {
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

func initConfig() error {
	confFile := "conf/sys.toml"
	if confData, err := ioutil.ReadFile(confFile); err != nil {
		DebugMustF("系统配置出错：%s", err.Error())
		return err
	} else {
		if _, err := toml.Decode(string(confData), &G_conf); err != nil {
			DebugMustF("系统配置出错：%s", err.Error())
			return err
		}
		DebugPrintList_Info(&G_conf)
	}

	return nil

	// var err error
	// iniconf, err = config.NewConfig("ini", "conf/app.conf")
	// if err != nil {
	// 	DebugMustF(err.Error())
	// } else {
	// 	//忽略特定文件
	// 	ignoreFiles := iniconf.Strings("ignoredFiles")

	// 	temp := []string{}
	// 	for _, file := range ignoreFiles {
	// 		if len(file) > 0 {
	// 			temp = append(temp, file)
	// 		}
	// 	}
	// 	ignoreFiles = temp
	// 	if len(ignoreFiles) > 0 {
	// 		IgnoreFileNameList = append(IgnoreFileNameList, ignoreFiles...)
	// 		beego.Info(fmt.Sprintf("现有 %d 个忽略的文件", len(IgnoreFileNameList)))
	// 		beego.Info("过滤文件名称如下：")
	// 		for _, keyword := range ignoreFiles {
	// 			beego.Debug(keyword)
	// 		}
	// 	} else {
	// 		beego.Info("没有需要忽略的文件")
	// 	}

	// 	ignoreFolders := iniconf.Strings("ignoredFolders")
	// 	temp = []string{}
	// 	for _, folder := range ignoreFolders {
	// 		if len(folder) > 0 {
	// 			temp = append(temp, folder)
	// 		}
	// 	}
	// 	ignoreFolders = temp
	// 	if len(ignoreFolders) > 0 {
	// 		G_ignoreFolderNameList = append(G_ignoreFolderNameList, ignoreFolders...)
	// 		beego.Info(fmt.Sprintf("现有 %d 个忽略的文件夹", len(G_ignoreFolderNameList)))
	// 		beego.Info("过滤的文件夹如下：")
	// 		for _, keyword := range ignoreFolders {
	// 			beego.Debug(keyword)
	// 		}
	// 	} else {
	// 		beego.Info("没有需要忽略的文件夹")
	// 	}

	// 	dirSrc := iniconf.String("srcDir")
	// 	if len(dirSrc) > 0 {
	// 		G_dirSrc = dirSrc
	// 	}
	// 	beego.Info("源目录：" + G_dirSrc)

	// 	dirDest := iniconf.String("destDir")
	// 	if len(dirDest) > 0 {
	// 		G_dirDest = dirDest
	// 	}
	// 	beego.Info("输出目录: " + G_dirDest)

	// 	appVersion := iniconf.String("appVersion")
	// 	if len(appVersion) > 0 {
	// 		G_appVersion = appVersion
	// 	}
	// 	beego.Info("应用版本号: " + G_appVersion)
	// }

}

func isIgnoredFile(name string) bool {
	for _, fileName := range G_conf.IgnoreFileNameList {
		if fileName == name {
			return true
		}
	}

	return false
}
func isIgnoredDir(name string) bool {
	for _, dirName := range G_conf.IgnoreFolderNameList {
		if strings.Contains(name, G_conf.DirSrc+dirName) {
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
