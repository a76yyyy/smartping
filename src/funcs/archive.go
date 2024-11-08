package funcs

import (
	"strconv"

	"github.com/a76yyyy/smartping/src/g"
	"github.com/cihub/seelog"
)

// clear timeout alert table
func ClearArchive() {
	seelog.Info("[func:ClearArchive] ", "starting run ClearArchive ")
	g.DLock.Lock()
	defer g.DLock.Unlock()
	g.Db.Exec("delete from alertlog where logtime < date('now','start of day','-" + strconv.Itoa(g.Cfg.Base["Archive"]) + " day')")
	g.Db.Exec("delete from mappinglog where logtime < date('now','start of day','-" + strconv.Itoa(g.Cfg.Base["Archive"]) + " day')")
	g.Db.Exec("delete from pinglog where logtime < date('now','start of day','-" + strconv.Itoa(g.Cfg.Base["Archive"]) + " day')")
	seelog.Info("[func:ClearArchive] ", "ClearArchive Finish ")
}
