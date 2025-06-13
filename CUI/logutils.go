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
	// debugMode は main.go で管理され、引数で渡される
)

// setupLogging: ロギングシステムを初期化・設定する
// destDir: ログファイルの出力先ディレクトリ (logToFile が true の場合)
// startTime: プログラム開始時刻 (ログファイル名に使用)
// logToFileFlag: ログをファイルにも書き出すかのフラグ (main から)
// debugModeFlag: デバッグモードを有効にするかのフラグ (main から)
func setupLogging(destDir string, startTime time.Time, logToFileFlag bool, debugModeFlag bool) {
	// --- 標準ロガー (logger) の設定 ---
	var logOutput io.Writer = os.Stdout
	logFilePath := ""

	if logToFileFlag {
		// ログディレクトリのチェックは setupLogging 内で行う
		// checkLogDir はディレクトリが存在しない場合や書き込めない場合に false を返す
		if checkLogDir(destDir) {
			logFileName := fmt.Sprintf("GoTransAV1_Log_%s.log", startTime.Format("20060102_150405"))
			logFilePath = filepath.Join(destDir, logFileName)
			var err error
			// OpenFile で追記モード (O_APPEND) を使用
			logFile, err = os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				// ログファイルが開けなかった場合は標準エラーに警告を出し、標準出力のみを使用
				log.Printf("警告: ログファイル '%s' を開けません (%v)。ログは標準出力にのみ出力されます。\n", logFilePath, err)
				logFile = nil // logFile を nil に設定
			} else {
				// ログファイルが開けた場合、標準出力とファイルの両方に書き込む MultiWriter を設定
				logOutput = io.MultiWriter(os.Stdout, logFile)
				// どのファイルにログが出力されるか標準出力に表示 (logger が初期化される前なので log.Printf を使用)
				log.Printf("ログを '%s' にも出力します。\n", logFilePath)
			}
		} else {
			// checkLogDir が false を返した場合 (ディレクトリがない、書き込めないなど)
			// 警告は checkLogDir 内で出力されるので、ここでは何もしない
			// logOutput は os.Stdout のまま
		}
	}
	// logger を初期化 (logOutput は logToFileFlag と checkLogDir の結果によって決まる)
	logger = log.New(logOutput, "", log.Ldate|log.Ltime) // プレフィックスなし、日付と時刻を表示

	// --- デバッグロガー (debugLogger) の設定 ---
	var debugOutput io.Writer = io.Discard // デフォルトではデバッグログを破棄
	// 引数で渡された debugModeFlag を使用
	if debugModeFlag {
		// デバッグモードが有効な場合
		if logFile != nil {
			// ログファイルが有効なら、標準出力とログファイルの両方にデバッグログを出力
			debugOutput = io.MultiWriter(os.Stdout, logFile)
		} else {
			// ログファイルが無効なら、標準出力にのみデバッグログを出力
			debugOutput = os.Stdout
		}
	}
	// debugLogger を初期化 (debugOutput は debugModeFlag の状態によって決まる)
	debugLogger = log.New(debugOutput, "[DEBUG] ", log.Ldate|log.Ltime|log.Lshortfile) // プレフィックス、日付、時刻、ファイル名と行番号を表示

	// --- 初期ログメッセージ ---
	logger.Println("--- ロギング開始 ---")
	// 引数で渡された debugModeFlag を使用してデバッグモードの有効/無効をログに出力
	if debugModeFlag {
		debugLogPrintf("デバッグモード有効") // debugLogPrintf は内部で debugModeFlag をチェックするので冗長かもしれないが、明示的に
	}
	if logFile != nil {
		// ログファイルが正常に開かれた場合のみ、ファイルパスをログに出力
		logger.Printf("ログファイル: %s", logFilePath)
	}
}

// checkLogDir: ログ出力先ディレクトリの存在と書き込み権限を確認する
// 戻り値: ディレクトリが存在し書き込み可能なら true, そうでなければ false
func checkLogDir(dirPath string) bool {
	info, err := os.Stat(dirPath)
	if os.IsNotExist(err) {
		// ディレクトリが存在しない場合
		log.Printf("警告: ログ出力先ディレクトリ '%s' が存在しません。ログファイルは作成されません。\n", dirPath)
		return false
	}
	if err != nil {
		// Stat でその他のエラーが発生した場合 (権限など)
		log.Printf("警告: ログ出力先ディレクトリ '%s' の状態確認エラー (%v)。ログファイルは作成されません。\n", dirPath, err)
		return false
	}
	if !info.IsDir() {
		// パスは存在するがディレクトリではない場合
		log.Printf("警告: ログ出力先パス '%s' はディレクトリではありません。ログファイルは作成されません。\n", dirPath)
		return false
	}

	// ディレクトリが存在し、ディレクトリであることが確認できたので、書き込み権限をテストする
	// 一時ファイルを作成してみて、エラーが発生しないか確認する
	tempFile, err := os.CreateTemp(dirPath, ".logcheck_")
	if err != nil {
		// 一時ファイルの作成に失敗した場合 (書き込み権限がない可能性が高い)
		log.Printf("警告: ログ出力先ディレクトリ '%s' に書き込めません (%v)。ログファイルは作成されません。\n", dirPath, err)
		return false
	}
	// 書き込みテストが成功した場合、作成した一時ファイルを閉じて削除する
	tempFile.Close()
	os.Remove(tempFile.Name())

	// ディレクトリが存在し、書き込み可能であるため true を返す
	return true
}

// debugLogPrintf: デバッグログを出力するヘルパー関数
// この関数は main.go の debugMode フラグに依存しなくなった
func debugLogPrintf(format string, v ...interface{}) {
	// debugLogger が初期化されており、かつ出力先が io.Discard でない場合にログを出力
	// (debugMode が false の場合、setupLogging で debugLogger の出力先が io.Discard に設定される)
	if debugLogger != nil && debugLogger.Writer() != io.Discard {
		// Output(2, ...) で呼び出し元の情報を正しく表示させる
		debugLogger.Output(2, fmt.Sprintf(format, v...))
	}
}

// closeLogFile: プログラム終了時にログファイルを閉じる (現在は main の defer で一時ディレクトリ削除を行うため、明示的な呼び出しは不要)
// 必要であれば main の最後で呼び出すことも可能
func closeLogFile() {
	if logFile != nil {
		logger.Println("--- ロギング終了 ---")
		logFile.Close()
		logFile = nil // クローズしたことを示す
	}
}
