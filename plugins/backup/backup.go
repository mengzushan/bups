package backup

import (
	"archive/zip"
	"errors"
	"flag"
	"fmt"
	"github.com/abingzo/bups/common/config"
	"github.com/abingzo/bups/common/path"
	"github.com/abingzo/bups/common/plugin"
	"github.com/zbh255/bilog"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

/*
	收集备份数据的插件
*/

const (
	Name           = "backup"
	ScopeFilePath  = "file_path"
	ScopeDataBase  = "database"
	BackupFilePath = path.DEFAULT_PATH_BACK_UPCACHE + "/" + Name
	Type           = plugin.BCollect
)

var (
	support   = []uint32{plugin.SUPPORT_LOGGER, plugin.SUPPORT_CONFIG_OBJ, plugin.SUPPORT_ARGS}
	debugShow = false
)

// 数据库备份的参数
var mysqlDumpArgs = []string{
	"--host",
	"--port",
	"--user",
	"--password",
	"--databases",
	"--lock-tables",
}

func New() plugin.Plugin {
	return &Backup{}
}

type Backup struct {
	stdOut    io.Writer
	accessLog bilog.Logger
	errorLog  bilog.Logger
	stdLog    bilog.Logger
	cfg       *config.AutoGenerated
}

func (b *Backup) Caller(s plugin.Single) {
	b.stdLog.Info("Caller")
}

func (b *Backup) Start(args []string) {
	if args != nil || len(args) != 0 {
		os.Args = args
		debugIf := flag.Bool("debug", false, "是否开启调试模式")
		flag.Parse()
		debugShow = *debugIf
	}
	b.backupFile()
	b.backupDatabase()
}

// 备份文件
func (b *Backup) backupFile() {
	b.cfg.SetPluginScope(ScopeFilePath)
	b.cfg.RangePluginData(func(k string, v interface{}) {
		src, ok := v.(string)
		if !ok {
			panic("file path data type is not a string")
		}
		// 根据备份的目录名加配置选项名创建一个目标zip文件
		// Example: /User/harder/blog.harder.com -> ./cache/backup/blog.harder.com->root.zip
		srcSplit := strings.Split(src, "/")
		dstFile := fmt.Sprintf("%s/%s->%s.zip", BackupFilePath, srcSplit[len(srcSplit)-1], k)
		err := Zip(src, dstFile)
		// 归档为zip时出现错误则panic
		if err != nil {
			panic(err)
		}
	})
	// 打印一条备份成功的日志
	b.accessLog.Info("backup file complete")
}

// 备份数据库
// 目前只支持mysql driver
func (b *Backup) backupDatabase() {
	b.cfg.SetPluginScope(ScopeDataBase)
	args := make([]string, 0, 2)
	// 检查驱动
	switch b.cfg.PluginGetData("driver").(string) {
	case "mysql":
		args = append(args, "mysqldump")
		args = append(args, encodeMysqldumpArguments(b.cfg)...)
	default:
		panic(errors.New("no support database driver"))
	}
	// 创建存储备份的文件
	file, err := os.OpenFile(BackupFilePath+"/database.sql", os.O_CREATE|os.O_WRONLY, 0777)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	if debugShow {
		_, _ = b.stdOut.Write([]byte(strings.Join(args, " ")))
	}
	// 创建子进程去执行任务，并重定向它的输出以获得结果
	cmd := exec.Command(args[0])
	// 修改参数
	cmd.Args = args
	cmd.Stdout = file
	err = cmd.Run()
	if err != nil {
		panic(errors.New(err.Error() + fmt.Sprintf(" args: %v",cmd.Args)))
	}
	// 打印一条备份成功的日志
	b.accessLog.Info("backup database complete")
}

// 编码参数
// 配置值均为字符串，否则会引起类型断言失败panic
func encodeMysqldumpArguments(cfg *config.AutoGenerated) []string {
	args := make([]string, 0, 10)
	cfg.SetPluginScope(ScopeDataBase)
	for _, v := range mysqlDumpArgs {
		switch v {
		case "--host", "--port", "--user", "--password":
			args = append(args,fmt.Sprintf("%s=%s",v,cfg.PluginGetData(v[2:]).(string)))
		case "--databases":
			c := cfg.PluginGetData(v[2:]).([]interface{})
			strSlice := make([]string, len(c))
			for k, v := range c {
				strSlice[k] = fmt.Sprintf(" %s ",v.(string))
			}
			args = append(args, strSlice...)
		default:
			args = append(args, v)
		}
	}
	return args
}

func (b *Backup) GetName() string {
	return Name
}

func (b *Backup) GetType() plugin.Type {
	return Type
}

func (b *Backup) GetSupport() []uint32 {
	return support
}

func (b *Backup) SetSource(source *plugin.Source) {
	b.cfg = source.Config
	b.cfg.SetPluginName(Name)
	b.accessLog = source.AccessLog
	b.errorLog = source.ErrorLog
	b.stdLog = source.StdLog
}

// Zip srcFile could be a single file or a directory
// destZip必须为一个正确的文件路径，否则返回错误
func Zip(srcFile string, destZip string) error {
	zipfile, err := os.OpenFile(destZip, os.O_CREATE|os.O_WRONLY, 0777)
	if err != nil {
		return err
	}
	defer zipfile.Close()

	archive := zip.NewWriter(zipfile)
	defer archive.Close()

	return filepath.Walk(srcFile, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		header.Name = strings.TrimPrefix(path, filepath.Dir(srcFile)+"/")
		// header.Name = path
		if info.IsDir() {
			header.Name += "/"
		} else {
			header.Method = zip.Deflate
		}

		writer, err := archive.CreateHeader(header)
		if err != nil {
			return err
		}

		if !info.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()
			_, err = io.Copy(writer, file)
		}
		return err
	})
}
