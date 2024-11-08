package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/a76yyyy/smartping/src/funcs"
	"github.com/a76yyyy/smartping/src/g"
	"github.com/a76yyyy/smartping/src/nettools"
	"github.com/cihub/seelog"
	"github.com/wcharczuk/go-chart"
	"github.com/wcharczuk/go-chart/drawing"
	"github.com/wzv5/pping/pkg/ping"
)

func configApiRoutes() {

	//配置文件API
	http.HandleFunc("/api/config.json", configHandle)

	//Ping数据API
	http.HandleFunc("/api/ping.json", pingHandle)

	//Ping拓扑API
	http.HandleFunc("/api/topology.json", topologyHandle)

	//报警API
	http.HandleFunc("/api/alert.json", alertHandle)

	//全国延迟API
	http.HandleFunc("/api/mapping.json", mappingHandle)

	//检测工具API
	http.HandleFunc("/api/tools.json", toolsHandle)

	//保存配置文件
	http.HandleFunc("/api/saveconfig.json", saveconfigHandle)

	//发送测试邮件
	http.HandleFunc("/api/sendmailtest.json", sendmailtestHandle)

	//发送测试消息
	http.HandleFunc("/api/sendwechattest.json", sendwechattestHandle)

	// 接收企微消息
	http.HandleFunc("/api/wechatmsg.json", wechatmsghandle)

	//Ping画图
	http.HandleFunc("/api/graph.png", graphHandle)

	//代理访问
	http.HandleFunc("/api/proxy.json", proxyHandle)

}

func copy_array(src []string) []string {
	dst := make([]string, len(src))
	copy(dst, src)
	return dst
}

func configHandle(w http.ResponseWriter, r *http.Request) {
	if !AuthUserIp(r.RemoteAddr) && !AuthAgentIp(r.RemoteAddr, true) {
		o := "Your ip address (" + r.RemoteAddr + ")  is not allowed to access this site!"
		http.Error(w, o, http.StatusUnauthorized)
		return
	}
	r.ParseForm()
	nconf := g.Config{}
	cfgJson, _ := json.Marshal(g.Cfg)
	json.Unmarshal(cfgJson, &nconf)
	nconf.Password = ""
	if !AuthAgentIp(r.RemoteAddr, false) {
		if nconf.Alert["SendEmailPassword"] != "" {
			nconf.Alert["SendEmailPassword"] = "samepasswordasbefore"
		}
		if nconf.Alert["CorpId"] != "" {
			nconf.Alert["CorpId"] = "samepasswordasbefore"
		}
		if nconf.Alert["CorpSecret"] != "" {
			nconf.Alert["CorpSecret"] = "samepasswordasbefore"
		}
		if nconf.Alert["CorpToken"] != "" {
			nconf.Alert["CorpToken"] = "samepasswordasbefore"
		}
		if nconf.Alert["CorpEncodingAESKey"] != "" {
			nconf.Alert["CorpEncodingAESKey"] = "samepasswordasbefore"
		}
	}
	//fmt.Print(g.Cfg.Alert["SendEmailPassword"])
	onconf, _ := json.Marshal(nconf)
	var out bytes.Buffer
	json.Indent(&out, onconf, "", "\t")
	o := out.String()
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintln(w, o)
}

func pingHandle(w http.ResponseWriter, r *http.Request) {
	if !AuthUserIp(r.RemoteAddr) && !AuthAgentIp(r.RemoteAddr, true) {
		o := "Your ip address (" + r.RemoteAddr + ")  is not allowed to access this site!"
		http.Error(w, o, http.StatusUnauthorized)
		return
	}
	r.ParseForm()
	// if len(r.Form["ip"]) == 0 {
	// 	o := "Missing Param !"
	// 	http.Error(w, o, http.StatusNotAcceptable)
	// 	return
	// }
	var tableip string
	var timeStart int64
	var timeEnd int64
	var timeStartStr string
	var timeEndStr string
	if len(r.Form["starttime"]) > 0 && len(r.Form["endtime"]) > 0 {
		timeStartStr = r.Form["starttime"][0]
		if timeStartStr != "" {
			tms, _ := time.Parse("2006-01-02 15:04", timeStartStr)
			timeStart = tms.Unix() - 8*60*60
		} else {
			timeStart = time.Now().Unix() - 2*60*60
			timeStartStr = time.Unix(timeStart, 0).Format("2006-01-02 15:04")
		}
		timeEndStr = r.Form["endtime"][0]
		if timeEndStr != "" {
			tmn, _ := time.Parse("2006-01-02 15:04", timeEndStr)
			timeEnd = tmn.Unix() - 8*60*60
		} else {
			timeEnd = time.Now().Unix()
			timeEndStr = time.Unix(timeEnd, 0).Format("2006-01-02 15:04")
		}
	} else {
		timeStart = time.Now().Unix() - 2*60*60
		timeStartStr = time.Unix(timeStart, 0).Format("2006-01-02 15:04")
		timeEnd = time.Now().Unix()
		timeEndStr = time.Unix(timeEnd, 0).Format("2006-01-02 15:04")
	}
	cnt := int((timeEnd - timeStart) / 60)
	var lastcheck []string
	var maxdelay []string
	var mindelay []string
	var avgdelay []string
	var losspk []string
	timwwnum := map[string]int{}
	preouts := make(map[string]map[string][]string, 0)
	for i := 0; i < cnt+1; i++ {
		ntime := time.Unix(timeStart, 0).Format("2006-01-02 15:04")
		timwwnum[ntime] = i
		lastcheck = append(lastcheck, ntime)
		maxdelay = append(maxdelay, "0")
		mindelay = append(mindelay, "0")
		avgdelay = append(avgdelay, "0")
		losspk = append(losspk, "0")
		timeStart = timeStart + 60
	}
	var querySql string
	if len(r.Form["ip"]) == 0 {
		querySql = "SELECT target,logtime,maxdelay,mindelay,avgdelay,losspk FROM `pinglog` where logtime between '" + timeStartStr + "' and '" + timeEndStr + "' order by logtime "
	} else {
		tableip = r.Form["ip"][0]
		querySql = "SELECT target,logtime,maxdelay,mindelay,avgdelay,losspk FROM `pinglog` where target='" + tableip + "' and logtime between '" + timeStartStr + "' and '" + timeEndStr + "'  order by logtime "
	}
	g.DLock.RLock()
	defer g.DLock.RUnlock()
	rows, err := g.Db.Query(querySql)
	seelog.Debug("[func:/api/ping.json] Query ", querySql)
	if err != nil {
		seelog.Error("[func:/api/ping.json] Query ", err)
	} else {
		for rows.Next() {
			l := new(g.PingLog)
			var target string
			err := rows.Scan(&target, &l.Logtime, &l.Maxdelay, &l.Mindelay, &l.Avgdelay, &l.Losspk)
			if err != nil {
				seelog.Error("[/api/ping.json] Rows", err)
				continue
			}
			if _, ok := preouts[target]; !ok {
				preouts[target] = map[string][]string{
					"lastcheck": copy_array(lastcheck),
					"maxdelay":  copy_array(maxdelay),
					"mindelay":  copy_array(mindelay),
					"avgdelay":  copy_array(avgdelay),
					"losspk":    copy_array(losspk),
				}
			}
			for n, v := range lastcheck {
				if v == l.Logtime {
					preouts[target]["maxdelay"][n] = l.Maxdelay
					preouts[target]["mindelay"][n] = l.Mindelay
					preouts[target]["avgdelay"][n] = l.Avgdelay
					preouts[target]["losspk"][n] = l.Losspk
					break
				}
			}
			// for n, v := range lastcheck {
			// if v == l.Logtime {
			// maxdelay[n] = l.Maxdelay
			// mindelay[n] = l.Mindelay
			// avgdelay[n] = l.Avgdelay
			// losspk[n] = l.Losspk
			// 	break
			// }
			// }
		}
		rows.Close()
	}
	// preout := map[string][]string{
	// 	"lastcheck": lastcheck,
	// 	"maxdelay":  maxdelay,
	// 	"mindelay":  mindelay,
	// 	"avgdelay":  avgdelay,
	// 	"losspk":    losspk,
	// }
	w.Header().Set("Content-Type", "application/json")
	if len(r.Form["ip"]) == 0 {
		RenderJson(w, preouts)
	} else {
		RenderJson(w, preouts[tableip])
	}
}

func topologyHandle(w http.ResponseWriter, r *http.Request) {
	if !AuthUserIp(r.RemoteAddr) && !AuthAgentIp(r.RemoteAddr, true) {
		o := "Your ip address (" + r.RemoteAddr + ")  is not allowed to access this site!"
		http.Error(w, o, http.StatusUnauthorized)
		return
	}
	preout := make(map[string]string)
	for _, v := range g.SelfCfg.Topology {
		if funcs.CheckAlertStatus(v) {
			preout[v["Addr"]] = "true"
		} else {
			preout[v["Addr"]] = "false"
		}
	}
	w.Header().Set("Content-Type", "application/json")
	RenderJson(w, preout)
}

func alertHandle(w http.ResponseWriter, r *http.Request) {
	if !AuthUserIp(r.RemoteAddr) && !AuthAgentIp(r.RemoteAddr, true) {
		o := "Your ip address (" + r.RemoteAddr + ")  is not allowed to access this site!"
		http.Error(w, o, http.StatusUnauthorized)
		return
	}
	type DateList struct {
		Ldate string
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	r.ParseForm()
	dtb := time.Unix(time.Now().Unix(), 0).Format("2006-01-02")
	if len(r.Form["date"]) > 0 {
		dtb = strings.Replace(r.Form["date"][0], "alertlog-", "", -1)
	}
	listpreout := []string{}
	datapreout := []g.AlertLog{}
	g.DLock.RLock()
	defer g.DLock.RUnlock()
	querySql := "select date(logtime) as ldate from alertlog group by date(logtime) order by logtime desc"
	rows, err := g.Db.Query(querySql)
	seelog.Debug("[func:/api/alert.json] Query ", querySql)
	if err != nil {
		seelog.Error("[func:/api/alert.json] Query ", err)
	} else {
		for rows.Next() {
			l := new(DateList)
			err := rows.Scan(&l.Ldate)
			if err != nil {
				seelog.Error("[/api/alert.json] Rows", err)
				continue
			}
			listpreout = append(listpreout, l.Ldate)
		}
		rows.Close()
	}
	querySql = "select logtime,targetname,targetip,tracert from alertlog where logtime between '" + dtb + " 00:00:00' and '" + dtb + " 23:59:59'"
	rows, err = g.Db.Query(querySql)
	seelog.Debug("[func:/api/alert.json] Query ", querySql)
	if err != nil {
		seelog.Error("[func:/api/alert.json] Query ", err)
	} else {
		for rows.Next() {
			l := new(g.AlertLog)
			err := rows.Scan(&l.Logtime, &l.Targetname, &l.Targetip, &l.Tracert)
			l.Fromname = g.Cfg.Name
			l.Fromip = g.Cfg.Addr
			if err != nil {
				seelog.Error("[/api/alert.json] Rows", err)
				continue
			}
			datapreout = append(datapreout, *l)
		}
		rows.Close()
	}
	lout, _ := json.Marshal(listpreout)
	dout, _ := json.Marshal(datapreout)
	fmt.Fprintln(w, "["+string(lout)+","+string(dout)+"]")
}

func mappingHandle(w http.ResponseWriter, r *http.Request) {
	if !AuthUserIp(r.RemoteAddr) && !AuthAgentIp(r.RemoteAddr, true) {
		o := "Your ip address (" + r.RemoteAddr + ")  is not allowed to access this site!"
		http.Error(w, o, http.StatusUnauthorized)
		return
	}
	m, _ := time.ParseDuration("-1m")
	dataKey := time.Now().Add(m).Format("2006-01-02 15:04")
	r.ParseForm()
	if len(r.Form["d"]) > 0 {
		dataKey = r.Form["d"][0]
	}
	type Mapjson struct {
		Mapjson string
	}
	chinaMp := g.ChinaMp{}
	chinaMp.Text = g.Cfg.Name
	chinaMp.Subtext = dataKey
	chinaMp.Avgdelay = map[string][]g.MapVal{}
	chinaMp.Avgdelay["ctcc"] = []g.MapVal{}
	chinaMp.Avgdelay["cucc"] = []g.MapVal{}
	chinaMp.Avgdelay["cmcc"] = []g.MapVal{}
	g.DLock.RLock()
	defer g.DLock.RUnlock()
	querySql := "select mapjson from mappinglog where logtime = '" + dataKey + "'"
	rows, err := g.Db.Query(querySql)
	seelog.Debug("[func:/api/mapping.json] Query ", querySql)
	if err != nil {
		seelog.Error("[func:/api/mapping.json] Query ", err)
	} else {
		for rows.Next() {
			l := new(Mapjson)
			err := rows.Scan(&l.Mapjson)
			if err != nil {
				seelog.Error("[/api/mapping.json] Rows", err)
				continue
			}
			json.Unmarshal([]byte(l.Mapjson), &chinaMp.Avgdelay)
		}
		rows.Close()
	}
	w.Header().Set("Content-Type", "application/json")
	RenderJson(w, chinaMp)
}

func toolsHandle(w http.ResponseWriter, r *http.Request) {
	if !AuthUserIp(r.RemoteAddr) && !AuthAgentIp(r.RemoteAddr, true) {
		o := "Your ip address (" + r.RemoteAddr + ")  is not allowed to access this site!"
		http.Error(w, o, http.StatusUnauthorized)
		return
	}
	preout := g.ToolsRes{}
	preout.Status = "false"
	r.ParseForm()
	if len(r.Form["t"]) == 0 {
		preout.Error = "target empty!"
		RenderJson(w, preout)
		return
	}
	nowtime := int(time.Now().Unix())
	if _, ok := g.ToolLimit[r.RemoteAddr]; ok {
		if (nowtime - g.ToolLimit[r.RemoteAddr]) <= g.Cfg.Toollimit {
			preout.Error = "Time Limit Exceeded!"
			RenderJson(w, preout)
			return
		}
	}
	g.ToolLimit[r.RemoteAddr] = nowtime
	reg := regexp.MustCompile("http[s]{0,1}://")
	// target := strings.Replace(strings.Replace(r.Form["t"][0], "https://", "", -1), "http://", "", -1)
	targets := strings.Split(reg.ReplaceAllString(r.Form["t"][0], ""), " ")
	target := targets[0]
	mode := r.Form["m"][0]
	preout.Ping = g.PingSt{}
	preout.Ping.MinDelay = -1
	ipaddr, err := net.ResolveIPAddr("ip", target)
	if err != nil {
		preout.Error = "Unable to resolve destination host"
		RenderJson(w, preout)
		return
	}
	preout.Ip = ipaddr.String()
	if mode == "icmp" {
		preout.Ping, err = nettools.GeneralPing(ipaddr, 5, 3*time.Second, 64)
		if err != nil {
			preout.Error = err.Error()
			RenderJson(w, preout)
			return
		}
		if preout.Ping.RevcPk == 0 {
			preout.Ping.AvgDelay = 3000
			preout.Ping.MinDelay = 3000
			preout.Ping.MaxDelay = 3000
		}
	} else {
		timeout := time.Second * 4
		s := funcs.Statistics{}
		switch mode {
		case "http":
			method := "GET"
			url := "http://" + preout.Ip
			p := ping.NewHttpPing(method, url, timeout)
			p.DisableHttp2 = false
			p.DisableCompression = false
			p.Insecure = false
			p.Referrer = ""
			p.UserAgent = ""
			p.IP = ipaddr.IP
			s, err = funcs.RunPing(p)
		case "tcp":
			port := 80
			if len(targets) > 1 {
				port, _ = funcs.StringToInt(targets[1])
			}
			p := ping.NewTcpPing(preout.Ip, uint16(port), timeout)
			s, err = funcs.RunPing(p)
		case "tls":
			port := 443
			if len(targets) > 1 {
				port, _ = funcs.StringToInt(targets[1])
			}
			handTime := time.Second * 10
			p := ping.NewTlsPing(preout.Ip, uint16(port), timeout, handTime)
			p.TlsVersion = 0
			p.Insecure = false
			p.IP = ipaddr.IP
			s, err = funcs.RunPing(p)
		case "dns":
			port := 53
			if len(targets) > 1 {
				port, _ = funcs.StringToInt(targets[1])
			}
			qType := "NS"
			if len(targets) > 2 {
				qType = targets[2]
			}
			p := ping.NewDnsPing(preout.Ip, timeout)
			p.Port = uint16(port)
			if port == 53 {
				p.Net = "udp"
			} else {
				p.Net = "tcp-tls"
			}
			p.Type = qType
			p.Domain = target
			p.Insecure = false
			s, err = funcs.RunPing(p)
		case "icmp":
			fallthrough
		default:
			p := ping.NewIcmpPing(preout.Ip, timeout)
			s, err = funcs.RunPing(p)
		}
		if err != nil {
			preout.Error = "Unable to resolve destination host " + target
			RenderJson(w, preout)
			return
		}
		preout.Ping.AvgDelay = s.Total / float64(s.Ok)
		preout.Ping.MinDelay = s.Min
		preout.Ping.MaxDelay = s.Max
		preout.Ping.SendPk = s.Sent
		preout.Ping.RevcPk = s.Ok
		preout.Ping.LossPk = s.Failed
	}
	preout.Status = "true"
	w.Header().Set("Content-Type", "application/json")
	RenderJson(w, preout)
}

func saveconfigHandle(w http.ResponseWriter, r *http.Request) {
	if !AuthUserIp(r.RemoteAddr) && !AuthAgentIp(r.RemoteAddr, true) {
		o := "Your ip address (" + r.RemoteAddr + ")  is not allowed to access this site!"
		http.Error(w, o, http.StatusUnauthorized)
		return
	}
	preout := make(map[string]string)
	r.ParseForm()
	preout["status"] = "false"
	if len(r.Form["password"]) == 0 || r.Form["password"][0] != g.Cfg.Password {
		preout["info"] = "密码错误!"
		RenderJson(w, preout)
		return
	}
	if len(r.Form["config"]) == 0 {
		preout["info"] = "参数错误!"
		RenderJson(w, preout)
		return
	}
	nconfig := g.Config{}
	err := json.Unmarshal([]byte(r.Form["config"][0]), &nconfig)
	if err != nil {
		preout["info"] = "配置文件解析错误!" + err.Error()
		RenderJson(w, preout)
		return
	}
	if nconfig.Name == "" {
		preout["info"] = "本机节点名称为空!"
		RenderJson(w, preout)
		return
	}
	if !ValidIP4(nconfig.Addr) {
		preout["info"] = "非法本机节点IP!"
		RenderJson(w, preout)
		return
	}
	//Base
	if _, ok := nconfig.Base["Timeout"]; !ok || nconfig.Base["Timeout"] <= 0 {
		preout["info"] = "非法超时时间!(>0)"
		RenderJson(w, preout)
		return
	}
	if _, ok := nconfig.Base["Archive"]; !ok || nconfig.Base["Archive"] <= 0 {
		preout["info"] = "非法存档天数!(>0)"
		RenderJson(w, preout)
		return
	}
	if _, ok := nconfig.Base["Refresh"]; !ok || nconfig.Base["Refresh"] <= 0 {
		preout["info"] = "非法刷新频率!(>0)"
		RenderJson(w, preout)
		return
	}
	//Topology
	if _, ok := nconfig.Topology["Tline"]; !ok || nconfig.Topology["Tline"] <= "0" {
		preout["info"] = "非法拓扑连线粗细(>0)"
		RenderJson(w, preout)
		return
	}
	if _, ok := nconfig.Topology["Tsymbolsize"]; !ok || nconfig.Topology["Tsymbolsize"] <= "0" {
		preout["info"] = "非法拓扑形状大小!(>0)"
		RenderJson(w, preout)
		return
	}
	if nconfig.Toollimit < 0 {
		preout["info"] = "非法检测工具限定频率!(>=0)"
		RenderJson(w, preout)
		return
	}
	//Network
	for k, network := range nconfig.Network {
		if !ValidIP4(network.Addr) || !ValidIP4(k) {
			preout["info"] = "Ping节点测试网络信息错误!(非法节点IP地址 " + k + ")"
			RenderJson(w, preout)
			return
		}
		if network.Name == "" {
			preout["info"] = "Ping节点测试网络信息错误!( " + k + " 节点名称为空)"
			RenderJson(w, preout)
			return
		}
		for _, topology := range network.Topology {
			if _, ok := topology["Thdchecksec"]; !ok {
				preout["info"] = "Ping节点测试网络信息错误!( " + k + "->" + topology["Addr"] + " 非法拓扑报警规则，秒) "
				RenderJson(w, preout)
				return
			} else {
				Thdchecksec, err := strconv.Atoi(topology["Thdchecksec"])
				if err != nil || Thdchecksec <= 0 {
					preout["info"] = "Ping节点测试网络信息错误!( " + k + "->" + topology["Addr"] + " 非法拓扑报警规则，>0 秒  ) "
					RenderJson(w, preout)
					return
				}
			}
			if _, ok := topology["Thdloss"]; !ok {
				preout["info"] = "Ping节点测试网络信息错误!( " + k + "->" + topology["Addr"] + " 非法拓扑报警规则，%) "
				RenderJson(w, preout)
				return
			} else {
				Thdloss, err := strconv.Atoi(topology["Thdloss"])
				if err != nil || (Thdloss < 0 || Thdloss > 100) {
					preout["info"] = "Ping节点测试网络信息错误!( " + k + "->" + topology["Addr"] + " 非法拓扑报警规则，0 <= % <=100  ) "
					RenderJson(w, preout)
					return
				}
			}
			if _, ok := topology["Thdavgdelay"]; !ok {
				preout["info"] = "Ping节点测试网络信息错误!( " + k + "->" + topology["Addr"] + " 非法拓扑报警规则，ms) "
				RenderJson(w, preout)
				return
			} else {
				Thdavgdelay, err := strconv.Atoi(topology["Thdavgdelay"])
				if err != nil || Thdavgdelay <= 0 {
					preout["info"] = "Ping节点测试网络信息错误!( " + k + "->" + topology["Addr"] + " 非法拓扑报警规则，> 0 ms  ) "
					RenderJson(w, preout)
					return
				}
			}
			if _, ok := topology["Thdoccnum"]; !ok {
				preout["info"] = "Ping节点测试网络信息错误!( " + k + "->" + topology["Addr"] + " 非法拓扑报警规则，次) "
				RenderJson(w, preout)
				return
			} else {
				Thdoccnum, err := strconv.Atoi(topology["Thdoccnum"])
				if err != nil || Thdoccnum <= 0 {
					preout["info"] = "Ping节点测试网络信息错误!( " + k + "->" + topology["Addr"] + " 非法拓扑报警规则，> 0 次  ) "
					RenderJson(w, preout)
					return
				}
			}
		}
	}
	//ChinaMap
	for _, provVal := range nconfig.Chinamap {
		for _, telcomVal := range provVal {
			for _, ip := range telcomVal {
				if ip != "" && !ValidIP4(ip) {
					preout["info"] = "Mapping Ip illegal!"
					RenderJson(w, preout)
					return
				}
			}
		}
	}
	nconfig.Ver = g.Cfg.Ver
	nconfig.Port = g.Cfg.Port
	nconfig.Password = g.Cfg.Password
	if nconfig.Alert["SendEmailPassword"] == "samepasswordasbefore" {
		nconfig.Alert["SendEmailPassword"] = g.Cfg.Alert["SendEmailPassword"]
	}
	if nconfig.Alert["CorpId"] == "samepasswordasbefore" {
		nconfig.Alert["CorpId"] = g.Cfg.Alert["CorpId"]
	}
	if nconfig.Alert["CorpSecret"] == "samepasswordasbefore" {
		nconfig.Alert["CorpSecret"] = g.Cfg.Alert["CorpSecret"]
	}
	if nconfig.Alert["CorpToken"] == "samepasswordasbefore" {
		nconfig.Alert["CorpToken"] = g.Cfg.Alert["CorpToken"]
	}
	if nconfig.Alert["CorpEncodingAESKey"] == "samepasswordasbefore" {
		nconfig.Alert["CorpEncodingAESKey"] = g.Cfg.Alert["CorpEncodingAESKey"]
	}
	g.Cfg = nconfig
	g.SelfCfg = g.Cfg.Network[g.Cfg.Addr]
	saveerr := g.SaveConfig()
	if saveerr != nil {
		preout["info"] = saveerr.Error()
		RenderJson(w, preout)
		return
	}
	preout["status"] = "true"
	RenderJson(w, preout)
}

func sendmailtestHandle(w http.ResponseWriter, r *http.Request) {
	if !AuthUserIp(r.RemoteAddr) && !AuthAgentIp(r.RemoteAddr, true) {
		o := "Your ip address (" + r.RemoteAddr + ")  is not allowed to access this site!"
		http.Error(w, o, http.StatusUnauthorized)
		return
	}
	preout := make(map[string]string)
	r.ParseForm()
	preout["status"] = "false"
	if len(r.Form["EmailHost"]) == 0 {
		preout["info"] = "邮件服务器不能为空!"
		RenderJson(w, preout)
		return
	}
	if len(r.Form["SendEmailAccount"]) == 0 {
		preout["info"] = "发件邮件不能为空!"
		RenderJson(w, preout)
		return
	}
	if len(r.Form["SendEmailPassword"]) == 0 {
		preout["info"] = "发件邮箱密码不能为空!"
		RenderJson(w, preout)
		return
	}
	password := r.Form["SendEmailPassword"][0]
	if password == "samepasswordasbefore" {
		password = g.Cfg.Alert["SendEmailPassword"]
	}
	if len(r.Form["RevcEmailList"]) == 0 {
		preout["info"] = "收件邮箱列表不能为空!"
		RenderJson(w, preout)
		return
	}

	err := funcs.SendMail(r.Form["SendEmailAccount"][0], password, r.Form["EmailHost"][0], r.Form["RevcEmailList"][0], "报警测试邮件 - SmartPing", "报警测试邮件")
	if err != nil {
		preout["info"] = err.Error()
		RenderJson(w, preout)
		return
	}
	preout["status"] = "true"
	RenderJson(w, preout)
}

func sendwechattestHandle(w http.ResponseWriter, r *http.Request) {
	if !AuthUserIp(r.RemoteAddr) && !AuthAgentIp(r.RemoteAddr, true) {
		o := "Your ip address (" + r.RemoteAddr + ")  is not allowed to access this site!"
		http.Error(w, o, http.StatusUnauthorized)
		return
	}
	preout := make(map[string]string)
	r.ParseForm()
	preout["status"] = "false"
	if len(r.Form["CorpId"]) == 0 {
		preout["info"] = "企业编号不能为空!"
		RenderJson(w, preout)
		return
	}
	if len(r.Form["CorpSecret"]) == 0 {
		preout["info"] = "企业密钥不能为空!"
		RenderJson(w, preout)
		return
	}
	if len(r.Form["AgentId"]) == 0 {
		preout["info"] = "应用编号不能为空!"
		RenderJson(w, preout)
		return
	}
	if len(r.Form["RevcWechatList"]) == 0 {
		preout["info"] = "收件用户列表不能为空!"
		RenderJson(w, preout)
		return
	}

	agentId, err := funcs.StringToInt(r.Form["AgentId"][0])
	if err != nil {
		preout["info"] = "应用编号非法, 必须为整数!"
		RenderJson(w, preout)
		return
	}
	corpId := r.Form["CorpId"][0]
	corpSecret := r.Form["CorpSecret"][0]
	if corpId == "samepasswordasbefore" {
		corpId = g.Cfg.Alert["CorpId"]
	}
	if corpSecret == "samepasswordasbefore" {
		corpSecret = g.Cfg.Alert["CorpSecret"]
	}
	toUser := r.Form["RevcWechatList"][0]
	toParty := ""
	token := funcs.TOKEN{}
	// token.ErrCode, _ = funcs.StringToInt64(g.Cfg.Alert["ErrCode"])
	// token.ErrMsg = g.Cfg.Alert["ErrMsg"]
	token.AccessToken = g.Cfg.Alert["AccessToken"]
	token.ExpiresIn, _ = funcs.StringToInt64(g.Cfg.Alert["ExpiresIn"])
	token.Time, _ = time.Parse("2006-01-02 15:04:05", g.Cfg.Alert["Time"])
	if token.AccessToken == "" || time.Since(token.Time).Seconds() > float64(token.ExpiresIn)-200 {
		token = funcs.GetAccessToken(corpId, corpSecret)
		if token.ErrCode != 0 {
			preout["info"] = fmt.Sprint("[func:AlertWechat] GetAccessToken Error ", token.ErrCode, token.ErrMsg)
			RenderJson(w, preout)
			return
		}
		// g.Cfg.Alert["ErrCode"] = funcs.Int64ToString(token.ErrCode)
		// g.Cfg.Alert["ErrMsg"] = token.ErrMsg
		g.Cfg.Alert["AccessToken"] = token.AccessToken
		g.Cfg.Alert["ExpiresIn"] = funcs.Int64ToString(token.ExpiresIn)
		g.Cfg.Alert["Time"] = token.Time.Format("2006-01-02 15:04:05")
		saveerr := g.SaveConfig()
		if saveerr != nil {
			preout["info"] = saveerr.Error()
			RenderJson(w, preout)
			return
		}
	}
	msg := strings.Replace(funcs.Messages(toUser, toParty, agentId, "测试消息", "SmartPing", ""), "\\\\", "\\", -1)
	err = funcs.SendMessage(token.AccessToken, msg)
	if err != nil {
		preout["info"] = err.Error()
		RenderJson(w, preout)
		return
	}
	preout["status"] = "true"
	RenderJson(w, preout)
}

func wechatmsghandle(w http.ResponseWriter, r *http.Request) {
	preout := make(map[string]string)
	r.ParseForm()
	preout["status"] = "false"
	if len(r.Form["msg_signature"]) == 0 {
		preout["info"] = "企业微信加密签名不能为空!"
		RenderJson(w, preout)
		return
	}
	if len(r.Form["timestamp"]) == 0 {
		preout["info"] = "时间戳不能为空!"
		RenderJson(w, preout)
		return
	}
	if len(r.Form["nonce"]) == 0 {
		preout["info"] = "随机数不能为空!"
		RenderJson(w, preout)
		return
	}
	if r.Method == "GET" {
		if len(r.Form["echostr"]) == 0 {
			preout["info"] = "加密的字符串不能为空!"
			RenderJson(w, preout)
			return
		}
		// verifyURL(w, r.Form["msg_signature"][0], r.Form["timestamp"][0], r.Form["nonce"][0], r.Form["echostr"][0])
		return
	} else {
		if len(r.Form["ToUserName"]) == 0 {
			preout["info"] = "加密的字符串不能为空!"
			RenderJson(w, preout)
			return
		}
		if len(r.Form["AgentID"]) == 0 {
			preout["info"] = "加密的字符串不能为空!"
			RenderJson(w, preout)
			return
		}
		if len(r.Form["Encrypt"]) == 0 {
			preout["info"] = "加密的字符串不能为空!"
			RenderJson(w, preout)
			return
		}
		// decryptMsg(r.Form["msg_signature"[0]], r.Form["timestamp"][0], r.Form["nonce"][0])
		// encryptMsg(w, r)
	}
}

func graphHandle(w http.ResponseWriter, r *http.Request) {
	if !AuthUserIp(r.RemoteAddr) {
		o := "Your ip address (" + r.RemoteAddr + ")  is not allowed to access this site!"
		http.Error(w, o, http.StatusUnauthorized)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	r.ParseForm()
	if len(r.Form["g"]) == 0 {
		GraphText(83, 70, "GET PARAM ERROR").Save(w)
		return
	}
	url := strings.Replace(strings.Replace(r.Form["g"][0], "%26", "&", -1), " ", "%20", -1)
	config := g.PingStMini{}
	timeout := time.Duration(time.Duration(g.Cfg.Base["Timeout"]) * time.Second)
	client := http.Client{
		Timeout: timeout,
	}
	resp, err := client.Get(url)
	if err != nil {
		GraphText(80, 70, "REQUEST API ERROR").Save(w)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode == 401 {
		GraphText(80, 70, "401-UNAUTHORIZED").Save(w)
		return
	}
	if resp.StatusCode != 200 {
		GraphText(85, 70, "ERROR CODE "+strconv.Itoa(resp.StatusCode)).Save(w)
		return
	}
	body, err := ioutil.ReadAll(resp.Body)
	err = json.Unmarshal(body, &config)
	if err != nil {
		GraphText(80, 70, "PARSE DATA ERROR").Save(w)
		return
	}
	Xals := []float64{}
	AvgDelay := []float64{}
	LossPk := []float64{}
	Bkg := []float64{}
	MaxDelay := 0.0
	for i := 0; i < len(config.LossPk); i = i + 1 {
		avg, _ := strconv.ParseFloat(config.AvgDelay[i], 64)
		if MaxDelay < avg {
			MaxDelay = avg
		}
		AvgDelay = append(AvgDelay, avg)
		losspk, _ := strconv.ParseFloat(config.LossPk[i], 64)
		LossPk = append(LossPk, losspk)
		Xals = append(Xals, float64(i))
		Bkg = append(Bkg, 100.0)
	}
	graph := chart.Chart{
		Width:  300 * 3,
		Height: 130 * 3,
		Background: chart.Style{
			FillColor: drawing.Color{R: 249, G: 246, B: 241, A: 255},
		},
		XAxis: chart.XAxis{
			Style: chart.Style{
				Show:     true,
				FontSize: 20,
			},
			TickPosition: chart.TickPositionBetweenTicks,
			ValueFormatter: func(v interface{}) string {
				return config.Lastcheck[int(v.(float64))][11:16]
			},
		},
		YAxis: chart.YAxis{
			Style: chart.Style{
				Show:     true,
				FontSize: 20,
			},
			Range: &chart.ContinuousRange{
				Min: 0.0,
				Max: 100.0,
			},
			ValueFormatter: func(v interface{}) string {
				if vf, isFloat := v.(float64); isFloat {
					return fmt.Sprintf("%0.0f", vf)
				}
				return ""
			},
		},
		YAxisSecondary: chart.YAxis{
			//NameStyle: chart.StyleShow(),
			Style: chart.Style{
				Show:     true,
				FontSize: 20,
			},
			Range: &chart.ContinuousRange{
				Min: 0.0,
				Max: MaxDelay + MaxDelay/10,
			},
			ValueFormatter: func(v interface{}) string {
				if vf, isFloat := v.(float64); isFloat {
					return fmt.Sprintf("%0.0f", vf)
				}
				return ""
			},
		},
		Series: []chart.Series{
			chart.ContinuousSeries{
				Style: chart.Style{
					Show:        true,
					StrokeColor: drawing.Color{R: 249, G: 246, B: 241, A: 255},
					FillColor:   drawing.Color{R: 249, G: 246, B: 241, A: 255},
				},
				XValues: Xals,
				YValues: Bkg,
			},
			chart.ContinuousSeries{
				Style: chart.Style{
					Show:        true,
					StrokeColor: drawing.Color{R: 0, G: 204, B: 102, A: 200},
					FillColor:   drawing.Color{R: 0, G: 204, B: 102, A: 200},
				},
				XValues: Xals,
				YValues: AvgDelay,
				YAxis:   chart.YAxisSecondary,
			},
			chart.ContinuousSeries{
				Style: chart.Style{
					Show:        true,
					StrokeColor: drawing.Color{R: 255, G: 0, B: 0, A: 200},
					FillColor:   drawing.Color{R: 255, G: 0, B: 0, A: 200},
				},
				XValues: Xals,
				YValues: LossPk,
			},
		},
	}
	graph.Render(chart.PNG, w)
}

func proxyHandle(w http.ResponseWriter, r *http.Request) {
	if !AuthUserIp(r.RemoteAddr) {
		o := "Your ip address (" + r.RemoteAddr + ")  is not allowed to access this site!"
		http.Error(w, o, http.StatusUnauthorized)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	r.ParseForm()
	if len(r.Form["g"]) == 0 {
		o := "Url Param Error!"
		http.Error(w, o, http.StatusNotAcceptable)
		return
	}
	to := strconv.Itoa(g.Cfg.Base["Timeout"])
	if len(r.Form["t"]) > 0 {
		to = r.Form["t"][0]
	}
	url := strings.Replace(strings.Replace(r.Form["g"][0], "%26", "&", -1), " ", "%20", -1)
	defaultto, err := strconv.Atoi(to)
	if err != nil {
		o := "Timeout Param Error!"
		http.Error(w, o, http.StatusNotAcceptable)
		return
	}
	timeout := time.Duration(time.Duration(defaultto) * time.Second)
	client := http.Client{
		Timeout: timeout,
	}
	resp, err := client.Get(url)
	if err != nil {
		o := "Request Remote Data Error:" + err.Error()
		http.Error(w, o, http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()
	resCode := resp.StatusCode
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		o := "Read Remote Data Error:" + err.Error()
		http.Error(w, o, http.StatusServiceUnavailable)
		return
	}
	if resCode != 200 {
		o := "Get Remote Data Status Error"
		http.Error(w, o, resCode)
	}
	var out bytes.Buffer
	json.Indent(&out, body, "", "\t")
	o := out.String()
	fmt.Fprintln(w, o)
}
