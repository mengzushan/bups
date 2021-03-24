package web

import (
	"github.com/gin-gonic/gin"
	"github.com/mengzushan/bups/common/conf"
	"github.com/mengzushan/bups/common/errors"
	"github.com/mengzushan/bups/utils"
	"io"
	"net/http"
	"os"
	"reflect"
)

func Run() {
	// 为Gin创建日志文件
	pathHead, _ := os.Getwd()
	file, _ := os.Create(pathHead + "/log/gin.log")
	gin.DefaultWriter = io.MultiWriter(file)
	// webui
	r := gin.Default()

	config := utils.GetConfig()
	// Group using gin.BasicAuth() middleware
	// gin.Accounts is a shortcut for map[string]string
	authorized := r.Group("/admin", gin.BasicAuth(gin.Accounts{
		config.WebConfig.UserName: config.WebConfig.UserPasswd,
	}))

	// /admin/secrets endpoint
	// hit "localhost:8080/admin/secrets
	authorized.GET("/config", ReturnConfigInfo)
	authorized.POST("/config", SetConfig)
	// Listen and serve on 0.0.0.0:8080
	_ = r.Run(config.WebConfig.Ipaddr + config.WebConfig.Port)
}

type BindJson struct {
	CloudAPI     string `json:"CloudAPI"`
	SaveName     string `json:"SaveName"`
	SaveTime     int    `json:"SaveTime"`
	SaveTimePass int    `json:"SaveTimePass"`
	Bucket       struct {
		BucketURL string `json:"BucketURL"`
		Secretid  string `json:"Secretid"`
		Secretkey string `json:"Secretkey"`
		Token     string `json:"Token"`
	} `json:"Bucket"`
	Database struct {
		Ipaddr     string `json:"Ipaddr"`
		Port       string `json:"Port"`
		UserName   string `json:"UserName"`
		UserPasswd string `json:"UserPasswd"`
		DbName     string `json:"DbName"`
		DbName2    string `json:"DbName2"`
	} `json:"Database"`
	Local struct {
		Web    string `json:"Web"`
		Static string `json:"Static"`
		Log    string `json:"Log"`
	} `json:"local"`
	WebConfig struct {
		Switch     string `json:"Switch"`
		Ipaddr     string `json:"Ipaddr"`
		Port       string `json:"Port"`
		UserName   string `json:"UserName"`
		UserPasswd string `json:"UserPasswd"`
	} `json:"WebConfig"`
	Encryption struct {
		Switch      string `json:"Switch"`
		EncryptMode string `json:"EncryptMode"`
		Aes         string `json:"Aes"`
	} `json:"Encryption"`
	Rsa struct {
		PubKey string `json:"PubKey"`
		PriKey string `json:"PriKey"`
	} `json:"Rsa"`
}

func ReturnConfigInfo(c *gin.Context) {
	config := utils.GetConfig()

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "ok",
		"data":    config,
	})
}

func SetConfig(c *gin.Context) {
	json := BindJson{}
	_ = c.BindJSON(&json)
	// 遍历结构体{BindJson}并将其值赋值到{AutoGenerated}
	// 之后根据Tag写入toml配置文件
	// 传入指针类型设置其值
	var tomlConf = conf.AutoGenerated{}
	_ = setValue(&json, &tomlConf)
	// 写入配置
	err := utils.SaveTomlConfig(&tomlConf)
	if err != nil {
		c.JSON(errors.ErrSaveTomlFileNot.HttpCode, gin.H{
			"code":    errors.ErrSaveTomlFileNot.Code,
			"message": errors.ErrSaveTomlFileNot.Message,
		})
		panic(err)
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "ok",
		"data":    tomlConf,
	})
}

func setValue(json *BindJson, tomlConf *conf.AutoGenerated) error {
	// 因为传入的都是指针类型,使用Elem获得其指向的值
	value1 := reflect.ValueOf(json).Elem()
	value2 := reflect.ValueOf(tomlConf).Elem()
	// 递归函数解决结构体嵌套
	setSubStructValue(&value1, &value2)
	return nil
}

func setSubStructValue(value1 *reflect.Value, value2 *reflect.Value) {
	// 循环检测字段，遇到结构体类型开始递归循环
	for i := 0; i < value2.NumField(); i++ {
		val1 := value1.Field(i)
		val2 := value2.Field(i)
		switch val2.Kind() {
		case reflect.String:
			val2.SetString(val1.String())
		case reflect.Int:
			val2.SetInt(val1.Int())
		case reflect.Struct:
			setSubStructValue(&val1, &val2)
		}
	}
}
