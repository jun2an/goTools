package publicLog

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var timestampFormat = "2006-01-02 15:04:05"
var nowFunc = time.Now
var defaultInputChanSize = 1024 * 32
var ip, logPathMd5 = "", ""
var opkeyMap map[string]int64 = make(map[string]int64)
var mutex sync.Mutex
var baseNum int64 = -1
var pidHex string

var defaultOfflineFileLog *OfflineFileLog

type OfflineFileLog struct {
	fs       *os.File
	input    chan []byte
	dir      string
	fileName string

	mu       *sync.Mutex
	curSize  uint64
	curIndex int64
	curHour  string

	rotateCount  int64
	rotateSize   uint64
	rotateByHour bool
	keepHours    int64
	close        chan struct{}
	wg           *sync.WaitGroup
}

func init() {
	ip = getIp()
	pidHex = fmt.Sprintf("%04x", os.Getpid())
}

func NewOfflineFileLog(dir string, fileName string) (*OfflineFileLog, error) {
	fl := &OfflineFileLog{
		close:    make(chan struct{}),
		mu:       new(sync.Mutex),
		wg:       new(sync.WaitGroup),
		fileName: fileName,
		input:    make(chan []byte, defaultInputChanSize),
		fs:       new(os.File),
		curHour:  getCurHour(nowFunc()),
	}

	fl.dir = dir

	_, err := os.Lstat(fl.dir)
	if err != nil {
		return nil, err
	}

	fl.genFile(fileName)
	go fl.writeLoop()
	return fl, nil
}

func InitOfflineLog(dir string, fileName string) {
	defaultOfflineFileLog, _ = NewOfflineFileLog(dir, fileName)
	if strings.HasSuffix(dir, "/") {
		dir = dir[0 : len(dir)-1]
	}
	data := []byte(dir + "/" + fileName + ".log")
	has := md5.Sum(data)
	logPathMd5 = fmt.Sprintf("%x", has)[16:24]
}

func (f *OfflineFileLog) Close() {
	close(f.close)
	f.wg.Wait()
	f.closeAll()
}

func (f *OfflineFileLog) closeAll() {
	f.fs.Close()
}

func (f *OfflineFileLog) writeLoop() {
	f.wg.Add(1)
	defer f.wg.Done()
	for {
		select {
		case r := <-f.input:
			f.write(r)
		case <-f.close:
			return
		}
	}
}

func (f *OfflineFileLog) write(r []byte) {
	cur := getCurHour(nowFunc())
	if cur != f.curHour {
		f.mu.Lock()
		f.curHour = cur
		f.rotate(f.fileName, f.fs)
		f.mu.Unlock()
	}
	f.fs.Write(r)
}

func (f *OfflineFileLog) rotate(prefix string, fd *os.File) {
	fd.Sync()
	fd.Close()
	f.genFile(prefix)
}

func (f *OfflineFileLog) genFile(prefix string) {
	fileName := fmt.Sprintf("%s.log.%s", prefix, getCurHour(nowFunc()))
	fileName = filepath.Join(f.dir, fileName)
	fd, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "openFile err:%v", err)
		return
	}
	f.fs = fd
}

func getCurHour(cur time.Time) string {
	return fmt.Sprintf("%04d%02d%02d%02d", cur.Year(), cur.Month(), cur.Day(), cur.Hour())
}

func (f *OfflineFileLog) writePublic(log bytes.Buffer) {
	f.write(log.Bytes())
}

func (f *OfflineFileLog) appendPLid(opera_stat_key string, log *bytes.Buffer) {
	PLid := bytes.Buffer{}
	hexIp := hex.EncodeToString(net.ParseIP(ip).To4())
	var opkey int64 = -1
	mutex.Lock()
	if baseNum == -1 {
		baseNum = int64(rand.Intn(256))
	}
	if _, ok := opkeyMap[opera_stat_key]; ok {
		opkeyMap[opera_stat_key] += 256
	} else {
		opkeyMap[opera_stat_key] = 0
	}
	opkey = opkeyMap[opera_stat_key]
	opkey += baseNum
	mutex.Unlock()

	PLid.WriteString(hexIp)
	PLid.WriteString(logPathMd5)
	opkeyStr := fmt.Sprintf("%x", opkey)
	if len(opkeyStr) == 1 {
		PLid.WriteString("0")
	}
	PLid.WriteString(opkeyStr)
	PLid.WriteString(pidHex)
	PLid.WriteString("x")
	log.WriteString("PLid=")
	log.Write(PLid.Bytes())
}

func (f *OfflineFileLog) TagLogPublic(opera_stat_key string, kv ...interface{}) {
	if len(kv)%2 != 0 || len(kv) == 0 {
		return
	}

	needTimeStamp := true

	split := "||"
	log := bytes.Buffer{}
	log.WriteString(opera_stat_key)
	for i := 0; i < len(kv); i += 2 {
		log.WriteString(split)
		log.WriteString(fmt.Sprintf("%v", kv[i]))
		if kv[i] == "timestamp" {
			needTimeStamp = false
		}
		log.WriteString("=")
		log.WriteString(fmt.Sprintf("%v", kv[i+1]))
	}

	log.WriteString(split)
	f.appendPLid(opera_stat_key, &log)
	if needTimeStamp {
		log.WriteString(split)
		log.WriteString("timestamp=")
		log.WriteString(time.Now().Format(timestampFormat))
	}
	log.WriteString("\n")
	f.writePublic(log)
}

func (f *OfflineFileLog) LogPublic(opera_stat_key string, kv map[string]interface{}) {
	needTimeStamp := true

	split := "||"
	log := bytes.Buffer{}
	log.WriteString(opera_stat_key)
	for k, v := range kv {
		log.WriteString(split)
		log.WriteString(fmt.Sprintf("%v", k))
		if k == "timestamp" {
			needTimeStamp = false
		}
		log.WriteString("=")
		log.WriteString(fmt.Sprintf("%v", v))
	}

	log.WriteString(split)
	f.appendPLid(opera_stat_key, &log)
	if needTimeStamp {
		log.WriteString(split)
		log.WriteString("timestamp=")
		log.WriteString(time.Now().Format(timestampFormat))
	}
	log.WriteString("\n")
	f.writePublic(log)
}

func TagLogPublic(opera_stat_key string, kv ...interface{}) {
	defaultOfflineFileLog.TagLogPublic(opera_stat_key, kv)
}

func LogPublic(opera_stat_key string, kv map[string]interface{}) {
	defaultOfflineFileLog.LogPublic(opera_stat_key, kv)
}

func getIp() string {
	var ip string
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		ip = "127.0.0.1"
		return ip
	}
	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				ip = ipnet.IP.String()
				break
			}
		}
	}
	return ip
}
