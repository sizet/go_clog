// ©.
// https://github.com/sizet/go_clog

package clog

import (
    "os"
    "fmt"
    "time"
    "sync"
    "errors"
    "runtime"
    "strings"
)

// 設定資料.
type CLogConfigInfo struct {
    // 是否輸出訊息到標準輸出, false = 否, true = 是.
    OutputToStdout bool
    // 是否輸出訊息到檔案, false = 否, true = 是.
    OutputToFile bool
    // 檔案的路徑.
    OutputFilePath string
    // 是否滾動檔案, false = 否, true = 是.
    RotateFile bool
    // 檔案大小多大時做滾動.
    RotateSize int
    // 保留的滾動檔案的數目.
    RotateCnt int
    // 是否顯示各種類型訊息, false = 否, true = 是.
    ShowErr bool
    ShowWrn bool
    ShowInf bool
    ShowDbg bool
}

const (
    // CLogConfigInfo.RotateSize 的範圍限制 (byte).
    minRotateSize = int(1 * 1024 * 1024)
    maxRotateSize = int(256 * 1024 * 1024)

    // CLogConfigInfo.RotateCnt 範圍限制.
    minRotateCnt = int(1)
    maxRotateCnt = int(99)

    // 開檔案的參數.
    openFlag = int(os.O_CREATE | os.O_RDWR | os.O_APPEND)
    openMode = os.FileMode(0666)

    // 訊息的時間的格式.
    timeFmt = string("2006/01/02-15:04:05")

    // 滾動的檔案的檔名格式.
    // 例如 :
    // CLogConfigInfo.OutputFilePath = log.txt,
    // 滾動的檔案的檔名 = log.txt.01, log.txt.02, ...
    rotateFmt = string("%s.%02d")

    // 訊息的前綴 (日期時間 檔名(行數).類型).
    prefixFmt = string("%s %s(%04d).%s: ")
)

// 顯示的訊息的類型 (msgTypeList).
const (
    // 錯誤訊息.
    MsgErr = iota
    // 警告訊息.
    MsgWrn
    // 一般訊息.
    MsgInf
    // 除錯訊息.
    MsgDbg
)
// 訊息類型的前綴.
var msgTypePrefixList = [...]string {
    "ERR",
    "WRN",
    "INF",
    "DBG",
}
// 哪些類型的訊息要顯示, false = 不輸出, true = 要輸出.
var msgTypeShowList = [...]bool {
    true,
    false,
    false,
    false,
}

var cLogCfg CLogConfigInfo

var fileRD *os.File

var fileMutex sync.Mutex




// 顯示訊息到標準輸出.
// 參數 :
// msgType
//   訊息類型 (msgTypeList).
// msgFmt
//   訊息格式.
// msgArg
//   訊息參數.
func PrintMsg(
    msgType int,
    msgFmt string,
    msgArg ...interface{}) {

    var filePath, msgData string
    var fileLine, nameIdx int

    if msgTypeShowList[msgType] == true {
        // 取得 call stack 的檔案路徑和第幾行, 1 是找到上 1 層.
        _, filePath, fileLine, _ = runtime.Caller(1)
        // 只取檔名部分.
        nameIdx = strings.LastIndex(filePath, "/") + 1
        // 填充訊息的前綴.
        msgData = fmt.Sprintf(prefixFmt,
                              time.Now().Format(timeFmt),  filePath[nameIdx:], fileLine,
                              msgTypePrefixList[msgType])
        // 顯示訊息.
        fmt.Printf(msgData + msgFmt + "\n", msgArg...)
    }
}

// 檢查檔案是存在.
// 參數 :
// filePath
//   檔案路徑.
// 回傳 :
// isExist
//   是否存在, false = 否, true = 是.
// fErr :
//   是否發生錯誤, nil = 否, not nil = 是.
func checkFileExist(
    filePath string)(
    isExist bool,
    fErr error) {

    _, fErr = os.Stat(filePath)
    if fErr == nil {
        isExist = true
    } else {
        if os.IsNotExist(fErr) == true {
            fErr = nil
        } else {
            PrintMsg(MsgErr, "call os.Stat(%s) fail [%s]", filePath, fErr.Error())
            return
        }
    }

    return
}

// 檢查和操作檔案滾動.
// 參數 :
// msgLen
//   訊息長度.
// 回傳 :
// fErr :
//   是否發生錯誤, nil = 否, not nil = 是.
func doRotate(
    msgLen int)(
    fErr error) {

    var osFileInfo os.FileInfo
    var usableNum int
    var isExist bool
    var oldFilePath, newFilePath string

    // 取得檔案的大小.
    osFileInfo, fErr = os.Stat(cLogCfg.OutputFilePath)
    if fErr != nil {
        PrintMsg(MsgErr, "call os.Stat(%s) fail [%s]", cLogCfg.OutputFilePath, fErr.Error())
        return
    }
    // 檢查是否達到檔案大小上限, 是的話要滾動檔案.
    if (int64(msgLen) + osFileInfo.Size()) < int64(cLogCfg.RotateSize) {
        return
    }

    // 滾動檔案之前先找到空著的編號.
    // 例如 :
    // CLogConfigInfo.RotateCnt = 4, 目前有 log.txt.01, log.txt.02, 空號就是 3.
    for usableNum = 1;; usableNum++ {
        // 從 1 開始檢查空著的編號.
        oldFilePath = fmt.Sprintf(rotateFmt, cLogCfg.OutputFilePath, usableNum)
        isExist, fErr = checkFileExist(oldFilePath)
        if fErr != nil {
            PrintMsg(MsgErr, "call checkFileExist() fail")
            return
        }
        // 有找到.
        if isExist == false {
            break
        }
        // 如果達到保留的滾動檔案的數目的上限, 把最後的檔案刪除.
        // 例如 :
        // CLogConfigInfo.RotateCnt = 4, 目前有 log.txt.01, log.txt.02, ..., log.txt.04,
        // 沒有可用的空號, 必須把 log.txt.04 移除.
        if usableNum >= cLogCfg.RotateCnt {
            fErr = os.Remove(oldFilePath)
            if fErr != nil {
                PrintMsg(MsgErr, "call os.Remove(%s) fail [%s]", oldFilePath, fErr.Error())
                return
            }
            // 空號就是最後.
            break
        }
    }

    // 開始滾動保留的檔案.
    // 例如 :
    // 現在有 log.txt.01, log.txt.02, 滾動後是 log.txt.02, log.txt.03.
    for ; usableNum > 1; usableNum-- {
        oldFilePath = fmt.Sprintf(rotateFmt, cLogCfg.OutputFilePath, usableNum - 1)
        newFilePath = fmt.Sprintf(rotateFmt, cLogCfg.OutputFilePath, usableNum)
        fErr = os.Rename(oldFilePath, newFilePath)
        if fErr != nil {
            PrintMsg(MsgErr, "call os.Rename(%s, %s) fail [%s]",
                     oldFilePath, newFilePath, fErr.Error())
            return
        }
    }

    // 把目前使用的檔案滾動到保留的檔案.
    // 例如 :
    // log.txt 滾動後是 log.txt.01.
    fileRD.Close()
    fileRD = nil
    newFilePath = fmt.Sprintf(rotateFmt, cLogCfg.OutputFilePath, usableNum)
    fErr = os.Rename(cLogCfg.OutputFilePath, newFilePath)
    if fErr != nil {
        PrintMsg(MsgErr, "call os.Rename(%s, %s) fail [%s]",
                 cLogCfg.OutputFilePath, newFilePath, fErr.Error())
        return
    }

    // 重新開檔案.
    fileRD, fErr = os.OpenFile(cLogCfg.OutputFilePath, openFlag, openMode)
    if fErr != nil {
        PrintMsg(MsgErr, "call os.OpenFile(%s) fail [%s]", fileRD, fErr.Error())
        return
    }

    return
}

// 顯示訊息.
// 參數 :
// msgType
//   訊息類型 (msgTypeList).
// msgFmt
//   訊息格式.
// msgArg
//   訊息參數.
// 回傳 :
// fErr :
//   是否發生錯誤, nil = 否, not nil = 是.
func LogMsg(
    msgType int,
    msgFmt string,
    msgArg ...interface{})(
    fErr error) {

    var filePath, msgData string
    var fileLine, nameIdx int

    if msgTypeShowList[msgType] == true {
        fileMutex.Lock()
        defer fileMutex.Unlock()

        if (cLogCfg.OutputToStdout == true) || (cLogCfg.OutputToFile == true) {
            // 取得 call stack 的檔案路徑和第幾行, 1 是找到上 1 層.
            _, filePath, fileLine, _ = runtime.Caller(1)
            // 只取檔名部分.
            nameIdx = strings.LastIndex(filePath, "/") + 1
            // 填充訊息的前綴.
            msgData = fmt.Sprintf(prefixFmt,
                                  time.Now().Format(timeFmt), filePath[nameIdx:], fileLine,
                                  msgTypePrefixList[msgType])
            // 組合完整的訊息.
            msgData = fmt.Sprintf(msgData + msgFmt + "\n", msgArg...)
        }
        // 顯示到標準輸出.
        if cLogCfg.OutputToStdout == true {
            fmt.Printf(msgData)
        }
        // 顯示到檔案.
        if cLogCfg.OutputToFile == true {
            // 檢查是否需要滾動.
            if cLogCfg.RotateFile == true {
                fErr = doRotate(len(msgData))
                if fErr != nil {
                    PrintMsg(MsgErr, "call doRotate() fail")
                    return
                }
            }
            _, fErr = fileRD.WriteString(msgData)
            if fErr != nil {
                PrintMsg(MsgErr, "call os.WriteString(%s) fail [%s]", msgData, fErr.Error())
            }
        }
    }

    return
}

// 設定哪種類型的訊息要顯示或不顯示.
// 參數 :
// argCLogCfg
//   clog 的設定資料.
func ChangeShow(
    argCLogCfg CLogConfigInfo) {

    msgTypeShowList[MsgDbg] = argCLogCfg.ShowDbg
    msgTypeShowList[MsgInf] = argCLogCfg.ShowInf
    msgTypeShowList[MsgWrn] = argCLogCfg.ShowWrn
    msgTypeShowList[MsgErr] = argCLogCfg.ShowErr

    return
}

// 初始化.
// 參數 :
// argCLogCfg
//   clog 的設定資料.
// 回傳 :
// fErr :
//   是否發生錯誤, nil = 否, not nil = 是.
func DoInit(
    argCLogCfg CLogConfigInfo)(
    fErr error) {

    var tmpMsg string

    defer func() {
        if fErr != nil {
			if fileRD != nil {
                fileRD.Close()
                fileRD = nil
            }
        }
    }()

    if argCLogCfg.OutputToFile == true {
        fileRD, fErr = os.OpenFile(argCLogCfg.OutputFilePath, openFlag, openMode)
        if fErr != nil {
            PrintMsg(MsgErr, "invalid OutputFilePath, call os.OpenFile(%s) fail [%s]",
                     argCLogCfg.OutputFilePath, fErr.Error())
            return
        }

        if argCLogCfg.RotateFile == true {
            // 檢查 CLogConfigInfo.RotateSize 的範圍.
            if (argCLogCfg.RotateSize < minRotateSize) || (maxRotateSize < argCLogCfg.RotateSize) {
                tmpMsg = fmt.Sprintf("invalid RotateSize [%d], must be [%d]~[%d]",
                                     argCLogCfg.RotateSize, minRotateSize, maxRotateSize)
                PrintMsg(MsgErr, tmpMsg)
                fErr = errors.New(tmpMsg)
                return
            }

            // 檢查 CLogConfigInfo.RotateCnt 的範圍.
            if (argCLogCfg.RotateCnt < minRotateCnt) || (maxRotateCnt < argCLogCfg.RotateCnt) {
                tmpMsg = fmt.Sprintf("invalid RotateCnt [%d], must be [%d]~[%d]",
                                     argCLogCfg.RotateCnt, minRotateCnt, maxRotateCnt)
                PrintMsg(MsgErr, tmpMsg)
                fErr = errors.New(tmpMsg)
                return
            }
        }
    }

    cLogCfg = argCLogCfg
    ChangeShow(cLogCfg)

    return
}

// 結束前的清理.
func DoExit() {
    if fileRD != nil {
        fileRD.Close()
        fileRD = nil
    }
}
