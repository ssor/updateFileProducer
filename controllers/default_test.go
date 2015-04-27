package controllers

import (
	// "net/http"
	// "net/http/httptest"
	// "path/filepath"
	// "runtime"
	"testing"
	// _ "updateFileProducer/routers"

	// "github.com/astaxie/beego"
	// . "github.com/smartystreets/goconvey/convey"
	"fmt"
)

func init() {

	// _, file, _, _ := runtime.Caller(1)
	// apppath, _ := filepath.Abs(filepath.Dir(filepath.Join(file, ".." + string(filepath.Separator))))
	// beego.TestBeegoInit(apppath)
}

func TestChecksumPath(t *testing.T) {
	if list, err := ChecksumPath("../testApp"); err != nil {
		t.Fatalf("%s", err)
	} else {
		list.Print()
	}
}

// TestMain is a sample to run an endpoint test
func TestChecksumForFile(t *testing.T) {
	// src := "e875492c7bc06833801186a8711421cae85afe8d08b3ad2def1468944ccee64d"
	// t.Log("1111")
	// t.FailNow()
	bytes, err := ChecksumForFile("../testApp/wangceshi.md")
	if err != nil {
		// t.Log(err.Error())
		// t.FailNow()
		t.Fatalf("%s", err)
	}
	fmt.Println(fmt.Sprintf("%x", bytes))
	// fmt.Println(src)
	// fmt.Println(src == fmt.Sprintf("%x", bytes))
	// t.FailNow()
}
