package main

import (
	"fmt" // fmt パッケージをインポート
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

// --- グローバル変数 (ロギング関連) ---
var (
	logger      *log.Logger
	debugLogger *log.Logger
)

// setupLogging: ロギングを設定
// main 関数から必要な情報を引数として受け取る
func setupLogging(destDir string, startTime time.Time, logToFile bool, debugMode bool) {
	// 標準出力用ロガーとデバッグ用ロガーの初期化
	stdLogger := log.New(os.Stdout, "", log.Ldate|log.Ltime)

	// デバッグロガーはデフォルトでは破棄 (io.Discard)
	debugLogger = log.New(io.Discard, "[DEBUG] ", log.Ldate|log.Ltime|log.Lshortfile)
	if debugMode {
		// デバッグモード有効なら標準出力に設定
		debugLogger.SetOutput(os.Stdout)
	}

	logFilePath := ""
	// logToFile フラグが true の場合のみファイル出力設定を試みる
	if logToFile {
		// 出力ディレクトリの状態を確認
		info, err := os.Stat(destDir)
		if os.IsNotExist(err) {
			log.Printf("警告: 出力ディレクトリ '%s' が存在しないため、ログファイルは作成されません。\n", destDir)
		} else if err != nil {
			log.Printf("警告: 出力ディレクトリ '%s' の状態確認エラー (%v)。ログファイルは作成されません。\n", destDir, err)
		} else if !info.IsDir() {
			log.Printf("警告: 出力パス '%s' はディレクトリではありません。ログファイルは作成されません。\n", destDir)
		} else {
			// ディレクトリが存在し、ディレクトリであることを確認
			// 書き込み権限チェックのために一時ファイルを作成してみる
			tempFile, err := os.CreateTemp(destDir, "logcheck_")
			if err != nil {
				log.Printf("警告: 出力ディレクトリ '%s' に書き込めません (%v)。ログファイルは作成されません。\n", destDir, err)
			} else {
				// 書き込み可能なら一時ファイルを閉じて削除
				tempFile.Close()
				os.Remove(tempFile.Name())

				// ログファイルパスを生成
				logFileName := fmt.Sprintf("GoTransAV1_Log_%s.log", startTime.Format("20060102_150405"))
				logFilePath = filepath.Join(destDir, logFileName)
			}
		}
	}

	// ログファイルパスが有効ならファイルを開いて MultiWriter を設定
	if logFilePath != "" {
		logFile, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Printf("警告: ログファイル '%s' を開けません (%v)。ログは標準出力にのみ出力されます。\n", logFilePath, err)
			logger = stdLogger // ファイルが開けなければ標準ロガーを使用
			// デバッグログは既に設定された出力先 (標準出力 or io.Discard) のまま
		} else {
			// 標準出力とファイルの両方に出力
			multiWriter := io.MultiWriter(os.Stdout, logFile)
			logger = log.New(multiWriter, "", log.Ldate|log.Ltime)
			log.Printf("ログを '%s' にも出力します。\n", logFilePath)
			// デバッグログもファイルに出力する場合 (標準出力にも出る)
			if debugMode {
				debugMultiWriter := io.MultiWriter(os.Stdout, logFile)
				debugLogger.SetOutput(debugMultiWriter)
				debugLogger.SetFlags(log.Ldate | log.Ltime | log.Lshortfile) // フラグ再設定
				debugLogger.SetPrefix("[DEBUG] ")                            // Prefix再設定
			}
			// 注意: logFile はプログラム終了まで閉じられない
			// defer logFile.Close() は main の外なのでここでは呼べない
		}
	} else {
		// ログファイルが無効な場合は標準出力のみ
		logger = stdLogger
		// デバッグログは既に設定された出力先 (標準出力 or io.Discard) のまま
	}
}

// debugLogPrintf: デバッグログ出力関数 (debugMode が true の場合のみ出力)
func debugLogPrintf(format string, v ...interface{}) {
	if debugMode && debugLogger != nil { // debugLogger が初期化されているか確認
		debugLogger.Printf(format, v...)
	}
}
