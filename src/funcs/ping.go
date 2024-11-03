package funcs

import (
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/cihub/seelog"
	_ "github.com/mattn/go-sqlite3"

	// _ "github.com/glebarez/sqlite"
	// _ "github.com/logoove/sqlite"
	// _ "github.com/go-sqlite/sqlite3"
	"github.com/a76yyyy/smartping/src/g"
	"github.com/a76yyyy/smartping/src/nettools"
)

func Ping() {
	var wg sync.WaitGroup
	for _, target := range g.SelfCfg.Ping {
		wg.Add(1)
		go PingTask(g.Cfg.Network[target], &wg)
	}
	wg.Wait()
	go StartAlert()
}

// ping main function
func PingTask(t g.NetworkMember, wg *sync.WaitGroup) {
	defer wg.Done()
	seelog.Info("Start Ping " + t.Addr + "..")
	stat := g.PingSt{
		MinDelay: -1,
	}
	ipaddr, err := net.ResolveIPAddr("ip", t.Addr)
	if err == nil {
		stat, err = nettools.UniformPing(ipaddr, 20, 3*time.Second, 64)
		if err != nil {
			seelog.Error("[func:PingTask] Target Addr: ", ipaddr, " err: ", err)
		}
	} else {
		stat.AvgDelay = 0.00
		stat.MinDelay = 0.00
		stat.MaxDelay = 0.00
		stat.SendPk = 0
		stat.RevcPk = 0
		stat.LossPk = 100
		seelog.Debug("[func:PingTask] Target Addr:", t.Addr, " err: Unable to resolve destination host")
	}
	PingStorage(stat, t.Addr)
	seelog.Info("Finish Ping " + t.Addr + "..")
}

// storage ping data
func PingStorage(pingres g.PingSt, Addr string) {
	logtime := time.Now().Format("2006-01-02 15:04")
	seelog.Info("[func:PingStorage] ", "(", logtime, ")Starting PingStorage: ", Addr)
	sql := "INSERT INTO [pinglog] (logtime, target, maxdelay, mindelay, avgdelay, sendpk, revcpk, losspk) values('" + logtime + "','" + Addr + "','" + strconv.FormatFloat(pingres.MaxDelay, 'f', 2, 64) + "','" + strconv.FormatFloat(pingres.MinDelay, 'f', 2, 64) + "','" + strconv.FormatFloat(pingres.AvgDelay, 'f', 2, 64) + "','" + strconv.Itoa(pingres.SendPk) + "','" + strconv.Itoa(pingres.RevcPk) + "','" + strconv.Itoa(pingres.LossPk) + "')"
	seelog.Debug("[func:PingStorage] ", sql)
	g.DLock.Lock()
	_, err := g.Db.Exec(sql)
	if err != nil {
		seelog.Error("[func:PingStorage] Sql Error ", err)
	}
	g.DLock.Unlock()
	seelog.Info("[func:PingStorage] ", "(", logtime, ") Finish PingStorage  ", Addr)
}
