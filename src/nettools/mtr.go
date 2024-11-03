package nettools

import (
	"net"
	"time"

	"github.com/a76yyyy/smartping/src/g"
	"github.com/cihub/seelog"
	probing "github.com/prometheus-community/pro-bing"
)

type Mtr struct {
	Host  string
	Send  int
	Loss  int
	Last  time.Duration
	Avg   time.Duration
	Best  time.Duration
	Wrst  time.Duration
	StDev time.Duration
}

func RunMtr(Addr string, maxrtt time.Duration, maxttl int, maxTimeoutCount int) ([]Mtr, error) {
	result := []Mtr{}
	mtr := map[int]*probing.Statistics{}
	timeoutCount := 0

	for ttl := 1; ttl <= maxttl; ttl++ {
		pinger, err := probing.NewPinger(Addr)
		if err != nil {
			return result, err
		}

		pinger.SetPrivileged(g.Cfg.PingPrivilege)
		pinger.Timeout = maxrtt
		pinger.Count = 10
		pinger.TTL = ttl
		pinger.Size = 64

		err = pinger.Run()
		if err != nil {
			return result, err
		}

		stats := pinger.Statistics()

		seelog.Debug("[func:RunMtr] Target Addr: ", Addr, " ttl: ", ttl, " stats: ", stats)
		mtr[ttl] = stats
		if stats.PacketsRecv > 0 {
			timeoutCount = 0 // 一旦收到回复，重置超时计数器
			if net.ParseIP(stats.IPAddr.String()).Equal(net.ParseIP(Addr)) {
				break
			}
		} else {
			timeoutCount++ // 没有收到回复，增加超时计数器
			if timeoutCount >= maxTimeoutCount {
				break // 如果连续超时次数达到阈值，停止测试
			}
		}

		if stats.PacketsSent == stats.PacketsRecv {
			break
		}
	}

	for _, stats := range mtr {
		imtr := Mtr{}

		imtr.Host = stats.IPAddr.String()
		if imtr.Host == "<nil>" {
			imtr.Host = "???"
		}

		imtr.Send = stats.PacketsSent
		imtr.Loss = stats.PacketsSent - stats.PacketsRecv
		if stats.PacketsRecv > 0 {
			imtr.Last = stats.Rtts[stats.PacketsRecv-1]
		} else {
			imtr.Last = 0
		}
		imtr.Avg = stats.AvgRtt
		imtr.Best = stats.MinRtt
		imtr.Wrst = stats.MaxRtt
		imtr.StDev = stats.StdDevRtt

		result = append(result, imtr)
	}
	seelog.Debug("[func:RunMtr] result: ", result)
	return result, nil
}

/*
func RunTrace(Addr string, maxrtt time.Duration, maxttl int, maxtimeout int) ([]ICMP, error) {
	hops := make([]ICMP, 0, maxttl)
	var res pkg
	var err error
	res.dest, err = net.ResolveIPAddr("ip", Addr)
	if err != nil {
		return nil, errors.New("Unable to resolve destination host")
	}
	res.maxrtt = maxrtt
	//res.id = rand.Int() % 0x7fff
	res.id = rand.Intn(65535)
	res.seq = 1
	res.msg = icmp.Message{Type: ipv4.ICMPTypeEcho, Code: 0, Body: &icmp.Echo{ID: res.id, Seq: res.seq}}
	res.netmsg, err = res.msg.Marshal(nil)
	if nil != err {
		return nil, err
	}
	timeouts := 0
	for i := 1; i <= maxttl; i++ {
		next := res.Send(i)
		next.MaxRTT = next.RTT
		next.MinRTT = next.RTT
		for j := 0; j < 2; j++ {
			tnext := res.Send(i)
			if tnext.RTT >= next.RTT {
				next.MaxRTT = tnext.RTT
			}
			if tnext.MinRTT <= next.RTT {
				next.MinRTT = tnext.RTT
			}
		}
		next.AvgRTT = time.Duration((next.MaxRTT + next.RTT + next.MinRTT) / 3)
		hops = append(hops, next)
		if next.Final {
			break
		}
		if next.Timeout {
			timeouts++
		} else {
			timeouts = 0
		}
		if timeouts == maxtimeout {
			break
		}
	}
	return hops, nil
}
*/
