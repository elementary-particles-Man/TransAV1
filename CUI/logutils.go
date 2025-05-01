package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

// --- グローバル変数 (ロギング関連) ---
var (
	logger      *log.Logger // 通常ログ用ロガー (標準出力 +/- ファイル)
	debugLogger *log.Logger // デバッグログ用ロガー (デバッグモード時のみ有効)
	logFile     *os.File    // ログファイルオブジェクト (ファイル出力時)
	// debugMode   bool        // ★★★ 削除: main.go のグローバル変数 debugMode を直接参照する ★★★
)

// setupLogging: ロギングシステムを初期化・設定する
// destDir: ログファイルの出力先ディレクトリ (logToFile が true の場合)
// startTime: プログラム開始時刻 (ログファイル名に使用)
// logToFileFlag: ログをファイルにも書き出すかのフラグ (main から)
// debugModeFlag: デバッグモードを有効にするかのフラグ (main から) -> ★★★ 引数名は残すが、グローバル変数 debugMode を直接使う ★★★
func setupLogging(destDir string, startTime time.Time, logToFileFlag bool, debugModeFlag bool /* 引数名は残すが、内部では main.debugMode を参照 */) {
	// main から受け取ったデバッグフラグを使用 (グローバル変数 main.debugMode)
	// debugMode = debugModeFlag // ★★★ この行を削除 ★★★

	// --- 標準ロガー (logger) の設定 ---
	var logOutput io.Writer = os.Stdout
	logFilePath := ""

	if logToFileFlag {
		if checkLogDir(destDir) {
			logFileName := fmt.Sprintf("GoTransAV1_Log_%s.log", startTime.Format("20060102_150405"))
			logFilePath = filepath.Join(destDir, logFileName)
			var err error
			logFile, err = os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				log.Printf("警告: ログファイル '%s' を開けません (%v)。ログは標準出力にのみ出力されます。\n", logFilePath, err)
				logFile = nil
			} else {
				logOutput = io.MultiWriter(os.Stdout, logFile)
				log.Printf("ログを '%s' にも出力します。\n", logFilePath)
			}
		}
	}
	logger = log.New(logOutput, "", log.Ldate|log.Ltime)

	// --- デバッグロガー (debugLogger) の設定 ---
	var debugOutput io.Writer = io.Discard
	// ★★★ main.go のグローバル変数 debugMode を直接参照 ★★★
	if debugMode {
		if logFile != nil {
			debugOutput = io.MultiWriter(os.Stdout, logFile)
		} else {
			debugOutput = os.Stdout
		}
	}
	debugLogger = log.New(debugOutput, "[DEBUG] ", log.Ldate|log.Ltime|log.Lshortfile)

	// --- 初期ログメッセージ ---
	logger.Println("--- ロギング開始 ---")
	// ★★★ main.go のグローバル変数 debugMode を直接参照 ★★★
	if debugMode {
		debugLogPrintf("デバッグモード有効")
	}
	if logFile != nil {
		logger.Printf("ログファイル: %s", logFilePath)
	}
}

// checkLogDir: ログ出力先ディレクトリの存在と書き込み権限を確認する
func checkLogDir(dirPath string) bool {
	info, err := os.Stat(dirPath)
	if os.IsNotExist(err) {
		log.Printf("警告: ログ出力先ディレクトリ '%s' が存在しません。ログファイルは作成されません。\n", dirPath)
		return false
	}
	if err != nil {
		log.Printf("警告: ログ出力先ディレクトリ '%s' の状態確認エラー (%v)。ログファイルは作成されません。\n", dirPath, err)
		return false
	}
	if !info.IsDir() {
		log.Printf("警告: ログ出力先パス '%s' はディレクトリではありません。ログファイルは作成されません。\n", dirPath)
		return false
	}
	tempFile, err := os.CreateTemp(dirPath, ".logcheck_")
	if err != nil {
		log.Printf("警告: ログ出力先ディレクトリ '%s' に書き込めません (%v)。ログファイルは作成されません。\n", dirPath, err)
		return false
	}
	tempFile.Close()
	os.Remove(tempFile.Name())
	return true
}

// debugLogPrintf: デバッグログを出力するヘルパー関数
func debugLogPrintf(format string, v ...interface{}) {
	// ★★★ main.go のグローバル変数 debugMode を直接参照 ★★★
	if debugMode && debugLogger != nil {
		debugLogger.Output(2, fmt.Sprintf(format, v...))
	}
}

// closeLogFile: プログラム終了時にログファイルを閉じる (現在は未使用)
// func closeLogFile() {
// 	if logFile != nil {
// 		logger.Println("--- ロギング終了 ---")
// 		logFile.Close()
// 	}
// }
