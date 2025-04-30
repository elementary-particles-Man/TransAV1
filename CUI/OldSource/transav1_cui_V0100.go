package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime" // Usageヘルパー用
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	// Windows API 呼び出しに必要
	"golang.org/x/sys/windows"
)

// --- グローバル変数 ---
var (
	// コマンドラインフラグ
	sourceDir           string
	destDir             string
	ffmpegDir           string
	ffmpegPriority      string
	hwEncoder           string // 優先して試すHWエンコーダ名
	cpuEncoder          string // フォールバック用CPUエンコーダ名
	hwEncoderOptions    string // HWエンコーダ用の追加オプション
	cpuEncoderOptions   string // CPUエンコーダ用の追加オプション
	timeoutSeconds      int
	useTempFileListFlag bool // 一時ファイルリスト使用フラグ
	logToFile           bool
	debugMode           bool // 詳細ログ出力
	restart             bool // 開始時に *.failed などと 0byte 動画を削除
	forceStart          bool // 開始時に DestDir を確認付きで削除

	// 内部変数
	ffmpegPath        string // ffmpeg 実行ファイルのフルパス
	ffprobePath       string // ffprobe 実行ファイルのフルパス (存在すれば)
	startTime         time.Time
	logger            *log.Logger // 標準/ファイル出力用ロガー
	debugLogger       *log.Logger // デバッグ用ロガー (debugMode有効時のみアクティブ)
	usingTempFileList bool        // 実際に一時ファイルリストを使用しているか
	tempFileListPath  string      // 一時ファイルリストのパス

	// ファイル拡張子 (小文字)
	videoExtensions = map[string]struct{}{
		".mp4": {}, ".avi": {}, ".mov": {}, ".mkv": {}, ".wmv": {}, ".flv": {},
		".webm": {}, ".m4v": {}, ".mpeg": {}, ".mpg": {}, ".ts": {}, ".mts": {},
		".m2ts": {}, ".3gp": {}, ".asf": {}, ".divx": {},
	}
	imageExtensions = map[string]struct{}{
		".jpg": {}, ".jpeg": {}, ".png": {}, ".gif": {}, ".bmp": {}, ".tif": {},
		".tiff": {}, ".webp": {}, ".heic": {}, ".heif": {}, ".raw": {}, ".cr2": {},
		".nef": {}, ".orf": {}, ".sr2": {}, ".svg": {}, ".avif": {},
	}
	// restart で削除する拡張子
	failedMarkersToDelete = []string{".failed", ".timeout", ".error", ".unreadable"}
)

const (
	// 出力ファイル名のサフィックス
	outputSuffix = "_AV1.mp4"
	// 一時ディレクトリ名のプレフィックス
	tempDirPrefix = "go_transav1_"
)

// --- PART 1 END ---
// --- OS別 優先度設定 ---

// Windows用プロセス優先度設定 (SetPriorityClass API 使用)
func setWindowsPriorityAfterStart(process *os.Process, priority string) error {
	if process == nil {
		return errors.New("プロセスが nil です")
	}
	var priorityClass uint32
	switch strings.ToLower(priority) {
	case "idle":
		priorityClass = windows.IDLE_PRIORITY_CLASS
	case "belownormal":
		priorityClass = windows.BELOW_NORMAL_PRIORITY_CLASS
	case "normal":
		priorityClass = windows.NORMAL_PRIORITY_CLASS
	case "abovenormal":
		priorityClass = windows.ABOVE_NORMAL_PRIORITY_CLASS
	default:
		return fmt.Errorf("無効なWindows優先度指定: %s (idle, BelowNormal, Normal, AboveNormal)", priority)
	}

	debugLogPrintf("Windowsプロセス (PID: %d) の優先度を %s (0x%x) に設定試行...", process.Pid, priority, priorityClass)

	// プロセスハンドル取得 (PROCESS_SET_INFORMATION アクセス権が必要)
	handle, err := windows.OpenProcess(windows.PROCESS_SET_INFORMATION, false, uint32(process.Pid))
	if err != nil {
		// エラーハンドリングを改善: アクセス権限がない場合などの情報をログに出力
		errMsg := fmt.Sprintf("OpenProcess (PID: %d) 失敗: %v.", process.Pid, err)
		if errors.Is(err, windows.ERROR_ACCESS_DENIED) {
			errMsg += " プロセス優先度変更に必要な権限がない可能性があります。"
		}
		// このエラーは警告に留め、処理は続行させることも検討できる
		// return fmt.Errorf(errMsg)
		logger.Printf("警告: %s", errMsg) // 警告としてログ出力
		return nil                      // エラーは返さない
	}
	defer windows.CloseHandle(handle)

	// 優先度設定
	err = windows.SetPriorityClass(handle, priorityClass)
	if err != nil {
		// エラーハンドリングを改善
		errMsg := fmt.Sprintf("SetPriorityClass (PID: %d, Priority: 0x%x) 失敗: %v.", process.Pid, priorityClass, err)
		// このエラーも警告に留めることを検討
		// return fmt.Errorf(errMsg)
		logger.Printf("警告: %s", errMsg) // 警告としてログ出力
		return nil                      // エラーは返さない
	}

	debugLogPrintf("Windowsプロセス (PID: %d) の優先度設定成功。", process.Pid)
	return nil
}

// Unix (Linux/macOS) 用 nice 値設定
func getUnixNiceArgs(priority string) ([]string, error) {
	var niceValue int
	switch strings.ToLower(priority) {
	case "idle": // 最低
		niceValue = 19
	case "belownormal":
		niceValue = 10
	case "normal":
		niceValue = 0
	case "abovenormal": // より高い優先度 (低いnice値)
		niceValue = -5
	// より高い値は root 権限が必要な場合が多い
	default:
		return nil, fmt.Errorf("無効なUnix優先度指定: %s (idle, BelowNormal, Normal, AboveNormal)", priority)
	}
	// nice コマンドの引数を返す
	return []string{"nice", "-n", strconv.Itoa(niceValue)}, nil
}

// --- PART 2 END ---
// --- ログ設定 ---
func setupLogging() {
	// 標準出力用ロガーとデバッグ用ロガーの初期化
	stdLogger := log.New(os.Stdout, "", log.Ldate|log.Ltime)
	// デバッグ時はファイル名と行番号も出力
	debugLogger = log.New(io.Discard, "[DEBUG] ", log.Ldate|log.Ltime|log.Lshortfile)
	if debugMode {
		debugLogger.SetOutput(os.Stdout) // デバッグ有効なら標準出力に設定
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

// デバッグログ出力関数 (debugMode が true の場合のみ出力)
func debugLogPrintf(format string, v ...interface{}) {
	if debugMode {
		// debugLogger は setupLogging で適切に設定されているはず
		debugLogger.Printf(format, v...)
	}
}

// --- PART 3 END ---
// --- ffmpeg 実行 ---
type ffmpegResult struct {
	err      error
	timedOut bool
	exitCode int
}

func executeFFmpeg(ctx context.Context, inputPath string, outputPath string, tempDir string, encoder string, encoderSpecificOptions string) ffmpegResult {
	result := ffmpegResult{exitCode: -1} // 初期値

	baseCmd := ffmpegPath
	args := []string{
		"-hide_banner", // バナー非表示
		"-i", inputPath,
		"-c:v", encoder, // 引数で指定されたエンコーダを使用
		"-c:a", "aac", // 音声はAAC
		"-y", // 出力ファイルを上書き
	}
	// エンコーダ固有オプションを追加 (空でない場合)
	if encoderSpecificOptions != "" {
		// スペースで分割。引用符は考慮しないシンプルな分割
		opts := strings.Fields(encoderSpecificOptions)
		args = append(args, opts...)
	}
	// ログレベル
	if debugMode {
		args = append(args, "-loglevel", "error") // デバッグ時は error レベル
	} else {
		args = append(args, "-loglevel", "fatal") // 通常時は fatal レベル
	}
	args = append(args, outputPath) // 最後の引数として出力パス

	// --- OS別 優先度/nice設定 ---
	var finalArgs []string
	var cmd *exec.Cmd

	if runtime.GOOS == "windows" {
		finalArgs = args
		cmd = exec.CommandContext(ctx, baseCmd, finalArgs...)
		// Windows では SysProcAttr に特別な設定は不要 (API呼び出しのため)
	} else if runtime.GOOS == "linux" || runtime.GOOS == "darwin" {
		niceArgs, err := getUnixNiceArgs(ffmpegPriority)
		if err != nil {
			logger.Printf("警告: Unix優先度設定エラー: %v。デフォルト優先度で実行します。", err)
			finalArgs = args
			cmd = exec.CommandContext(ctx, baseCmd, finalArgs...) // nice なしで実行
		} else {
			// nice コマンド経由で ffmpeg を実行
			finalArgs = append(niceArgs, baseCmd) // nice -n X ffmpeg ...
			finalArgs = append(finalArgs, args...)
			cmd = exec.CommandContext(ctx, finalArgs[0], finalArgs[1:]...) // finalArgs[0] は "nice"
			debugLogPrintf("Unix niceコマンドを使用して実行: %v", finalArgs)
		}
	} else {
		logger.Printf("警告: 未対応OS (%s) のため、優先度設定はスキップされます。", runtime.GOOS)
		finalArgs = args
		cmd = exec.CommandContext(ctx, baseCmd, finalArgs...)
	}

	// --- 実行設定 ---
	if cmd.SysProcAttr == nil { // Linux/macOS で nice を使う場合などに必要
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	setOSSpecificAttrs(cmd.SysProcAttr) // OS固有の属性設定 (例: Windowsでコンソール非表示)

	// ffmpeg の標準出力/エラーをキャプチャするためのパイプ
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		result.err = fmt.Errorf("ffmpeg (%s) stderr パイプ作成エラー: %w", encoder, err)
		return result
	}
	stdoutPipe, err := cmd.StdoutPipe() // 標準出力も一応キャプチャ (loglevel fatal ならほぼ出ないはず)
	if err != nil {
		result.err = fmt.Errorf("ffmpeg (%s) stdout パイプ作成エラー: %w", encoder, err)
		return result
	}

	// --- 実行開始 ---
	logger.Printf("ffmpeg 実行開始 (%s): %s -> %s", encoder, filepath.Base(inputPath), filepath.Base(outputPath))
	debugLogPrintf("コマンド (%s): %s %s", encoder, cmd.Path, strings.Join(cmd.Args[1:], " ")) // デバッグ用にコマンド全体表示
	if err := cmd.Start(); err != nil {
		result.err = fmt.Errorf("ffmpeg (%s) プロセス開始エラー: %w", encoder, err)
		return result
	}

	// --- Windows 優先度設定 (プロセス開始後) ---
	if runtime.GOOS == "windows" {
		// 少し待ってから優先度を設定 (プロセス初期化のため)
		// この待機時間は環境によって調整が必要な場合がある
		time.Sleep(150 * time.Millisecond)
		if err := setWindowsPriorityAfterStart(cmd.Process, ffmpegPriority); err != nil {
			// 優先度設定エラーは警告に留める (既に関数内でログ出力される)
			// logger.Printf("警告: Windowsプロセス優先度設定失敗 (PID: %d): %v", cmd.Process.Pid, err)
		}
	}

	// --- 出力監視 (非同期) ---
	var ffmpegOutput strings.Builder
	stderrScanner := bufio.NewScanner(stderrPipe)
	stdoutScanner := bufio.NewScanner(stdoutPipe)
	stderrChan := make(chan struct{}) // stderr 監視終了通知用
	stdoutChan := make(chan struct{}) // stdout 監視終了通知用

	go func() {
		defer close(stderrChan) // ゴルーチン終了時にチャネルを閉じる
		for stderrScanner.Scan() {
			line := stderrScanner.Text()
			ffmpegOutput.WriteString(line + "\n")
			// エラーを含む行、またはデバッグモードならログ出力
			// より詳細なエラー判定が必要なら調整
			if debugMode || strings.Contains(strings.ToLower(line), "error") {
				logger.Printf("ffmpeg stderr (%s): %s", encoder, line)
			}
		}
	}()
	go func() {
		defer close(stdoutChan) // ゴルーチン終了時にチャネルを閉じる
		for stdoutScanner.Scan() {
			line := stdoutScanner.Text()
			ffmpegOutput.WriteString(line + "\n")
			// デバッグモードなら標準出力もログ出力
			if debugMode {
				logger.Printf("ffmpeg stdout (%s): %s", encoder, line)
			}
		}
	}()

	// --- プロセス終了待機 ---
	err = cmd.Wait()

	// --- 出力監視ゴルーチンの終了を待つ ---
	<-stderrChan
	<-stdoutChan

	// --- 結果処理 ---
	// コンテキストがキャンセルされたか（タイムアウト）
	if ctx.Err() == context.DeadlineExceeded {
		result.err = fmt.Errorf("ffmpeg (%s) タイムアウト (%d秒)", encoder, timeoutSeconds)
		result.timedOut = true
		result.exitCode = -2 // タイムアウトを示す内部コード
		// タイムアウト時にはプロセスが強制終了されているはずだが念のためKillを試みる
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
	} else if err != nil {
		// Wait() がエラーを返した場合 (タイムアウト以外)
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			// プロセスは終了したが、ゼロ以外のコードで終了
			result.exitCode = exitErr.ExitCode()
			// エラーメッセージに取得した出力を追加
			result.err = fmt.Errorf("ffmpeg (%s) 失敗 (ExitCode: %d)。\n--- ffmpeg出力 ---\n%s\n--- 出力終了 ---", encoder, result.exitCode, strings.TrimSpace(ffmpegOutput.String()))
		} else {
			// その他の実行時エラー (例: コマンドが見つからない、パイプエラーなど)
			result.err = fmt.Errorf("ffmpeg (%s) 実行時エラー: %w。\n--- ffmpeg出力 ---\n%s\n--- 出力終了 ---", encoder, err, strings.TrimSpace(ffmpegOutput.String()))
			result.exitCode = -3 // 実行時エラーを示す内部コード
		}
	} else {
		// 正常終了 (ExitCode 0)
		result.exitCode = 0
		debugLogPrintf("ffmpeg (%s) 正常終了 (ExitCode: 0)", encoder)
		// 正常終了時でも、デバッグモードなら出力をログに残す
		if debugMode && ffmpegOutput.Len() > 0 {
			debugLogPrintf("ffmpeg (%s) 正常終了時の出力:\n--- ffmpeg出力 ---\n%s\n--- 出力終了 ---", encoder, strings.TrimSpace(ffmpegOutput.String()))
		}
	}

	return result
}

// OS固有のSysProcAttr設定
func setOSSpecificAttrs(attr *syscall.SysProcAttr) {
	if runtime.GOOS == "windows" {
		// Windowsでコンソールウィンドウを表示しない
		attr.HideWindow = true
	}
	// Linux/macOS では通常不要
}

// --- PART 4 END ---
// --- ファイル処理 ---

// 入力パスに対応する出力パスを生成
func getOutputPath(inputFile, srcRoot, dstRoot string) (string, error) {
	// sourceDirからの相対パスを取得
	relPath, err := filepath.Rel(srcRoot, inputFile)
	if err != nil {
		// srcRoot が inputFile の親でない場合などにエラーになる
		return "", fmt.Errorf("相対パスの計算に失敗 ('%s' は '%s' の中にありませんか？): %w", inputFile, srcRoot, err)
	}

	// 拡張子を除去して新しいサフィックスを付与
	ext := filepath.Ext(relPath)
	baseNameWithoutExt := relPath // デフォルトは元の名前
	// 拡張子がある場合のみ、拡張子を除去
	if ext != "" {
		baseNameWithoutExt = relPath[:len(relPath)-len(ext)]
	} else if strings.HasPrefix(filepath.Base(relPath), ".") {
		// 拡張子がなく、ファイル名が '.' で始まる場合（Unix系隠しファイルなど）
		// そのままの名前を使う（サフィックス付与のため）
		baseNameWithoutExt = relPath
	}

	// 新しいベース名とサフィックスを結合
	outputBaseName := baseNameWithoutExt + outputSuffix

	// 出力先のサブディレクトリパスを計算
	outputRelDir := filepath.Dir(relPath) // 元の相対ディレクトリ構造
	finalDestDir := filepath.Join(dstRoot, outputRelDir)

	// 最終的な出力フルパスを結合
	return filepath.Join(finalDestDir, filepath.Base(outputBaseName)), nil
}

// 画像ファイルをコピーする関数
func copyImageFile(inputFile, outputFile string) error {
	// 入力ファイルの情報を取得（パーミッション維持のため）
	srcInfo, err := os.Stat(inputFile)
	if err != nil {
		return fmt.Errorf("入力ファイル '%s' の情報取得エラー: %w", inputFile, err)
	}
	if srcInfo.IsDir() {
		return fmt.Errorf("入力パス '%s' はディレクトリです", inputFile)
	}

	// 出力先にファイルが存在するかチェック
	if _, err := os.Stat(outputFile); err == nil {
		// ファイルが存在する場合、スキップ
		logger.Printf("スキップ (既存): %s", filepath.Base(outputFile))
		return nil // 正常終了として扱う
	} else if !os.IsNotExist(err) {
		// Statで "存在しない" 以外のエラーが発生した場合
		return fmt.Errorf("出力ファイル '%s' の確認エラー: %w", outputFile, err)
	}

	// 出力ディレクトリ作成 (MkdirAll は存在してもエラーにならない)
	outputDir := filepath.Dir(outputFile)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("出力ディレクトリ '%s' の作成エラー: %w", outputDir, err)
	}

	// コピー実行
	logger.Printf("コピー中: %s -> %s", filepath.Base(inputFile), filepath.Base(outputFile))

	// ファイルを開く
	sourceFile, err := os.Open(inputFile)
	if err != nil {
		return fmt.Errorf("入力ファイル '%s' オープンエラー: %w", inputFile, err)
	}
	defer sourceFile.Close()

	// 出力ファイルを作成
	destFile, err := os.OpenFile(outputFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return fmt.Errorf("出力ファイル '%s' 作成エラー: %w", outputFile, err)
	}
	defer destFile.Close()

	// データをコピー
	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		// コピー失敗したら出力ファイルを削除試行
		_ = os.Remove(outputFile)
		return fmt.Errorf("ファイルコピーエラー ('%s' -> '%s'): %w", inputFile, outputFile, err)
	}

	// Closeの前にエラーチェック (Copyのエラーはここで補足されることが多い)
	if err := destFile.Close(); err != nil {
		_ = os.Remove(outputFile) // Close失敗でもファイルを削除試行
		return fmt.Errorf("出力ファイル '%s' クローズエラー: %w", outputFile, err)
	}
	// sourceFile.Close() のエラーは通常無視しても問題ないことが多い

	return nil
}

// --- PART 5 END ---
// processVideoFile: 動画ファイルを処理するメイン関数
// quickModeFlag によって処理フローが分岐する
func processVideoFile(inputFile, outputFile, tempDir string) error {
	originalInputFile := inputFile // 元の入力パスを保持
	baseName := filepath.Base(inputFile)
	outBaseName := filepath.Base(outputFile)

	// --- 既存ファイル/マーカーチェック (変更なし) ---
	if outInfo, err := os.Stat(outputFile); err == nil {
		if outInfo.Size() > 0 {
			logger.Printf("スキップ (変換済み > 0 byte): %s", outBaseName)
			return nil
		} else {
			logger.Printf("警告: サイズ0の出力ファイル '%s' が存在。削除して続行。", outBaseName)
			if err := os.Remove(outputFile); err != nil {
				logger.Printf("警告: サイズ0ファイル削除失敗(%s): %v", outputFile, err)
			}
		}
	} else if !os.IsNotExist(err) {
		logger.Printf("警告: 出力ファイル確認エラー(%s): %v。処理試行。", outputFile, err)
	}
	for _, markerSuffix := range failedMarkersToDelete {
		markerFile := outputFile + markerSuffix
		if _, err := os.Stat(markerFile); err == nil {
			logger.Printf("スキップ (旧マーカー %s 存在): %s", markerSuffix, outBaseName)
			return nil
		} else if !os.IsNotExist(err) {
			logger.Printf("警告: マーカー確認エラー(%s): %v", markerFile, err)
		}
	}

	// --- 処理モード分岐 ---
	if quickModeFlag {
		// === Quick Mode (直接処理) ===
		renamedSourcePath := inputFile + "._processing" // リネーム後の一時的なソースパス
		logger.Printf("--- 動画処理開始 [Quick Mode]: %s ---", baseName)
		debugLogPrintf("ソースリネーム試行: %s -> %s", inputFile, renamedSourcePath)

		// 1. ソースファイルをリネーム
		if err := os.Rename(inputFile, renamedSourcePath); err != nil {
			logger.Printf("エラー [Quick Mode]: ソースリネーム失敗 (%s -> %s): %v", inputFile, renamedSourcePath, err)
			return fmt.Errorf("ソースリネーム失敗: %w", err)
		}
		renameBackNeeded := true
		defer func() {
			if renameBackNeeded {
				debugLogPrintf("処理終了またはエラー、ソースを元に戻します: %s -> %s", renamedSourcePath, originalInputFile)
				if err := os.Rename(renamedSourcePath, originalInputFile); err != nil {
					logger.Printf("警告 [Quick Mode]: ソースのリネームバック失敗 (%s -> %s): %v", renamedSourcePath, originalInputFile, err)
					logger.Printf("  手動で '%s' を '%s' に戻してください。", renamedSourcePath, originalInputFile)
				}
			}
		}()

		// 2. 出力先ディレクトリを確認・作成
		outputDir := filepath.Dir(outputFile)
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			logger.Printf("エラー [Quick Mode]: 出力先ディレクトリ作成失敗 (%s): %v", outputDir, err)
			return fmt.Errorf("出力先ディレクトリ作成失敗: %w", err) // deferがリネームバック試行
		}

		// 3. ffmpeg 実行 (HW/CPUフォールバックあり、出力先は最終パス)
		var ctx context.Context
		var cancel context.CancelFunc
		if timeoutSeconds > 0 {
			ctx, cancel = context.WithTimeout(context.Background(), time.Duration(timeoutSeconds)*time.Second)
		} else {
			ctx, cancel = context.WithCancel(context.Background())
		}
		defer cancel()
		var finalResult ffmpegResult
		logger.Printf("エンコード試行 1 (%s) [Quick Mode]...", hwEncoder)
		result1 := executeFFmpeg(ctx, renamedSourcePath, outputFile, "", hwEncoder, hwEncoderOptions) // tempDirは空文字
		finalResult = result1
		if result1.exitCode != 0 && ctx.Err() == nil && cpuEncoder != "" {
			logger.Printf("試行 1 (%s) 失敗(Exit:%d)。CPU (%s) で再試行 [Quick Mode]...", hwEncoder, result1.exitCode, cpuEncoder)
			_ = os.Remove(outputFile) // 失敗した(かもしれない)出力ファイルを削除試行
			var ctx2 context.Context
			var cancel2 context.CancelFunc
			if timeoutSeconds > 0 {
				ctx2, cancel2 = context.WithTimeout(context.Background(), time.Duration(timeoutSeconds)*time.Second)
			} else {
				ctx2, cancel2 = context.WithCancel(context.Background())
			}
			defer cancel2()
			result2 := executeFFmpeg(ctx2, renamedSourcePath, outputFile, "", cpuEncoder, cpuEncoderOptions)
			finalResult = result2
			if result2.exitCode == 0 {
				logger.Printf("CPU (%s) 再試行成功 [Quick Mode]。", cpuEncoder)
			} else {
				logger.Printf("CPU (%s) 再試行も失敗(Exit:%d, Timeout:%t) [Quick Mode]。", cpuEncoder, result2.exitCode, result2.timedOut)
			}
		} // (その他のログ省略)

		// 4. 結果処理
		if finalResult.exitCode == 0 {
			// --- Quick Mode 成功 ---
			renameBackNeeded = false // 正常終了したので defer でのリネームバックは不要
			logger.Printf("成功 [Quick Mode]: %s", filepath.Base(outputFile))

			// --- CORRECTED: リネームしたソースファイルを元の名前に戻す ---
			debugLogPrintf("エンコード成功、ソースを元に戻します: %s -> %s", renamedSourcePath, originalInputFile)
			if err := os.Rename(renamedSourcePath, originalInputFile); err != nil {
				// 正常終了のはずが、ソースを元に戻せない場合 (致命的ではないが要警告)
				logger.Printf("警告 [Quick Mode]: エンコード成功後、ソースの復元に失敗 (%s -> %s): %v", renamedSourcePath, originalInputFile, err)
				logger.Printf("  手動で '%s' を '%s' に戻す必要があるかもしれません。", renamedSourcePath, originalInputFile)
				// ここでエラーを返すか警告に留めるか。エンコード自体は成功しているので警告に留めることも考えられる。
				// return fmt.Errorf("エンコード成功、しかしソース復元失敗: %w", err) // エラーとして扱う場合
			}
			// --- END CORRECTION ---

			// REMOVED: 元のソースファイルを削除する処理を削除
			// debugLogPrintf("エンコード成功、リネーム後ソース削除: %s", renamedSourcePath)
			// if err := os.Remove(renamedSourcePath); err != nil { ... }

			return nil // 正常終了
		} else {
			// --- Quick Mode 失敗 ---
			renameBackNeeded = true // defer でリネームバックを試みるようにフラグを立てる
			return handleProcessingFailure(originalInputFile, outputFile, finalResult, true, renamedSourcePath, "")
		}

	} else {
		// --- Normal (Temp) Mode (一時ディレクトリ使用) ---
		// (このブロックは変更なし)
		logger.Printf("--- 動画処理開始 [Temp Mode]: %s ---", baseName)
		tempInputExt := filepath.Ext(inputFile)
		tempInputName := fmt.Sprintf("%s_input%s", strings.TrimSuffix(baseName, tempInputExt), tempInputExt)
		tempOutputName := fmt.Sprintf("%s_output%s", strings.TrimSuffix(baseName, tempInputExt), outputSuffix)
		tempInputPath := filepath.Join(tempDir, tempInputName)
		tempOutputPath := filepath.Join(tempDir, tempOutputName)
		debugLogPrintf("一時入力: %s", tempInputPath)
		debugLogPrintf("一時出力: %s", tempOutputPath)
		inputData, err := os.ReadFile(inputFile)
		if err != nil {
			return handleProcessingFailure(originalInputFile, outputFile, ffmpegResult{err: fmt.Errorf("一時モード: 入力読込失敗: %w", err), exitCode: -10}, false, "", "")
		}
		if err := os.WriteFile(tempInputPath, inputData, 0666); err != nil {
			return handleProcessingFailure(originalInputFile, outputFile, ffmpegResult{err: fmt.Errorf("一時モード: 一時入力作成失敗: %w", err), exitCode: -11}, false, "", "")
		}
		defer func() { debugLogPrintf("一時入力削除: %s", tempInputPath); _ = os.Remove(tempInputPath) }()
		var ctx context.Context
		var cancel context.CancelFunc
		if timeoutSeconds > 0 {
			ctx, cancel = context.WithTimeout(context.Background(), time.Duration(timeoutSeconds)*time.Second)
		} else {
			ctx, cancel = context.WithCancel(context.Background())
		}
		defer cancel()
		var finalResult ffmpegResult
		logger.Printf("エンコード試行 1 (%s) [Temp Mode]...", hwEncoder)
		result1 := executeFFmpeg(ctx, tempInputPath, tempOutputPath, tempDir, hwEncoder, hwEncoderOptions)
		finalResult = result1
		if result1.exitCode != 0 && ctx.Err() == nil && cpuEncoder != "" {
			logger.Printf("試行 1 (%s) 失敗(Exit:%d)。CPU (%s) で再試行 [Temp Mode]...", hwEncoder, result1.exitCode, cpuEncoder)
			_ = os.Remove(tempOutputPath)
			var ctx2 context.Context
			var cancel2 context.CancelFunc
			if timeoutSeconds > 0 {
				ctx2, cancel2 = context.WithTimeout(context.Background(), time.Duration(timeoutSeconds)*time.Second)
			} else {
				ctx2, cancel2 = context.WithCancel(context.Background())
			}
			defer cancel2()
			result2 := executeFFmpeg(ctx2, tempInputPath, tempOutputPath, tempDir, cpuEncoder, cpuEncoderOptions)
			finalResult = result2
			if result2.exitCode == 0 {
				logger.Printf("CPU (%s) 再試行成功 [Temp Mode]。", cpuEncoder)
			} else {
				logger.Printf("CPU (%s) 再試行も失敗(Exit:%d, Timeout:%t) [Temp Mode]。", cpuEncoder, result2.exitCode, result2.timedOut)
			}
		}
		if finalResult.exitCode == 0 {
			tempOutInfo, err := os.Stat(tempOutputPath)
			if err != nil {
				return handleProcessingFailure(originalInputFile, outputFile, ffmpegResult{err: fmt.Errorf("一時モード: 一時出力確認エラー: %w", err), exitCode: -12}, false, "", tempOutputPath)
			}
			if tempOutInfo.Size() == 0 {
				_ = os.Remove(tempOutputPath)
				return handleProcessingFailure(originalInputFile, outputFile, ffmpegResult{err: fmt.Errorf("一時モード: 一時出力サイズ0"), exitCode: -13}, false, "", tempOutputPath)
			}
			outputDir := filepath.Dir(outputFile)
			if err := os.MkdirAll(outputDir, 0755); err != nil {
				_ = os.Remove(tempOutputPath)
				return handleProcessingFailure(originalInputFile, outputFile, ffmpegResult{err: fmt.Errorf("一時モード: 最終Dir作成失敗: %w", err), exitCode: -14}, false, "", tempOutputPath)
			}
			debugLogPrintf("一時出力ファイルを移動試行: %s -> %s", tempOutputPath, outputFile)
			err = os.Rename(tempOutputPath, outputFile)
			if err != nil {
				logger.Printf("警告: os.Rename 失敗 (%v)。コピーフォールバック試行...", err)
				debugLogPrintf("フォールバックコピー開始: %s -> %s", tempOutputPath, outputFile)
				copyErr := copyFileManually(tempOutputPath, outputFile)
				if copyErr != nil {
					_ = os.Remove(tempOutputPath)
					return handleProcessingFailure(originalInputFile, outputFile, ffmpegResult{err: fmt.Errorf("一時モード: Rename失敗後のコピーも失敗: %w (Renameエラー: %v)", copyErr, err), exitCode: -15}, false, "", tempOutputPath)
				}
				debugLogPrintf("フォールバックコピー成功。元の一時ファイル削除: %s", tempOutputPath)
				if removeErr := os.Remove(tempOutputPath); removeErr != nil {
					logger.Printf("警告: コピー成功後の一時ファイル削除失敗 (%s): %v", tempOutputPath, removeErr)
				}
				logger.Printf("成功 (コピー) [Temp Mode]: %s", outBaseName)
			} else {
				debugLogPrintf("os.Rename 成功。")
				logger.Printf("成功 (Rename) [Temp Mode]: %s", outBaseName)
			}
			return nil
		} else {
			return handleProcessingFailure(originalInputFile, outputFile, finalResult, false, "", tempOutputPath)
		}
	}
	// 通常はここまで到達しないはず
	// return errors.New("internal error: processVideoFile reached end unexpectedly")
}

// --- 他の関数 (変更なし) ---
// copyFileManually, handleProcessingFailure, fileExists, createMarkerFile, removeRestartFiles など
// (handleProcessingFailure は processVideoFile 内から呼び出される)
// 手動でファイルをコピーするヘルパー関数
// (processVideoFile より前に配置してください)
func copyFileManually(src, dst string) error {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("コピー元statエラー (%s): %w", src, err)
	}
	if !sourceFileStat.Mode().IsRegular() {
		return fmt.Errorf("%s は通常ファイルではありません", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("コピー元オープンエラー (%s): %w", src, err)
	}
	defer source.Close()

	// 出力ファイルを作成 (パーミッションは元ファイルに合わせる)
	// TRUNC フラグで既存ファイルを空にする
	destination, err := os.OpenFile(dst, os.O_RDWR|os.O_CREATE|os.O_TRUNC, sourceFileStat.Mode())
	if err != nil {
		return fmt.Errorf("コピー先作成エラー (%s): %w", dst, err)
	}
	// defer で destination.Close() を呼ぶが、エラーチェックのため後で明示的にも呼ぶ
	defer destination.Close()

	// io.Copyでデータをコピー
	bytesCopied, err := io.Copy(destination, source)
	if err != nil {
		_ = os.Remove(dst) // コピー失敗時は作成したファイルを削除試行
		return fmt.Errorf("io.Copyエラー (%s -> %s): %w", src, dst, err)
	}
	debugLogPrintf("%d バイトコピー完了: %s -> %s", bytesCopied, src, dst)

	// 書き込みをディスクに確定させる (必須ではないが確実性を高める)
	err = destination.Sync()
	if err != nil {
		_ = os.Remove(dst)
		return fmt.Errorf("コピー先Syncエラー (%s): %w", dst, err)
	}

	// Closeのエラーもチェック (deferでも呼ばれるが念のため)
	err = destination.Close()
	if err != nil {
		_ = os.Remove(dst)
		return fmt.Errorf("コピー先クローズエラー (%s): %w", dst, err)
	}

	return nil
}

// マーカーファイルを作成する関数
func createMarkerFile(markerPath string, content string) {
	debugLogPrintf("マーカーファイル作成: %s (内容: %s)", markerPath, content)
	markerDir := filepath.Dir(markerPath)
	if err := os.MkdirAll(markerDir, 0755); err != nil {
		logger.Printf("警告: マーカー用ディレクトリ作成失敗 (%s): %v", markerDir, err)
	}
	fileContent := content
	maxLen := 200
	if len(fileContent) > maxLen {
		fileContent = fileContent[:maxLen] + "..."
	}
	if err := os.WriteFile(markerPath, []byte(fileContent), 0644); err != nil {
		logger.Printf("警告: マーカーファイル書き込み失敗 (%s): %v", markerPath, err)
	}
}

// -Restart オプション用のファイル削除関数
func removeRestartFiles(dir string) error {
	logger.Printf("-Restart: %s 内のエラーマーカーと0バイト動画ファイルを削除します...", dir)
	filesRemoved := 0
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			logger.Printf("警告: ディレクトリ '%s' 走査エラー: %v。スキップ。", path, err)
			if d != nil && d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		fileNameLower := strings.ToLower(d.Name())
		for _, markerSuffix := range failedMarkersToDelete {
			if strings.HasSuffix(fileNameLower, markerSuffix) {
				logger.Printf("-Restart: マーカー削除: %s", path)
				if err := os.Remove(path); err != nil {
					logger.Printf("警告: マーカー削除失敗 (%s): %v", path, err)
				} else {
					filesRemoved++
				}
				return nil
			}
		}
		ext := strings.ToLower(filepath.Ext(fileNameLower))
		if _, isVideo := videoExtensions[ext]; isVideo {
			info, infoErr := os.Stat(path)
			if infoErr != nil {
				logger.Printf("警告: ファイル情報取得エラー (%s): %v。スキップ。", path, infoErr)
				return nil
			}
			if info.Size() == 0 {
				logger.Printf("-Restart: 0バイト動画削除: %s", path)
				if err := os.Remove(path); err != nil {
					logger.Printf("警告: 0バイト動画削除失敗 (%s): %v", path, err)
				} else {
					filesRemoved++
				}
				return nil
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("-Restart 処理中に予期せぬエラー: %w", err)
	}
	logger.Printf("-Restart: %d 個削除。", filesRemoved)
	return nil
}

// --- PART 6 END ---
// --- ヘルパー関数 (Usage用) ---
func getVideoExtList() string {
	keys := make([]string, 0, len(videoExtensions))
	for k := range videoExtensions {
		keys = append(keys, k)
	}
	sort.Strings(keys) // アルファベット順にソート
	return strings.Join(keys, ", ")
}

func getImageExtList() string {
	keys := make([]string, 0, len(imageExtensions))
	for k := range imageExtensions {
		keys = append(keys, k)
	}
	sort.Strings(keys) // アルファベット順にソート
	return strings.Join(keys, ", ")
}

// デフォルトのffmpegディレクトリパスを返す（Usage表示用）
func getDefaultFfmpegDir() string {
	// 実際のデフォルト値設定は main 関数内で行う
	// ここでは Usage 表示のためだけに返す
	return "." // カレントディレクトリを示す
}

// --- printUsage 関数 ---
func printUsage() {
	// プログラム名を取得
	progName := filepath.Base(os.Args[0])

	// Usageメッセージを標準エラー出力に書き込む
	// ヒアドキュメント(`)を使うと改行やインデントが楽
	fmt.Fprintf(os.Stderr, `概要 (Synopsis):
  %s は、指定されたディレクトリ内の動画ファイルをAV1コーデックに変換し、
  画像ファイルをコピーするツールです。ffmpegを低優先度で実行し、
  ハードウェアエンコード失敗時にはCPUエンコードにフォールバックします。

使用法 (Usage):
  %s -s <入力元ディレクトリ> -o <出力先ディレクトリ> [オプション...]

説明 (Description):
  入力元ディレクトリを再帰的に検索し、動画と画像を処理します。
  - 動画ファイル (%s) は AV1 にエンコードされます。
    - まず -hwenc で指定されたHWエンコーダ (-hwopt オプション適用) を試行します。
    - 失敗時 (タイムアウト以外) は -cpuenc (-cpuopt オプション適用) で再試行します。
    - 音声は AAC に変換されます。
    - 出力ファイル名は元の名前に「%s」が付与されます。
  - 画像ファイル (%s) はそのまま出力先の対応するサブディレクトリにコピーされます。
  - エンコード処理は一時ディレクトリで行われます。
  - ffmpeg/ffprobe は指定されたディレクトリまたはPATHから検索されます。
  - ffmpeg プロセスは指定された優先度で実行されます (Windows: SetPriorityClass, Linux/macOS: nice)。

必須引数:
`, progName, progName, getVideoExtList(), outputSuffix, getImageExtList())

	// 各フラグの説明を出力
	// flag.FlagSet.PrintDefaults() は使わず、手動でフォーマット
	fmt.Fprintf(os.Stderr, "  -s <パス>\n\t変換またはコピーするファイルが含まれる入力元ディレクトリ。\n")
	fmt.Fprintf(os.Stderr, "  -o <パス>\n\t変換後またはコピー後のファイルを出力するディレクトリ。\n")

	fmt.Fprintln(os.Stderr, "\nオプション:")
	fmt.Fprintf(os.Stderr, "  -ffmpegdir <パス>\n\tffmpeg と ffprobe が格納されているディレクトリ。\n\t(デフォルト: \"%s\", または環境変数PATHから検索)\n", getDefaultFfmpegDir())
	fmt.Fprintf(os.Stderr, "  -priority <レベル>\n\tffmpeg プロセスの優先度。\n\t(idle, BelowNormal, Normal, AboveNormal)\n\t(デフォルト: \"BelowNormal\")\n")
	fmt.Fprintf(os.Stderr, "  -hwenc <名前>\n\t優先して試行するハードウェアエンコーダ名。\n\t(デフォルト: \"av1_nvenc\")\n")
	fmt.Fprintf(os.Stderr, "  -cpuenc <名前>\n\tフォールバック用CPUエンコーダ名 (空文字で無効)。\n\t(デフォルト: \"libsvtav1\")\n")
	// デフォルト値を変数から取得するように変更 (より正確に)
	fmt.Fprintf(os.Stderr, "  -hwopt \"<オプション>\"\n\tHWエンコーダ用の追加ffmpegオプション (引用符で囲む)。\n\t(デフォルト: \"%s\")\n", "-cq 25 -preset p5")   // flag定義のデフォルト値を反映
	fmt.Fprintf(os.Stderr, "  -cpuopt \"<オプション>\"\n\tCPUエンコーダ用の追加ffmpegオプション (引用符で囲む)。\n\t(デフォルト: \"%s\")\n", "-crf 28 -preset 7") // flag定義のデフォルト値を反映
	fmt.Fprintf(os.Stderr, "  -timeout <秒>\n\tffmpeg 各処理のタイムアウト秒数 (0で無効)。\n\t(デフォルト: %d)\n", 7200)                                 // flag定義のデフォルト値を反映
	fmt.Fprintf(os.Stderr, "  -usetemp\n\t多数の動画ファイルを処理する場合に一時ファイルリストを使用します。\n\t(デフォルト: false - メモリ内リストを使用)\n")
	fmt.Fprintf(os.Stderr, "  -log\n\tログを出力ディレクトリ内のファイル (GoTransAV1_Log_*.log) にも書き出します。\n\t(デフォルト: false)\n")
	fmt.Fprintf(os.Stderr, "  -debug\n\t詳細なデバッグログを有効にします。\n\t(デフォルト: false)\n")
	fmt.Fprintf(os.Stderr, "  -restart\n\t処理開始前に出力先のマーカーファイル (*.failedなど) と\n\tサイズ 0 の動画ファイルを削除します。\n\t(デフォルト: false)\n")
	fmt.Fprintf(os.Stderr, "  -force\n\t処理開始前に出力先ディレクトリを対話的に確認した後、\n\t完全に削除します。注意して使用してください。\n\t(デフォルト: false)\n")
	fmt.Fprintf(os.Stderr, "  -h, --help\n\tこのヘルプメッセージを表示します。\n") // -h, --help は flag パッケージが自動で処理

	fmt.Fprintln(os.Stderr, `
注意事項:
  - ffmpeg および ffprobe (任意) がシステムにインストールされている必要があります。
    -ffmpegdir でパスを指定するか、環境変数PATHに登録してください。
  - ハードウェアエンコード (-hwenc) を使用するには、対応するGPUとドライバが必要です。
  - -force オプションは出力先を完全に削除するため、実行前に確認メッセージが表示されます。
`)
}

// --- PART 7 END ---
// --- main 関数 ---
func main() {
	startTime = time.Now()
	flag.Usage = printUsage
	// --- フラグ定義 (変更なし) ---
	flag.StringVar(&sourceDir, "s", "", "入力元ディレクトリ (必須)") // Usageメッセージ変更
	flag.StringVar(&destDir, "o", "", "出力先ディレクトリ (必須)")   // Usageメッセージ変更
	defaultFfmpegDir := getDefaultFfmpegDir()
	flag.StringVar(&ffmpegDir, "ffmpegdir", defaultFfmpegDir, "ffmpeg/ffprobe 格納ディレクトリ")
	flag.StringVar(&ffmpegPriority, "priority", "Idle", "プロセス優先度 (idle|BelowNormal|Normal|AboveNormal)")
	flag.StringVar(&hwEncoder, "hwenc", "av1_nvenc", "優先HWエンコーダ名")
	flag.StringVar(&cpuEncoder, "cpuenc", "libsvtav1", "フォールバックCPUエンコーダ名")
	const defaultHwOpt = "-cq 25 -preset p5"
	const defaultCpuOpt = "-crf 28 -preset 7"
	const defaultTimeout = 7200
	flag.StringVar(&hwEncoderOptions, "hwopt", defaultHwOpt, "HWエンコーダ用ffmpegオプション")
	flag.StringVar(&cpuEncoderOptions, "cpuopt", defaultCpuOpt, "CPUエンコーダ用ffmpegオプション")
	flag.IntVar(&timeoutSeconds, "timeout", defaultTimeout, "タイムアウト秒数 (0で無効)")
	flag.BoolVar(&useTempFileListFlag, "usetemp", false, "一時ファイルリストを使用")
	flag.BoolVar(&logToFile, "log", false, "ログをファイルにも書き出す")
	flag.BoolVar(&debugMode, "debug", false, "詳細ログ出力")
	flag.BoolVar(&restart, "restart", false, "マーカー/0バイト動画削除")
	flag.BoolVar(&forceStart, "force", false, "出力Dirを強制削除 (確認あり)")
	flag.Parse()

	// --- 基本設定、パス検証、前処理 (変更なし) ---
	setupLogging()
	if sourceDir == "" || destDir == "" {
		logger.Println("エラー: -s と -o は必須。")
		flag.Usage()
		os.Exit(1)
	}
	var err error
	sourceDir, err = filepath.Abs(filepath.Clean(sourceDir))
	if err != nil {
		logger.Fatalf("エラー: 入力元パス正規化失敗: %v", err)
	}
	destDir, err = filepath.Abs(filepath.Clean(destDir))
	if err != nil {
		logger.Fatalf("エラー: 出力先パス正規化失敗: %v", err)
	}
	logger.Printf("ソース: %s", sourceDir)
	logger.Printf("出力: %s", destDir)
	if sourceDir == destDir {
		logger.Fatalf("エラー: 入力元と出力先が同じ。")
	}
	ffmpegBase := "ffmpeg"
	if runtime.GOOS == "windows" {
		ffmpegBase += ".exe"
	}
	ffmpegPath = filepath.Join(ffmpegDir, ffmpegBase)
	if _, err := exec.LookPath(ffmpegPath); err != nil {
		ffmpegPathFromPath, errPath := exec.LookPath(ffmpegBase)
		if errPath != nil {
			logger.Fatalf("エラー: ffmpegが見つかりません(%s or PATH)。", ffmpegPath)
		}
		logger.Printf("ffmpeg をPATHから使用: %s", ffmpegPathFromPath)
		ffmpegPath = ffmpegPathFromPath
	} else {
		ffmpegPath, _ = filepath.Abs(ffmpegPath)
		logger.Printf("ffmpeg を使用: %s", ffmpegPath)
	}
	ffprobeBase := "ffprobe"
	if runtime.GOOS == "windows" {
		ffprobeBase += ".exe"
	}
	ffprobePath = filepath.Join(ffmpegDir, ffprobeBase)
	if _, err := exec.LookPath(ffprobePath); err != nil {
		ffprobePathFromPath, errPath := exec.LookPath(ffprobeBase)
		if errPath != nil {
			logger.Printf("警告: ffprobeが見つかりません(%s or PATH)。", filepath.Join(ffmpegDir, ffprobeBase))
			ffprobePath = ""
		} else {
			logger.Printf("ffprobe をPATHから使用: %s", ffprobePathFromPath)
			ffprobePath = ffprobePathFromPath
		}
	} else {
		ffprobePath, _ = filepath.Abs(ffprobePath)
		logger.Printf("ffprobe を使用: %s", ffprobePath)
	}
	// --- 前処理 (-force / -restart) ---
	if forceStart {
		logger.Printf("!!! 警告: -force オプションが指定されました。出力ディレクトリ '%s' を完全に削除します。", destDir)
		fmt.Print("本当に実行しますか？ (yes/no): ") // 標準入力から確認を取得
		reader := bufio.NewReader(os.Stdin)
		// ReadStringのエラーもハンドリングする
		input, err := reader.ReadString('\n')
		if err != nil {
			// 標準入力からの読み取りエラーは致命的とする
			logger.Fatalf("エラー: 確認入力の読み取りに失敗しました: %v", err)
		}
		// 読み取った入力から改行コードなどを除去し、小文字に変換
		input = strings.TrimSpace(strings.ToLower(input))

		// 入力が "yes" の場合のみ削除を実行
		if input == "yes" {
			logger.Printf("出力ディレクトリ '%s' を削除しています...", destDir)
			// RemoveAll は存在しないディレクトリに対してはエラーを返さない
			if err := os.RemoveAll(destDir); err != nil {
				// ディレクトリの削除に失敗した場合は致命的エラーとして終了
				logger.Fatalf("エラー: 出力ディレクトリ削除失敗: %v", err)
			}
			logger.Println("出力ディレクトリを削除しました。")
		} else {
			// "yes" 以外が入力された場合はキャンセルとみなし、処理を中断
			logger.Println("削除をキャンセルしました。処理を中断します。")
			os.Exit(0) // 正常終了コード 0 で終了
		}
	} // <-- if forceStart の閉じ括弧

	// --- これ以降の処理 (出力ディレクトリ作成、restart処理など) が続く ---
	if err := os.MkdirAll(destDir, 0755); err != nil { /* ... */
	}
	if restart { /* ... */
	}
	// ...
	if err := os.MkdirAll(destDir, 0755); err != nil {
		logger.Fatalf("エラー: 出力Dir作成失敗 (%s): %v", destDir, err)
	}
	if restart {
		if err := removeRestartFiles(destDir); err != nil {
			logger.Fatalf("エラー: -restart 処理エラー: %v", err)
		}
	}
	tempDir, err := os.MkdirTemp("", tempDirPrefix)
	if err != nil {
		logger.Fatalf("エラー: 一時Dir作成失敗: %v", err)
	}
	logger.Printf("一時ディレクトリ: %s", tempDir)
	defer func() {
		debugLogPrintf("一時Dir削除: %s", tempDir)
		if err := os.RemoveAll(tempDir); err != nil {
			logger.Printf("警告: 一時Dir削除失敗(%s): %v", tempDir, err)
		}
	}()

	// --- ファイルリスト作成 (otherFiles を追加) ---
	logger.Println("--- ファイルリスト作成開始 ---")
	var videoFiles []string
	var imageFiles []string
	var otherFiles []string // ADDED: その他のファイル用リスト
	fileCount := 0
	err = filepath.WalkDir(sourceDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			logger.Printf("警告: Dir '%s' アクセスエラー: %v。スキップ。", path, err)
			if d != nil && d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		fileCount++
		ext := strings.ToLower(filepath.Ext(path))
		if _, isVideo := videoExtensions[ext]; isVideo {
			videoFiles = append(videoFiles, path)
		} else if _, isImage := imageExtensions[ext]; isImage {
			imageFiles = append(imageFiles, path)
		} else {
			// --- CHANGED: スキップせず otherFiles に追加 ---
			otherFiles = append(otherFiles, path)
			debugLogPrintf("その他のファイルとしてリストアップ: %s", path)
			// skippedCount は不要になったので削除
		}
		if fileCount%1000 == 0 && fileCount > 0 {
			logger.Printf("リスト作成中... %d件スキャン済み", fileCount)
		}
		return nil
	})
	if err != nil {
		logger.Fatalf("エラー: ファイルリスト作成中にエラー: %v", err)
	}
	// skippedCount を削除
	logger.Printf("リスト作成完了。 動画: %d件, 画像: %d件, その他: %d件 (総ファイル: %d件)", len(videoFiles), len(imageFiles), len(otherFiles), fileCount)

	// --- 一時ファイルリスト書き出し判定 (変更なし) ---
	if useTempFileListFlag {
		usingTempFileList = true
		tempFileName := fmt.Sprintf("GoTransAV1_FileList_%s.txt", startTime.Format("20060102_150405"))
		tempFileListPath = filepath.Join(tempDir, tempFileName)
		logger.Printf("-usetemp 指定のため、一時リストを使用: %s", tempFileListPath)
		file, err := os.Create(tempFileListPath)
		if err != nil {
			logger.Printf("エラー: 一時リスト作成失敗(%s): %v。メモリ処理続行。", tempFileListPath, err)
			usingTempFileList = false
		} else {
			writer := bufio.NewWriter(file)
			for _, vf := range videoFiles {
				if _, err := writer.WriteString(vf + "\n"); err != nil {
					file.Close()
					logger.Fatalf("エラー: 一時リスト書込失敗 (%s): %v", tempFileListPath, err)
				}
			}
			if err := writer.Flush(); err != nil {
				file.Close()
				logger.Fatalf("エラー: 一時リストフラッシュ失敗 (%s): %v", tempFileListPath, err)
			}
			if err := file.Close(); err != nil {
				logger.Fatalf("エラー: 一時リストクローズ失敗 (%s): %v", tempFileListPath, err)
			}
			videoFiles = nil
			logger.Printf("一時リスト書込完了。")
		}
	} else {
		usingTempFileList = false
		logger.Printf("メモリ上のリストを使用 (-usetemp 未指定)")
	}

	// --- 画像コピー処理 (変更なし) ---
	logger.Println("--- 画像コピー処理開始 ---")
	var imageCopyErrors []string
	imageCount := len(imageFiles)
	if imageCount > 0 {
		logger.Printf("%d 件の画像をコピーします...", imageCount)
		for _, imgFile := range imageFiles {
			relPath, err := filepath.Rel(sourceDir, imgFile)
			if err != nil {
				errMsg := fmt.Sprintf("画像相対パス計算失敗(%s): %v", imgFile, err)
				logger.Printf("エラー: %s", errMsg)
				imageCopyErrors = append(imageCopyErrors, errMsg)
				continue
			}
			imgOutputPath := filepath.Join(destDir, relPath)
			debugLogPrintf("画像コピー試行: %s -> %s", imgFile, imgOutputPath)
			if err := copyImageFile(imgFile, imgOutputPath); err != nil {
				errMsg := fmt.Sprintf("画像コピー失敗(%s): %v", filepath.Base(imgFile), err)
				logger.Printf("エラー: %s", errMsg)
				imageCopyErrors = append(imageCopyErrors, errMsg)
			}
		}
	} else {
		logger.Println("コピー対象の画像ファイルはありません。")
	}
	logger.Println("--- 画像コピー処理終了 ---")

	// --- その他のファイルコピー処理 --- ADDED
	logger.Println("--- その他のファイルコピー処理開始 ---")
	var otherCopyErrors []string
	otherCount := len(otherFiles)
	if otherCount > 0 {
		logger.Printf("%d 件のその他のファイルをコピーします...", otherCount)
		for _, otherFile := range otherFiles {
			// 画像コピーと同様のロジックで出力パスを計算
			relPath, err := filepath.Rel(sourceDir, otherFile)
			if err != nil {
				errMsg := fmt.Sprintf("その他ファイル相対パス計算失敗 (%s): %v", otherFile, err)
				logger.Printf("エラー: %s", errMsg)
				otherCopyErrors = append(otherCopyErrors, errMsg)
				continue
			}
			otherOutputPath := filepath.Join(destDir, relPath)
			debugLogPrintf("その他ファイルコピー試行: %s -> %s", otherFile, otherOutputPath)

			// copyImageFile 関数を再利用 (ファイルコピーロジックは同じため)
			if err := copyImageFile(otherFile, otherOutputPath); err != nil {
				errMsg := fmt.Sprintf("その他ファイルコピー失敗 (%s): %v", filepath.Base(otherFile), err)
				logger.Printf("エラー: %s", errMsg)
				otherCopyErrors = append(otherCopyErrors, errMsg)
			}
		}
	} else {
		logger.Println("コピー対象のその他のファイルはありません。")
	}
	logger.Println("--- その他のファイルコピー処理終了 ---")

	// --- 動画エンコード処理 (変更なし) ---
	logger.Println("--- 動画エンコード処理開始 ---")
	var videoProcessingErrors []string
	if usingTempFileList {
		logger.Printf("一時リスト %s から動画パス読込。", tempFileListPath)
		file, err := os.Open(tempFileListPath)
		if err != nil {
			logger.Fatalf("エラー: 一時リスト読込失敗 (%s): %v", tempFileListPath, err)
		}
		defer file.Close()
		scanner := bufio.NewScanner(file)
		videoIndex := 0
		for scanner.Scan() {
			videoIndex++
			filePath := strings.TrimSpace(scanner.Text())
			if filePath == "" {
				continue
			}
			logger.Printf("--- 動画エンコード (%d/不明): %s ---", videoIndex, filepath.Base(filePath))
			outputPath, pathErr := getOutputPath(filePath, sourceDir, destDir)
			if pathErr != nil {
				errMsg := fmt.Sprintf("動画出力パス計算失敗 (%s): %v", filePath, pathErr)
				logger.Printf("エラー: %s", errMsg)
				videoProcessingErrors = append(videoProcessingErrors, errMsg)
				continue
			}
			if err := processVideoFile(filePath, outputPath, tempDir); err != nil {
				videoProcessingErrors = append(videoProcessingErrors, fmt.Sprintf("%s: %v", filepath.Base(filePath), err))
			}
		}
		if err := scanner.Err(); err != nil {
			logger.Printf("エラー: 一時リストのスキャンエラー: %v", err)
		}
	} else {
		videoCount := len(videoFiles)
		if videoCount > 0 {
			logger.Printf("メモリ上のリストから %d 件の動画を処理。", videoCount)
			for i, vidFile := range videoFiles {
				logger.Printf("--- 動画エンコード (%d/%d): %s ---", i+1, videoCount, filepath.Base(vidFile))
				outputPath, pathErr := getOutputPath(vidFile, sourceDir, destDir)
				if pathErr != nil {
					errMsg := fmt.Sprintf("動画出力パス計算失敗 (%s): %v", vidFile, pathErr)
					logger.Printf("エラー: %s", errMsg)
					videoProcessingErrors = append(videoProcessingErrors, errMsg)
					continue
				}
				if err := processVideoFile(vidFile, outputPath, tempDir); err != nil {
					videoProcessingErrors = append(videoProcessingErrors, fmt.Sprintf("%s: %v", filepath.Base(vidFile), err))
				}
			}
		} else {
			logger.Println("エンコード対象の動画ファイルはありません。")
		}
	}
	logger.Println("--- 動画エンコード処理終了 ---")

	// --- 終了処理 (エラーリスト結合を修正) ---
	endTime := time.Now()
	elapsedTime := endTime.Sub(startTime)
	logger.Printf("総処理時間: %v", elapsedTime.Round(time.Second))
	// CHANGED: 3つのエラーリストを結合
	allErrors := append(append(imageCopyErrors, otherCopyErrors...), videoProcessingErrors...)
	if len(allErrors) > 0 {
		logger.Printf("--- 処理中に %d 件のエラーが発生しました ---", len(allErrors))
		limit := 20
		for i, e := range allErrors {
			if i >= limit {
				logger.Printf("  ...他 %d 件のエラー (ログを確認してください)", len(allErrors)-limit)
				break
			}
			logger.Printf("  [%d] %s", i+1, e)
		}
		os.Exit(1)
	} else {
		logger.Println("全ての処理が正常に完了しました。")
	}
}

// --- PART 8 END ---
