package upload

import (
	"context"
	"flag"
	"fmt"
	"github.com/abingzo/bups/common/config"
	"github.com/abingzo/bups/common/logger"
	"github.com/abingzo/bups/common/path"
	"github.com/abingzo/bups/common/plugin"
	"github.com/tencentyun/cos-go-sdk-v5"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"time"
)

const (
	Name           = "upload"
	DownloadCached = path.DEFAULT_PATH_BACK_UPCACHE + "/download"
	BackUpFilePath = path.DEFAULT_PATH_BACK_UPCACHE + "/encrypt/backup.zip"
	Type           = plugin.BCallBack
)

var Support = []uint32{
	plugin.SUPPORT_LOGGER,
	plugin.SUPPORT_CONFIG_OBJ,
	plugin.SUPPORT_ARGS,
}

func New() plugin.Plugin {
	return &Upload{
		Name:       Name,
		Type:       Type,
		Support:    Support,
		cosElement: nil,
	}
}

func InitCosElement(u *Upload) {
	// 初始化实例
	u.cosElement = &CosElement{}
	cfg := u.conf
	cfg.SetPluginName(u.Name)
	cfg.SetPluginScope("cos")
	// 设置属性
	u.cosElement.sId = cfg.PluginGetData("sId").(string)
	u.cosElement.sKey = cfg.PluginGetData("sKey").(string)
	u.cosElement.bucketUrl = cfg.PluginGetData("bucketUrl").(string)
	u.cosElement.serviceUrl = cfg.PluginGetData("serviceUrl").(string)
	// 连接服务端
	bu, _ := url.Parse(u.cosElement.bucketUrl)
	bsu, _ := url.Parse(u.cosElement.serviceUrl)
	bucket := cos.BaseURL{
		BucketURL:  bu,
		ServiceURL: bsu,
	}
	client := cos.NewClient(&bucket, &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:  u.cosElement.sId,
			SecretKey: u.cosElement.sKey,
		},
	})
	u.cosElement.client = client
}

/*
	配置文件选项:plugin.upload.cos
	基础的上传至Cos的接口，提供上传，下载，检索
*/

type CosElement struct {
	client     *cos.Client
	sId        string
	sKey       string
	bucketUrl  string
	serviceUrl string
}

func (c *CosElement) Push(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	// 根据备份的时间取名
	fileName := time.Now().Format("2006-01-02-15-04") + ".zip"
	_, err = c.client.Object.Put(context.Background(), fileName, file, nil)
	if err != nil {
		return err
	}
	return nil
}

func (c *CosElement) Download(fileName string) ([]byte, error) {
	res, err := c.client.Object.Get(context.Background(), fileName, nil)
	if err != nil {
		return nil, err
	}
	file, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	return file, nil
}

func (c *CosElement) Delete(fileName string) error {
	_, err := c.client.Object.Delete(context.Background(), fileName)
	if err != nil {
		return err
	}
	return nil
}

func (c *CosElement) Search() {}

type Upload struct {
	plugin.Plugin
	Name       string
	Type       plugin.Type
	Support    []uint32
	conf       *config.AutoGenerated
	stdLog     logger.Logger
	accessLog  logger.Logger
	errorLog   logger.Logger
	cosElement *CosElement
}

func (u *Upload) SetSource(source *plugin.Source) {
	u.stdLog = source.StdLog
	u.errorLog = source.ErrorLog
	u.accessLog = source.AccessLog
	u.conf = source.Config
}

func (u *Upload) GetName() string {
	return u.Name
}

func (u *Upload) GetType() plugin.Type {
	return u.Type
}

func (u *Upload) GetSupport() []uint32 {
	return u.Support
}

func (u *Upload) Caller(s plugin.Single) {
	u.accessLog.Info(Name + ".Caller")
}

// Start 启动函数
func (u *Upload) Start(args []string) {
	// 初始化实例
	if u.cosElement == nil {
		InitCosElement(u)
	}
	if args == nil || len(args) == 0 {
		// 上传尝试3次
		for i := 0 ; i < 3; i++{
			err := u.cosElement.Push(BackUpFilePath)
			if err == nil {
				break
			} else {
				u.errorLog.Error(err.Error())
			}
		}
		// 上传成功则打印日志
		u.accessLog.Info("upload cos successfully")
		return
	} else {
		os.Args = args
	}
	downloadFileName := flag.String("download", "", "需要下载的文件名")
	searchFileName := flag.String("search", "", "需要搜索的文件名")
	if *downloadFileName != "" {
		bytes, err := u.cosElement.Download(*downloadFileName)
		if err != nil {
			panic(err)
		}
		err = ioutil.WriteFile(DownloadCached+"/"+*downloadFileName, bytes, 0755)
		if err != nil {
			panic(err)
		}
		// 打印消息
		u.stdLog.Debug(fmt.Sprintf("%s 下载成功\n", *downloadFileName))
		u.accessLog.Info(fmt.Sprintf("%s 下载成功\n", *downloadFileName))
	} else if *searchFileName != "" {

	}
}
