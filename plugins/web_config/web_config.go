package web_config

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
var support = []uint32{
	plugin.SUPPORT_STDLOG,
	plugin.SUPPORT_ARGS,
	plugin.SUPPORT_RAW_CONFIG,
}

func New() plugin.Plugin {
	return &WebConfig{
		name:    Name,
		typ:     Type,
		support: support,
	}
}

type WebConfig struct {
	stdLog     logger.Logger
	name       string
	typ        plugin.Type
	support    []uint32
	confReader io.Reader
	confWriter io.Writer
	plugin.Plugin
}

func (w *WebConfig) SetSource(source *plugin.Source) {
	w.stdLog = source.StdLog
	w.confReader = source.RawConfig
	w.confWriter = source.RawConfig
}

func (w *WebConfig) GetName() string {
	return w.name
}

func (w *WebConfig) GetType() plugin.Type {
	return w.typ
}

func (w *WebConfig) GetSupport() []uint32 {
	return w.support
}


func (w *WebConfig) Caller(s plugin.Single) {
	w.stdLog.Info(Name + ".Caller")
}

// Start 启动函数
func (w *WebConfig) Start(args []string) {
	// args不为nil时代表参数启动
	if args == nil {
		return
	}
	os.Args = args
	_ = flag.CommandLine.Parse(args)
	// 处理参数
	sw := flag.String("switch", "off", "web_config的开关")
	bind := flag.String("bind", "127.0.0.1:8080", "web_config绑定的ip&port")
	flag.Parse()
	if *sw == "off" {
		w.stdLog.Info("off")
		return
	}
	gin.DefaultWriter = os.Stdout
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
