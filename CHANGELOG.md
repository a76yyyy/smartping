# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]

Nothing right now.

## [0.8.1] - 2024.11.03

- 增加企业微信告警 @Aquila
- 前端图表功能优化 @Aquila, @a76yyyy
- 使用 pro-bing 作为 ping 工具 @a76yyyy
- 修改为纯 GO 方案 SQLite @a76yyyy
- DB 锁替换为读写锁 @a76yyyy
- 添加跨平台自动构建脚本 @a76yyyy

## [0.8.0] - 2019.07.02

- 报警记录挪至Ping拓扑节点，traceroute功能变更为mtr
- 增加全国延迟测试(绘图)功能
- 增加检测工具(检测目标IP网络Ping)功能
- 增加云模式(中心端配置)
- 增加邮件报警功能(附带MTR发送
- 增加用户IP授权功能
- ping图由JS画图切换为由golang生成图片
- 重新设计系统配置界面
- 语言改为中文
- 支持修改数据存档天数

## [0.5.0] - 2018.04.02

- 增加traceroute功能，当报警时进行Traceroute检测
- 去除sysping功能，仅保留goping，并重写goping
- 正向Ping/反向Ping页面增加定时刷新功能
- 轻微修改配置面板显示
- 代码优化与重构

##  [0.4.1] - 2017.09.26

- 修订拓扑图自身节点显示红线BUG

## [0.4.0] - 2017.09.15

- 增加日志支持[基于seelog]
- 修订使用系统Ping命令的取值方法，支持Win/Linux/Mac
- 增加用户选择使用系统Ping或GoPing[基于go-fastping]的选项（推荐使用SysPing，若存在兼容性问题时选用-GoPing）
- Windows下增加后台运行服务支持[基于nssm],并可使用双击control.cmd运行SmartPing
- Linux下增加pack指令，可以快速将SmartPing进行编译并打包
- WebDashbord中AgentList更名为SmartPing NetWork
- 修订部分数据类型

## [0.3.1] - 2017.09.16

- 修订计算算法，每3s发送一个ping包,每分钟发送20个，计算其最高,最低,平均延迟、丢包率。
- API接口header变更为json
- 修复拓扑节点自动刷新失效问题

## [0.3.0] - 2017.09.16

- 全新的Dashbord设计，支持报警数据存储，配置WEB管理
- 更新数据结构，进行数据拆表
- 开放API接口，Dashbord数据全部采用API接口方式通过Ajax获取
- 修订了一大堆的BUG
- 项目官网上线smartping.org，文档上线docs.smartping.org
