package funcs

import (
	"encoding/json"
	"fmt"
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

var (
	MapLock   = new(sync.Mutex)
	MapStatus map[string][]g.MapVal
)

func Mapping() {
	var wg sync.WaitGroup
	MapStatus = map[string][]g.MapVal{}
	seelog.Debug("[func:Mapping]", g.Cfg.Chinamap)
	for tel, provDetail := range g.Cfg.Chinamap {
		for prov := range provDetail {
			seelog.Debug("[func:Mapping]", g.Cfg.Chinamap[tel][prov])
			if len(g.Cfg.Chinamap[tel][prov]) > 0 {
				go MappingTask(tel, prov, g.Cfg.Chinamap[tel][prov], &wg)
				wg.Add(1)
			}
		}
	}
	wg.Wait()
	MapPingStorage()
}

// ping main function
func MappingTask(tel string, prov string, ips []string, wg *sync.WaitGroup) {
	seelog.Info("Start MappingTask " + tel + " " + prov + "..")
	statMap := []g.PingSt{}
	for _, ip := range ips {
		seelog.Debug("[func:MappingTask] Target Addr: ", ip)
		ipaddr, err := net.ResolveIPAddr("ip", ip)
		if err == nil {
			stat, err := nettools.GeneralPing(ipaddr, 3, 3*time.Second, 64)
			if err != nil {
				seelog.Error("[func:MappingTask] Target Addr: ", ipaddr, " err: ", err)
			}

			if stat.RevcPk == 0 {
				stat.AvgDelay = 3000.00
				stat.MinDelay = 3000.00
				stat.MaxDelay = 3000.00
			}
			statMap = append(statMap, stat)
		} else {
			seelog.Error("[func:MappingTask] Target Addr: ", ip, " err: Unable to resolve destination host")
			stat := g.PingSt{}
			stat.AvgDelay = 3000.00
			stat.MinDelay = 3000.00
			stat.MaxDelay = 3000.00
			stat.SendPk = 0
			stat.RevcPk = 0
			stat.LossPk = 100
			statMap = append(statMap, stat)
		}
	}
	fStatDetail := g.PingSt{}
	fT := 0
	effCnt := 0
	for _, stat := range statMap {
		if len(statMap) > 1 && fT < (len(statMap)+3)/4 {
			if stat.LossPk == 100 {
				fT = fT + 1
				continue
			}
		}
		// fStatDetail.MaxDelay = fStatDetail.MaxDelay + stat.MaxDelay
		// fStatDetail.MinDelay = fStatDetail.MinDelay + stat.MinDelay
		fStatDetail.AvgDelay = fStatDetail.AvgDelay + stat.AvgDelay
		// fStatDetail.SendPk = fStatDetail.SendPk + stat.SendPk
		// fStatDetail.RevcPk = fStatDetail.RevcPk + stat.RevcPk
		// fStatDetail.LossPk = fStatDetail.SendPk - fStatDetail.RevcPk
		effCnt = effCnt + 1
	}
	gMapVal := g.MapVal{}
	gMapVal.Name = tel
	value, _ := strconv.ParseFloat(fmt.Sprintf("%.2f", fStatDetail.AvgDelay/float64(effCnt)), 64)
	gMapVal.Value = value
	MapLock.Lock()
	MapStatus[prov] = append(MapStatus[prov], gMapVal)
	MapLock.Unlock()
	wg.Done()
	seelog.Info("Finish MappingTask " + tel + " " + prov + "..")
}

func MapPingStorage() {
	seelog.Info("Start MapPingStorage...")
	logtime := time.Now().Format("2006-01-02 15:04")
	seelog.Debug("[func:MapPingStorage] ", "(", logtime, ")Starting MapPingStorage ", MapStatus)
	jdata, err := json.Marshal(MapStatus)
	if err != nil {
		seelog.Error("[func:MapPingStorage] Json Error ", err)
	}
	sql := "REPLACE INTO [mappinglog] (logtime, mapjson) values('" + logtime + "','" + string(jdata) + "')"
	seelog.Debug("[func:MapPingStorage] ", sql)
	g.DLock.Lock()
	g.Db.Exec(sql)
	_, err = g.Db.Exec(sql)
	seelog.Debug(sql)
	if err != nil {
		seelog.Error("[func:MapPingStorage] Sql Error ", err)
	}
	g.DLock.Unlock()
	seelog.Info("Finish MapPingStorage.")
}
