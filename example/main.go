// ©.
// https://github.com/sizet/go_clog

package main

import (
    "os"
    "../clog"
)

func main() {
    var cLogCfg clog.CLogConfigInfo
    var cErr error

    // 設定參數.
    cLogCfg.OutputToStdout = true
    cLogCfg.OutputToFile = true
    cLogCfg.OutputFilePath = "log.txt"
    cLogCfg.RotateFile = true
    cLogCfg.RotateSize = 2 * 1024 * 1024
    cLogCfg.RotateCnt = 4
    cLogCfg.ShowErr = true
    cLogCfg.ShowWrn = true
    cLogCfg.ShowInf = false
    cLogCfg.ShowDbg = false

    // clog.PrintMsg() 可以不需要先 clog.DoInit() 就使用.
    // 在 clog.DoInit() 之前就使用要注意 clog.go 的 msgTypeShowList 的預設值,
    // 設定哪些類型的訊息要顯示.
    clog.PrintMsg(clog.MsgErr, "start %d", 0)

    // 先初始化.
    cErr = clog.DoInit(cLogCfg)
    if cErr != nil {
        clog.PrintMsg(clog.MsgErr, "call clog.DoInit() fail")
        os.Exit(1)
    }
    // 結束前釋放資源.
    defer clog.DoExit()

    // 使用 clog.LogMsg() 顯示訊息.
    clog.LogMsg(clog.MsgErr, "%s message %d", "error", 1)
    clog.LogMsg(clog.MsgWrn, "warning message 1")
    clog.LogMsg(clog.MsgInf, "information message 1")
    clog.LogMsg(clog.MsgDbg, "debug message 1")

    // 使用 clog.ChangeShow() 調整哪些類型的訊息要顯示.
    cLogCfg.ShowDbg = true
    cLogCfg.ShowInf = true
    cLogCfg.ShowWrn = true
    cLogCfg.ShowErr = true
    clog.ChangeShow(cLogCfg)

    clog.LogMsg(clog.MsgInf, "information message 2")
    clog.LogMsg(clog.MsgDbg, "debug message 2")

    os.Exit(0)
}
