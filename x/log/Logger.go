package log

import (
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

func NewLogger(logpath, name string, level int) *Logger {
	ins := &Logger{}
	ins.Init(logpath, name, level)

	return ins
}

// Log levels
const (
	LevelNone   = 0x00
	LevelError  = 0x01
	LevelWarn   = 0x02
	LevelAccess = 0x04
	LevelInfo   = 0x08
	LevelDebug  = 0x10
	LevelAll    = 0xFF
)

type Logger struct {
	loggerMap map[string]*log.Logger
	curDate   map[string]string
	logPath   string
	logName   string
	logLevel  int
	output    io.Writer
	prefix    string
	lock      sync.RWMutex
}

func (this *Logger) Init(logpath, logname string, loglevel int) { // {{{
	this.logPath = logpath
	this.logName = this.reviseLogName(logname)
	this.curDate = make(map[string]string)
	this.loggerMap = make(map[string]*log.Logger)
	this.logLevel = loglevel

	os.Setenv("YGO_LOG_PATH", this.logPath)
	os.Setenv("YGO_LOG_LEVEL", strconv.Itoa(this.logLevel))

	os.MkdirAll(this.logPath, 0777)
} // }}}

//格式化文件名
func (this *Logger) reviseLogName(logname string) string { // {{{
	if l := len(logname); l < 4 || logname[l-4:] != ".log" {
		logname = logname + ".log"
	}

	return logname
} // }}}

func (this *Logger) getLogger(levelname, logname string) (*log.Logger, error) { // {{{
	if logname == "" {
		logname = this.logName
		if levelname != "access" {
			logname += "." + levelname
		}
	}

	nowDate := time.Now().Format("20060102")
	filePath := this.logPath + "/" + logname + "." + nowDate
	this.lock.RLock()
	retLogger, ok := this.loggerMap[logname]
	curDate, ok := this.curDate[logname]
	if !ok || nowDate != curDate {
		this.lock.RUnlock()
		this.lock.Lock()
		defer this.lock.Unlock()

		retLoggerRetry, ok := this.loggerMap[logname]
		curDateRetry, ok := this.curDate[logname]
		//双重判断，减少抢锁
		if !ok || nowDate != curDateRetry {
			fd, err := os.OpenFile(filePath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0777)
			if err != nil {
				return nil, err
			}
			//创建文件的时候指定777权限不管用，所有只能在显式chmod
			fd.Chmod(0777)
			this.loggerMap[logname] = log.New(fd, this.prefix, 0)
			this.curDate[logname] = nowDate
			fmt.Println("new logger:", filePath)

			retLogger = this.loggerMap[logname]
		} else {
			retLogger = retLoggerRetry
		}
	} else {
		this.lock.RUnlock()
	}

	return retLogger, nil
} // }}}

func (this *Logger) writeLog(levelname, logname string, v ...interface{}) { // {{{
	go this._writeLog(levelname, logname, v...)
} // }}}

//指定输出
func (this *Logger) SetOutput(w io.Writer) { // {{{
	this.output = w
} // }}}

//指定前缀
func (this *Logger) SetPrefix(p string) { // {{{
	this.prefix = p
} // }}}

func (this *Logger) _writeLog(levelname, logname string, v ...interface{}) { // {{{
	var logger *log.Logger
	var err error

	if this.output != nil {
		logger = log.New(this.output, this.prefix, 0)
	} else {
		logger, err = this.getLogger(levelname, logname)
		if err != nil {
			fmt.Println("log failed", err)
			return
		}
	}

	msgstr := ""
	for _, msg := range v {
		if msg1, ok := msg.(map[string]interface{}); ok {
			//map每次输出的顺序是随机的，以下保证每次输出的顺序一致，如果map比较大，可能有一定性能损耗
			var keys []string
			for k := range msg1 {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				msgstr = msgstr + fmt.Sprintf("%s[%+v] ", k, msg1[k])
			}
		} else {
			msgstr = msgstr + fmt.Sprintf("%+v ", msg)
		}
	}
	msgstr = strings.TrimRight(msgstr, ",")
	timeNow := time.Now().Format("06-01-02 15:04:05")
	logger.Printf("%s: %s %s\n", levelname, timeNow, msgstr)
} // }}}

func (this *Logger) Debug(v ...interface{}) { // {{{
	if this.logLevel&LevelDebug == 0 {
		return
	}

	fmt.Printf("debug:")
	for _, val := range v {
		fmt.Printf(" %#v ", val)
	}
	fmt.Println("")
	this.writeLog("debug", "", v...)
} // }}}

func (this *Logger) Info(v ...interface{}) { // {{{
	if this.logLevel&LevelInfo == 0 {
		return
	}
	this.writeLog("info", "", v...)
} // }}}

func (this *Logger) Access(v ...interface{}) { // {{{
	if this.logLevel&LevelAccess == 0 {
		return
	}
	this.writeLog("access", "", v...)
} // }}}

func (this *Logger) Warn(v ...interface{}) { // {{{
	if this.logLevel&LevelWarn == 0 {
		return
	}
	this.writeLog("warn", "", v...)
} // }}}

func (this *Logger) Error(v ...interface{}) { // {{{
	if this.logLevel&LevelError == 0 {
		return
	}
	this.writeLog("error", "", v...)
} // }}}

func (this *Logger) Other(logname string, v ...interface{}) { // {{{
	this.writeLog(logname, this.reviseLogName(logname), v...)
} // }}}
