package lib

import (
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

var Logger = &FileLogger{}

func NewLogger(logpath, name string, level int) *FileLogger {
	ins := &FileLogger{}
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

type FileLogger struct {
	loggerMap map[string]*log.Logger
	curDate   map[string]string
	logPath   string
	logName   string
	logLevel  int
	lock      sync.RWMutex
}

func (this *FileLogger) Init(logpath, logname string, loglevel int) { // {{{
	this.logPath = logpath
	this.logName = this.reviseLogName(logname)
	this.curDate = make(map[string]string)
	this.loggerMap = make(map[string]*log.Logger)
	this.logLevel = loglevel

	os.MkdirAll(this.logPath, 0777)
} // }}}

//格式化文件名
func (this *FileLogger) reviseLogName(logname string) string { // {{{
	if l := len(logname); l < 4 || logname[l-4:] != ".log" {
		logname = logname + ".log"
	}

	return logname
} // }}}

func (this *FileLogger) getLogger(logname string) (*log.Logger, error) { // {{{
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
			this.loggerMap[logname] = log.New(fd, "", 0)
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

func (this *FileLogger) writeLog(logname string, v ...interface{}) { // {{{
	go this._writeLog(logname, v...)
} // }}}

func (this *FileLogger) _writeLog(logname string, v ...interface{}) { // {{{
	logger, err := this.getLogger(logname)
	if err != nil {
		fmt.Println("log failed", err)
		return
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
	timeNow := time.Now().Format("2006-01-02 15:04:05") //go的坑，必须是2006-01-02 15:04:05
	logger.Printf("time[%s] %s\n", timeNow, msgstr)
} // }}}

func (this *FileLogger) Debug(v ...interface{}) { // {{{
	if this.logLevel&LevelDebug == 0 {
		return
	}

	fmt.Printf("debug:")
	for _, val := range v {
		fmt.Printf(" %#v ", val)
	}
	fmt.Println("")
	this.writeLog(this.logName+".debug", v...)
} // }}}

func (this *FileLogger) Info(v ...interface{}) { // {{{
	if this.logLevel&LevelInfo == 0 {
		return
	}
	this.writeLog(this.logName+".info", v...)
} // }}}

func (this *FileLogger) Access(v ...interface{}) { // {{{
	if this.logLevel&LevelAccess == 0 {
		return
	}
	this.writeLog(this.logName, v...)
} // }}}

func (this *FileLogger) Warn(v ...interface{}) { // {{{
	if this.logLevel&LevelWarn == 0 {
		return
	}
	this.writeLog(this.logName+".warn", v...)
} // }}}

func (this *FileLogger) Error(v ...interface{}) { // {{{
	if this.logLevel&LevelError == 0 {
		return
	}
	this.writeLog(this.logName+".error", v...)
} // }}}

func (this *FileLogger) Other(logname string, v ...interface{}) { // {{{
	this.writeLog(this.reviseLogName(logname), v...)
} // }}}
