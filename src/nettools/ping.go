package nettools

import (
	"errors"
	"net"
	"time"

	"github.com/a76yyyy/smartping/src/g"
	"github.com/cihub/seelog"

	probing "github.com/prometheus-community/pro-bing"
)

type ICMP struct {
	Addr    net.Addr
	RTT     time.Duration
	MaxRTT  time.Duration
	MinRTT  time.Duration
	AvgRTT  time.Duration
	Final   bool
	Timeout bool
	Down    bool
	Error   error
}

func SinglePing(TargetIPAddr *net.IPAddr, Timeout time.Duration, MaxTTL int) (float64, error) {
	addr := TargetIPAddr.String()
	pinger, err := probing.NewPinger(addr)
	if err != nil {
		return 0, err
	}

	pinger.SetPrivileged(g.Cfg.PingPrivilege)
	pinger.Timeout = Timeout
	pinger.Count = 1
	pinger.TTL = MaxTTL
	pinger.Size = 64 // 设置为原始代码中的数据大小 (16 * 4 = 64 bytes)

	err = pinger.Run()
	if err != nil {
		return 0, err
	}

	stats := pinger.Statistics()
	if len(stats.Rtts) > 0 {
		return float64(stats.Rtts[0].Nanoseconds()) / 1e6, nil
	}

	return 0, errors.New("no Reply")
}

// GeneralPing 是一个通用的 Ping 函数，可以被多个任务共用
func GeneralPing(TargetIPAddr *net.IPAddr, Count int, Timeout time.Duration, MaxTTL int) (g.PingSt, error) {
	stats := g.PingSt{
		MinDelay: -1,
	}

	pinger, err := probing.NewPinger(TargetIPAddr.String())
	if err != nil {
		return stats, err
	}

	pinger.SetPrivileged(g.Cfg.PingPrivilege)
	pinger.Timeout = Timeout
	pinger.Count = Count
	pinger.TTL = MaxTTL
	pinger.Size = 64 // 设置为原始代码中的数据大小 (16 * 4 = 64 bytes)

	err = pinger.Run()
	if err != nil {
		return stats, err
	}

	pingStats := pinger.Statistics()

	// 计算统计信息
	for _, rtt := range pingStats.Rtts {
		delay := float64(rtt.Nanoseconds()) / 1e6
		stats.AvgDelay += delay
		if stats.MaxDelay < delay {
			stats.MaxDelay = delay
		}
		if stats.MinDelay == -1 || stats.MinDelay > delay {
			stats.MinDelay = delay
		}
	}

	stats.SendPk = pingStats.PacketsSent
	stats.RevcPk = pingStats.PacketsRecv
	stats.LossPk = int(pingStats.PacketLoss)

	if stats.RevcPk > 0 {
		stats.AvgDelay /= float64(stats.RevcPk)
	} else {
		stats.AvgDelay = 0
	}
	seelog.Debug("[func:GeneralPing] Target IP: ", TargetIPAddr.String(),
		" SendPk: ", stats.SendPk,
		" RevcPk: ", stats.RevcPk,
		" LossPk: ", stats.LossPk,
		" AvgDelay: ", stats.AvgDelay,
		" MaxDelay: ", stats.MaxDelay,
		" MinDelay: ", stats.MinDelay)
	return stats, nil
}

func UniformPing(TargetIPAddr *net.IPAddr, Count int, Timeout time.Duration, MaxTTL int) (g.PingSt, error) {
	stats := g.PingSt{
		MinDelay: -1,
	}
	lossPk := 0

	for i := 0; i < Count; i++ {
		start := time.Now().UnixNano()
		delay, err := SinglePing(TargetIPAddr, Timeout, MaxTTL)
		if err == nil {
			stats.AvgDelay += delay
			if stats.MaxDelay < delay {
				stats.MaxDelay = delay
			}
			if stats.MinDelay == -1 || stats.MinDelay > delay {
				stats.MinDelay = delay
			}
			stats.RevcPk++
			seelog.Debug("[func:UniformPing] Target IP: ", TargetIPAddr, " ID: ", i, " delay: ", delay)
		} else {
			lossPk++
			seelog.Error("[func:UniformPing] Target IP: ", TargetIPAddr, " ID: ", i, " err: ", err)
		}
		stats.SendPk++
		duringTime := time.Now().UnixNano() - start
		time.Sleep(time.Duration(Timeout-time.Duration(duringTime)) * time.Nanosecond)
	}
	stats.LossPk = int(float64(lossPk) / float64(stats.SendPk) * 100)
	if stats.RevcPk > 0 {
		stats.AvgDelay /= float64(stats.RevcPk)
	} else {
		stats.AvgDelay = 0
	}
	seelog.Debug("[func:UniformPing] Target IP: ", TargetIPAddr.String(),
		" SendPk: ", stats.SendPk,
		" RevcPk: ", stats.RevcPk,
		" LossPk: ", stats.LossPk,
		" AvgDelay: ", stats.AvgDelay,
		" MaxDelay: ", stats.MaxDelay,
		" MinDelay: ", stats.MinDelay)
	return stats, nil
}
