package main

import (
	"flag"
	"github.com/abingzo/bups/common/config"
	"github.com/abingzo/bups/common/logger"
	"github.com/abingzo/bups/common/plugin"
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
	"os"
)

const (
	Name = "web_config"
	Type = plugin.Init
)

// 插件需要的支持
var support = []int{plugin.SupportNativeStdout, plugin.SupportArgs, plugin.SupportConfigWrite, plugin.SupportConfigRead}

func New() plugin.Plugin {
	return &WebConfig{
		name:    Name,
		typ:     Type,
		support: support,
	}
}

type WebConfig struct {
	stdOut     io.Writer
	logOut     logger.Logger
	name       string
	typ        plugin.Type
	support    []int
	confReader io.Reader
	confWriter io.Writer
	plugin.Plugin
}

func (w *WebConfig) SetStdout(out io.Writer) {
	w.stdOut = out
}

func (w *WebConfig) SetLogOut(out logger.Logger) {
	w.logOut = out
}

func (w *WebConfig) GetName() string {
	return w.name
}

func (w *WebConfig) GetType() plugin.Type {
	return w.typ
}

func (w *WebConfig) GetSupport() []int {
	return w.support
}

func (w *WebConfig) ConfRead(reader io.Reader) {
	w.confReader = reader
}

func (w *WebConfig) ConfWrite(writer io.Writer) {
	w.confWriter = writer
}

// Start 启动函数
func (w *WebConfig) Start(args []string) {
	os.Args = args
	_ = flag.CommandLine.Parse(args)
	// 处理参数
	sw := flag.String("switch", "off", "web_config的开关")
	bind := flag.String("bind", "127.0.0.1:8080", "web_config绑定的ip&port")
	flag.Parse()
	if *sw == "off" {
		_, _ = w.stdOut.Write([]byte("off\n"))
		w.logOut.Info("off")
		return
	}
	gin.DefaultWriter = w.stdOut
	r := gin.Default()
	r.GET("/config", func(context *gin.Context) {
		cfg := config.Read(w.confReader)
		context.JSON(http.StatusOK, cfg)
	})
	r.POST("/config", func(context *gin.Context) {
		json := &config.AutoGenerated{}
		err := context.BindJSON(json)
		if err != nil {
			context.JSON(http.StatusBadRequest, gin.H{
				"msg": "参数错误",
			})
			return
		}
		err = config.Write(w.confWriter, json)
		if err != nil {
			context.JSON(http.StatusOK, gin.H{
				"msg": "写入配置文件失败",
			})
			return
		}
		context.JSON(http.StatusOK, gin.H{
			"msg": "写入配置文件成功",
		})
	})
	r.Run(*bind)
}
