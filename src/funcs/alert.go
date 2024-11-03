package funcs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"github.com/cihub/seelog"
	_ "modernc.org/sqlite"

	// _ "github.com/mattn/go-sqlite3"
	// _ "github.com/glebarez/sqlite"
	// _ "github.com/logoove/sqlite"
	// _ "github.com/go-sqlite/sqlite3"
	"github.com/a76yyyy/smartping/src/g"
	"github.com/a76yyyy/smartping/src/nettools"
	// "github.com/xhit/go-simple-mail"
)

type Cnt struct {
	Cnt int
}

func StartAlert() {
	seelog.Info("[func:StartAlert] ", "starting run AlertCheck ")
	for _, v := range g.SelfCfg.Topology {
		if v["Addr"] != g.SelfCfg.Addr {
			sFlag := CheckAlertStatus(v)
			if sFlag {
				g.AlertStatus[v["Addr"]] = true
			}
			_, haskey := g.AlertStatus[v["Addr"]]
			if (!haskey && !sFlag) || (!sFlag && g.AlertStatus[v["Addr"]]) {
				seelog.Debug("[func:StartAlert] ", v["Addr"]+" Alert!")
				g.AlertStatus[v["Addr"]] = false
				l := g.AlertLog{}
				l.Fromname = g.SelfCfg.Name
				l.Fromip = g.SelfCfg.Addr
				l.Logtime = time.Unix(time.Now().Unix(), 0).Format("2006-01-02 15:04")
				l.Targetname = v["Name"]
				l.Targetip = v["Addr"]
				mtrString := ""
				hops, err := nettools.RunMtr(v["Addr"], time.Second, 64, 6)
				if nil != err {
					seelog.Error("[func:StartAlert] Traceroute error ", err)
					mtrString = err.Error()
				} else {
					jHops, err := json.Marshal(hops)
					if err != nil {
						mtrString = err.Error()
					} else {
						mtrString = string(jHops)
					}
				}
				l.Tracert = mtrString
				go AlertStorage(l)
				if g.Cfg.Alert["SendEmailAccount"] != "" && g.Cfg.Alert["SendEmailPassword"] != "" && g.Cfg.Alert["EmailHost"] != "" && g.Cfg.Alert["RevcEmailList"] != "" {
					go AlertSendMail(l)
				}
				if g.Cfg.Alert["AgentId"] != "" && g.Cfg.Alert["CorpId"] != "" && g.Cfg.Alert["CorpSecret"] != "" && g.Cfg.Alert["RevcWechatList"] != "" {
					seelog.Info("[func:StartAlert] ", "AlertCheck SendWechat ")
					go AlertWechat(l)
				}
			}
		}
	}
	seelog.Info("[func:StartAlert] ", "AlertCheck finish ")
}

func CheckAlertStatus(v map[string]string) bool {
	Thdchecksec, _ := strconv.Atoi(v["Thdchecksec"])
	timeStartStr := time.Unix((time.Now().Unix() - int64(Thdchecksec)), 0).Format("2006-01-02 15:04")
	g.DLock.RLock()
	defer g.DLock.RUnlock()
	querysql := "SELECT count(1) cnt FROM  `pinglog` where logtime >= '" + timeStartStr + "' and target = '" + v["Addr"] + "' and (cast(avgdelay as double) > " + v["Thdavgdelay"] + " or cast(losspk as double) > " + v["Thdloss"] + ") "
	seelog.Debug("[func:CheckAlertStatus] ", querysql)
	rows, err := g.Db.Query(querysql)
	if err != nil {
		seelog.Error("[func:CheckAlertStatus] Query Error: ", err)
		return false
	}
	defer rows.Close()
	for rows.Next() {
		l := new(Cnt)
		err := rows.Scan(&l.Cnt)
		if err != nil {
			seelog.Error("[func:CheckAlertStatus] Scan Error: ", err)
			return false
		}
		Thdoccnum, _ := strconv.Atoi(v["Thdoccnum"])
		if l.Cnt <= Thdoccnum {
			return true
		} else {
			return false
		}
	}
	return false
}

func AlertStorage(t g.AlertLog) {
	seelog.Info("[func:AlertStorage] ", "(", t.Logtime, ")Starting AlertStorage ", t.Targetname)
	sql := "INSERT INTO [alertlog] (logtime, targetip, targetname, tracert) values('" + t.Logtime + "','" + t.Targetip + "','" + t.Targetname + "','" + t.Tracert + "')"
	g.DLock.Lock()
	defer g.DLock.Unlock()
	_, err := g.Db.Exec(sql)
	if err != nil {
		seelog.Error("[func:AlertStorage] Sql Error ", err)
	}
	seelog.Info("[func:AlertStorage] ", "(", t.Logtime, ") AlertStorage on ", t.Targetname, " finish!")
}

func AlertSendMail(t g.AlertLog) {
	hops := []nettools.Mtr{}
	err := json.Unmarshal([]byte(t.Tracert), &hops)
	if err != nil {
		seelog.Error("[func:AlertSendMail] json Error ", err)
		return
	}
	mtrstr := bytes.NewBufferString("")
	fmt.Fprintf(mtrstr, "<table>")
	fmt.Fprintf(mtrstr, "<tr><td>Host</td><td>Loss</td><td>Snt</td><td>Last</td><td>Avg</td><td>Best</td><td>Wrst</td><td>StDev</td></tr>")
	for i, hop := range hops {
		fmt.Fprintf(mtrstr, "<tr><td>%d %s</td><td>%.2f</td><td>%d</td><td>%v</td><td>%v</td><td>%v</td><td>%v</td><td>%v</td></tr>", i+1, hop.Host, ((float64(hop.Loss) / float64(hop.Send)) * 100), hop.Send, hop.Last, hop.Avg, hop.Best, hop.Wrst, hop.StDev)
	}
	fmt.Fprintf(mtrstr, "</table>")
	title := "【" + t.Fromname + "->" + t.Targetname + "】网络异常报警（" + t.Logtime + "）- SmartPing"
	content := "报警时间：" + t.Logtime + " <br> 来路：" + t.Fromname + "(" + t.Fromip + ") <br>  目的：" + t.Targetname + "(" + t.Targetip + ") <br> "
	SendEmailAccount := g.Cfg.Alert["SendEmailAccount"]
	SendEmailPassword := g.Cfg.Alert["SendEmailPassword"]
	EmailHost := g.Cfg.Alert["EmailHost"]
	RevcEmailList := g.Cfg.Alert["RevcEmailList"]
	err = SendMail(SendEmailAccount, SendEmailPassword, EmailHost, RevcEmailList, title, content+mtrstr.String())
	if err != nil {
		seelog.Error("[func:AlertSendMail] SendMail Error ", err)
	}
}

func SendMail(user, pwd, host, to, subject, body string) error {
	if len(strings.Split(host, ":")) == 1 {
		host = host + ":25"
	}
	auth := smtp.PlainAuth("", user, pwd, strings.Split(host, ":")[0])
	content_type := "Content-Type: text/html" + "; charset=UTF-8"
	msg := []byte("To: " + to + "\r\nFrom: " + user + "\r\nSubject: " + subject + "\r\n" + content_type + "\r\n\r\n" + body)
	send_to := strings.Split(to, ";")
	err := smtp.SendMail(host, auth, user, send_to, msg)
	if err != nil {
		seelog.Error("[func:SendMail] SMTP Send Mail Error ", err)
		return err
	}
	return nil
}

func AlertWechat(t g.AlertLog) {
	hops := []nettools.Mtr{}
	err := json.Unmarshal([]byte(t.Tracert), &hops)
	if err != nil {
		seelog.Error("[func:AlertWechat] json Error ", err)
		return
	}
	if hops[len(hops)-1].Loss == 0 {
		seelog.Info("[func:AlertWechat] loss 0")
		return
	}
	mtrstr := bytes.NewBufferString("")
	fmt.Fprintf(mtrstr, "Host     Loss\n")
	for i, hop := range hops {
		fmt.Fprintf(mtrstr, "%d %s    %.2f\n", i+1, hop.Host, ((float64(hop.Loss) / float64(hop.Send)) * 100))
	}
	title := "【" + t.Fromname + "->" + t.Targetname + "】网络异常报警（" + t.Logtime + "）- SmartPing\n"
	content := "报警时间：" + t.Logtime + " \n" // 来路：" + t.Fromname + "(" + t.Fromip + ") \n目的：" + t.Targetname + "(" + t.Targetip + ") \n"
	agentId, err := StringToInt(g.Cfg.Alert["AgentId"])
	if err != nil {
		seelog.Error("[func:AlertWechat] AgentId Error ", err)
		return
	}
	corpId := g.Cfg.Alert["CorpId"]
	corpSecret := g.Cfg.Alert["CorpSecret"]
	toUser := g.Cfg.Alert["RevcWechatList"]
	toParty := ""
	token := TOKEN{}
	// token.ErrCode, _ = funcs.StringToInt64(g.Cfg.Alert["ErrCode"])
	// token.ErrMsg = g.Cfg.Alert["ErrMsg"]
	token.AccessToken = g.Cfg.Alert["AccessToken"]
	token.ExpiresIn, _ = StringToInt64(g.Cfg.Alert["ExpiresIn"])
	token.Time, _ = time.Parse("2006-01-02 15:04:05", g.Cfg.Alert["Time"])
	if token.AccessToken == "" || time.Since(token.Time).Seconds() > float64(token.ExpiresIn)-200 {
		token = GetAccessToken(corpId, corpSecret)
		if token.ErrCode != 0 {
			seelog.Error("[func:AlertWechat] GetAccessToken Error ", token.ErrCode, token.ErrMsg)
			return
		}
		// g.Cfg.Alert["ErrCode"] = funcs.Int64ToString(token.ErrCode)
		// g.Cfg.Alert["ErrMsg"] = token.ErrMsg
		g.Cfg.Alert["AccessToken"] = token.AccessToken
		g.Cfg.Alert["ExpiresIn"] = Int64ToString(token.ExpiresIn)
		g.Cfg.Alert["Time"] = token.Time.Format("2006-01-02 15:04:05")
		saveerr := g.SaveConfig()
		if saveerr != nil {
			seelog.Error("[func:AlertWechat] SaveAccessToken Error ", saveerr.Error())
			return
		}
	}
	msg := strings.Replace(Messages(toUser, toParty, agentId, content+mtrstr.String(), title, ""), "\\\\", "\\", -1)
	err = SendMessage(token.AccessToken, msg)
	if err != nil {
		seelog.Error("[func:AlertWechat] SendWechat Error ", err)
	}
}
