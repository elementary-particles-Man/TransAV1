package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	// "io" // io パッケージは使用しないため削除
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
)

// ffmpeg 実行結果構造体
type ffmpegResult struct {
	err      error
	timedOut bool
	exitCode int
}

// executeFFmpeg: ffmpegプロセスを実行し、結果を返す
// encoderSpecificOptions: 各エンコーダ固有のオプション文字列 (-hwopt や -cpuopt の値)
// ffmpegPriority は main.go から引数として受け取る
// debugMode は logutils.go で定義されているが、同じパッケージなので直接参照可能
// ffmpegPath は main.go で定義されたグローバル変数
func executeFFmpeg(ctx context.Context, inputPath string, outputPath string, tempDir string, ffmpegPriority string, encoder string, encoderSpecificOptions string) ffmpegResult {
	result := ffmpegResult{exitCode: -1} // 初期値

	// main.go で設定されたグローバル変数 ffmpegPath を使用
	baseCmd := ffmpegPath

	// ffmpeg に渡す引数リストを作成
	args := []string{
		"-hide_banner", // バナー非表示
		"-stats",       // 進捗状況表示を有効にする (表示されるかは別問題)
		"-i", inputPath, // 入力ファイル
		"-c:v", encoder, // 映像エンコーダ (hwenc または cpuenc)
		"-c:a", "aac", // 音声エンコーダ
		"-y", // 出力ファイルを常に上書き
		// ここに追加オプションやログレベルが入る
	}

	// エンコーダ固有オプションを追加 (空文字列でなければ)
	if encoderSpecificOptions != "" {
		opts := strings.Fields(encoderSpecificOptions)
		args = append(args, opts...)
	}

	// logutils.go で設定された debugMode を使用
	if debugMode {
		args = append(args, "-loglevel", "error") // デバッグ時は error
	} else {
		args = append(args, "-loglevel", "fatal") // 通常時は fatal レベル
	}

	// 最後に出力ファイルパスを追加
	args = append(args, outputPath)

	// --- OS別 優先度/nice設定 ---
	var finalArgs []string
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		finalArgs = args
		cmd = exec.CommandContext(ctx, baseCmd, finalArgs...)
	} else if runtime.GOOS == "linux" || runtime.GOOS == "darwin" {
		// Unix (Linux/macOS) 用 nice 値設定
		niceArgs, err := getUnixNiceArgs(ffmpegPriority) // この関数は ffmpeg.go に残す
		if err != nil {
			// logutils.go の logger を使用
			logger.Printf("警告: Unix優先度設定エラー: %v。デフォルト優先度。", err)
			finalArgs = args
			cmd = exec.CommandContext(ctx, baseCmd, finalArgs...) // nice なしで実行
		} else {
			// nice コマンド経由で ffmpeg を実行
			finalArgs = append(niceArgs, baseCmd) // nice -n X ffmpeg ...
			finalArgs = append(finalArgs, args...)
			cmd = exec.CommandContext(ctx, finalArgs[0], finalArgs[1:]...) // finalArgs[0] は "nice"
			// logutils.go の debugLogPrintf を使用
			debugLogPrintf("Unix niceコマンドを使用して実行: %v", finalArgs)
		}
	} else {
		// logutils.go の logger を使用
		logger.Printf("警告: 未対応OS (%s) のため、優先度設定はスキップされます。", runtime.GOOS)
		finalArgs = args
		cmd = exec.CommandContext(ctx, baseCmd, finalArgs...)
	}

	// --- 実行設定 ---
	if cmd.SysProcAttr == nil { // Linux/macOS で nice を使う場合などに必要
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	// OS固有のSysProcAttr設定
	setOSSpecificAttrs(cmd.SysProcAttr) // この関数は ffmpeg.go に残す (OS固有のシステムコール関連のため)

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
	// logutils.go の logger を使用
	logger.Printf("ffmpeg 実行開始 (%s): %s -> %s", encoder, filepath.Base(inputPath), filepath.Base(outputPath))
	// logutils.go の debugLogPrintf を使用
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
		// Windows用プロセス優先度設定
		if err := setWindowsPriorityAfterStart(cmd.Process, ffmpegPriority); err != nil { // この関数は ffmpeg.go に残す
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
			// logutils.go の logger を使用
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
			// logutils.go の debugLogPrintf を使用
			if debugMode {
				debugLogPrintf("ffmpeg stdout (%s): %s", encoder, line)
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
		result.err = fmt.Errorf("ffmpeg (%s) タイムアウト (%d秒)", encoder, timeoutSeconds) // main.go の timeoutSeconds を使用
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
		// logutils.go の debugLogPrintf を使用
		debugLogPrintf("ffmpeg (%s) 正常終了 (ExitCode: 0)", encoder)
		// 正常終了時でも、デバッグモードなら出力をログに残す
		if debugMode && ffmpegOutput.Len() > 0 { // main.go の debugMode を使用
			debugLogPrintf("ffmpeg (%s) 正常終了時の出力:\n--- ffmpeg出力 ---\n%s\n--- 出力終了 ---", strings.TrimSpace(ffmpegOutput.String()))
		}
	}

	return result
}

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

	// logutils.go の debugLogPrintf を使用
	debugLogPrintf("Windowsプロセス (PID: %d) の優先度を %s (0x%x) に設定試行...", process.Pid, priority, priorityClass)

	handle, err := windows.OpenProcess(windows.PROCESS_SET_INFORMATION, false, uint32(process.Pid))
	if err != nil {
		// エラーハンドリングを改善: アクセス権限がない場合などの情報をログに出力
		errMsg := fmt.Sprintf("OpenProcess (PID: %d) 失敗: %v.", process.Pid, err)
		if errors.Is(err, windows.ERROR_ACCESS_DENIED) {
			errMsg += " プロセス優先度変更に必要な権限がない可能性があります。"
		}
		// このエラーは警告に留め、処理は続行させることも検討できる
		// return fmt.Errorf(errMsg)
		// logutils.go の logger を使用
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
		// logutils.go の logger を使用
		logger.Printf("警告: %s", errMsg) // 警告としてログ出力
		return nil                      // エラーは返さない
	}

	// logutils.go の debugLogPrintf を使用
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


// OS固有のSysProcAttr設定
func setOSSpecificAttrs(attr *syscall.SysProcAttr) {
	if runtime.GOOS == "windows" {
		// Windowsでコンソールウィンドウを表示しない
		attr.HideWindow = true
	}
	// Linux/macOS では通常不要
}


// processVideoFile: 動画ファイルをAV1に変換する関数
// main 関数から呼び出される
// ffmpegPriority, hwEncoder, cpuEncoder, hwEncoderOptions, cpuEncoderOptions, timeoutSeconds, quickModeFlag は main.go から引数として受け取る
// logger, debugLogPrintf は logutils.go から参照
// fileExists, copyFileManually, handleProcessingFailure, createMarkerFile, outputSuffix は fileutils.go から参照
func processVideoFile(inputFile string, outputFile string, tempDir string, ffmpegPriority string, hwEncoder string, cpuEncoder string, hwEncoderOptions string, cpuEncoderOptions string, timeoutSeconds int, quickModeFlag bool) error {
	// logutils.go の logger を使用
	logger.Printf("処理開始: %s", filepath.Base(inputFile))

	// goto で飛び越えないように、markerPath, markerContent, markerSuffix をここで宣言
	var markerPath string
	var markerContent string
	var markerSuffix string


	// 出力ファイルが既に存在するかチェック
	// fileutils.go の fileExists を使用
	if fileExists(outputFile) {
		// logutils.go の logger を使用
		logger.Printf("スキップ (既存): %s", filepath.Base(outputFile))
		return nil // 正常終了として扱う
	}

	// 出力ディレクトリ作成 (MkdirAll は存在してもエラーにならない)
	outputDir := filepath.Dir(outputFile)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("出力ディレクトリ '%s' の作成エラー: %w", outputDir, err)
	}

	// タイムアウト設定
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSeconds)*time.Second)
	if timeoutSeconds <= 0 { // タイムアウト無効の場合
		ctx, cancel = context.WithCancel(context.Background())
	}
	defer cancel() // 必ずキャンセルを呼び出す

	// 一時ファイルパスの決定
	tempOutputFileName := fmt.Sprintf("%s_%d%s",
		strings.TrimSuffix(filepath.Base(inputFile), filepath.Ext(filepath.Base(inputFile))),
		time.Now().UnixNano(), // ユニークな名前のためにナノ秒タイムスタンプを使用
		outputSuffix) // fileutils.go の outputSuffix を使用
	tempOutputPath := filepath.Join(tempDir, tempOutputFileName)

	var currentInputFile string
	var renamedSourcePath string // Quickモードでソースをリネームした場合のパス

	// Quickモードの場合、ソースファイルをリネームして直接エンコード
	if quickModeFlag { // main.go の quickModeFlag を使用
		// 元のファイル名に一時的なサフィックスを追加
		renamedSourcePath = inputFile + ".processing"
		// logutils.go の logger を使用
		logger.Printf("Quick Mode: ソースファイルをリネームして直接エンコード: %s -> %s", filepath.Base(inputFile), filepath.Base(renamedSourcePath))

		// リネーム実行
		if err := os.Rename(inputFile, renamedSourcePath); err != nil {
			// リネーム失敗は致命的
			return fmt.Errorf("Quick Mode ソースファイルリネーム失敗 (%s -> %s): %w", inputFile, renamedSourcePath, err)
		}
		currentInputFile = renamedSourcePath // ffmpegへの入力はリネーム後のファイル
		tempOutputPath = outputFile          // Quick Modeでは一時ファイルは使わず直接出力
		// logutils.go の debugLogPrintf を使用
		debugLogPrintf("Quick Mode: 一時出力パスを最終出力パスに設定: %s", tempOutputPath)

	} else {
		// Quickモードでない場合、一時ディレクトリにコピーしてエンコード
		// logutils.go の logger を使用
		logger.Printf("Temp Mode: 一時ディレクトリにコピーしてエンコード")
		currentInputFile = filepath.Join(tempDir, filepath.Base(inputFile)) // 一時ディレクトリ内のファイル名
		// logutils.go の debugLogPrintf を使用
		debugLogPrintf("Temp Mode: 一時入力ファイルパス: %s", currentInputFile)

		// 元ファイルを一時ディレクトリにコピー
		// logutils.go の logger を使用
		logger.Printf("一時コピー中: %s -> %s", filepath.Base(inputFile), filepath.Base(currentInputFile))
		// fileutils.go の copyFileManually を呼び出し
		if err := copyFileManually(inputFile, currentInputFile); err != nil {
			// コピー失敗時は一時ファイルを削除試行
			_ = os.Remove(currentInputFile)
			return fmt.Errorf("一時コピー失敗 (%s -> %s): %w", inputFile, currentInputFile, err)
		}
		// logutils.go の debugLogPrintf を使用
		debugLogPrintf("一時コピー完了: %s", currentInputFile)
	}

	// --- エンコード処理 ---
	var result ffmpegResult
	var usedEncoder string
	var usedOptions string

	// 1. HWエンコーダを試行
	if hwEncoder != "" { // main.go の hwEncoder を使用
		// logutils.go の logger を使用
		logger.Printf("HWエンコーダ (%s) で試行...", hwEncoder)
		usedEncoder = hwEncoder // main.go の hwEncoder を使用
		usedOptions = hwEncoderOptions // main.go の hwEncoderOptions を使用
		// executeFFmpeg を呼び出し
		result = executeFFmpeg(ctx, currentInputFile, tempOutputPath, tempDir, ffmpegPriority, usedEncoder, usedOptions)

		if result.err == nil && result.exitCode == 0 {
			// logutils.go の logger を使用
			logger.Printf("HWエンコード成功 (%s)", hwEncoder)
			goto encodeSuccess // 成功したら後処理へジャンプ
		}

		// HWエンコード失敗時の処理
		// logutils.go の logger を使用
		logger.Printf("HWエンコード失敗 (%s, ExitCode: %d, TimedOut: %t): %v", hwEncoder, result.exitCode, result.timedOut, result.err)

		// タイムアウト以外の失敗、かつCPUエンコーダが指定されている場合、CPUエンコーダを試行
		if !result.timedOut && cpuEncoder != "" { // main.go の cpuEncoder を使用
			// logutils.go の logger を使用
			logger.Printf("タイムアウトではない失敗のため、CPUエンコーダ (%s) で再試行...", cpuEncoder)
			// コンテキストはそのまま使用（タイムアウト設定は維持される）
			usedEncoder = cpuEncoder // main.go の cpuEncoder を使用
			usedOptions = cpuEncoderOptions // main.go の cpuEncoderOptions を使用
			// executeFFmpeg を呼び出し
			result = executeFFmpeg(ctx, currentInputFile, tempOutputPath, tempDir, ffmpegPriority, usedEncoder, usedOptions)

			if result.err == nil && result.exitCode == 0 {
				// logutils.go の logger を使用
				logger.Printf("CPUエンコード成功 (%s)", cpuEncoder)
				goto encodeSuccess // 成功したら後処理へジャンプ
			}

			// CPUエンコードも失敗
			// logutils.go の logger を使用
			logger.Printf("CPUエンコード失敗 (%s, ExitCode: %d, TimedOut: %t): %v", cpuEncoder, result.exitCode, result.timedOut, result.err)
			// 後処理に進む前に失敗を記録
			goto encodeFailure
		} else if result.timedOut {
			// HWエンコードがタイムアウトした場合、CPUエンコーダは試行しない（タイムアウトする可能性が高いため）
			// logutils.go の logger を使用
			logger.Printf("HWエンコードがタイムアウトしたため、CPUエンコーダでの再試行はスキップします。")
			goto encodeFailure
		} else {
			// HWエンコード失敗、かつCPUエンコーダが指定されていない場合
			// logutils.go の logger を使用
			logger.Printf("CPUエンコーダが指定されていないため、再試行はスキップします。")
			goto encodeFailure
		}
	} else if cpuEncoder != "" { // main.go の cpuEncoder を使用
		// HWエンコーダが指定されておらず、CPUエンコーダが指定されている場合
		// logutils.go の logger を使用
		logger.Printf("HWエンコーダが指定されていないため、CPUエンコーダ (%s) で試行...", cpuEncoder)
		usedEncoder = cpuEncoder // main.go の cpuEncoder を使用
		usedOptions = cpuEncoderOptions // main.go の cpuEncoderOptions を使用
		// executeFFmpeg を呼び出し
		result = executeFFmpeg(ctx, currentInputFile, tempOutputPath, tempDir, ffmpegPriority, usedEncoder, usedOptions)

		if result.err == nil && result.exitCode == 0 {
			// logutils.go の logger を使用
			logger.Printf("CPUエンコード成功 (%s)", cpuEncoder)
			goto encodeSuccess // 成功したら後処理へジャンプ
		}

		// CPUエンコード失敗
		// logutils.go の logger を使用
		logger.Printf("CPUエンコード失敗 (%s, ExitCode: %d, TimedOut: %t): %v", cpuEncoder, result.exitCode, result.timedOut, result.err)
		goto encodeFailure

	} else {
		// どちらのエンコーダも指定されていない場合
		return fmt.Errorf("エンコーダ (-hwenc または -cpuenc) が指定されていません。")
	}

encodeFailure:
	// エンコード失敗時の処理
	// マーカーファイルを作成
	markerContent = fmt.Sprintf("Encoder: %s, Options: \"%s\", ExitCode: %d, TimedOut: %t, Error: %v", usedEncoder, usedOptions, result.exitCode, result.timedOut, result.err)
	markerSuffix = ".error"
	if result.timedOut {
		markerSuffix = ".timeout"
	} else if result.exitCode != 0 {
		markerSuffix = ".failed"
	}
	// マーカーファイルパスは出力ファイルパスにサフィックスを追加
	markerPath = outputFile + markerSuffix // ここで markerPath に値を代入
	// fileutils.go の createMarkerFile を呼び出し
	createMarkerFile(markerPath, markerContent)

	// 失敗ハンドラを呼び出し、一時ファイルやリネームしたソースを処理
	// Quickモードの場合は renamedSourcePath を渡し、Tempモードの場合は tempOutputPath を渡す
	// fileutils.go の handleProcessingFailure を呼び出し
	return handleProcessingFailure(inputFile, outputFile, result, quickModeFlag, renamedSourcePath, tempOutputPath)


encodeSuccess:
	// エンコード成功時の処理
	// logutils.go の logger を使用
	logger.Printf("エンコード成功: %s", filepath.Base(tempOutputPath))

	// Quickモードでない場合、一時ファイルを最終出力先に移動
	if !quickModeFlag { // main.go の quickModeFlag を使用
		// logutils.go の debugLogPrintf を使用
		debugLogPrintf("Temp Mode: 一時ファイルを最終出力先に移動: %s -> %s", tempOutputPath, outputFile)
		// 最終出力先にファイルが存在しないことを再確認（念のため）
		// fileutils.go の fileExists を使用
		if fileExists(outputFile) {
			// logutils.go の logger を使用
			logger.Printf("警告: 移動先 '%s' にファイルが既に存在します。一時ファイルは削除されます。", outputFile)
			// 一時ファイルを削除して終了
			if err := os.Remove(tempOutputPath); err != nil {
				// logutils.go の logger を使用
				logger.Printf("警告: 成功後の一時ファイル削除失敗 (%s): %v", tempOutputPath, err)
			}
			// Quickモードでない場合の元のファイルはそのまま残る
			return nil // 成功として終了
		} else if _, err := os.Stat(outputFile); !os.IsNotExist(err) {
			// Statで "存在しない" 以外のエラーが発生した場合
			// 一時ファイルを削除してエラーを返す
			if err := os.Remove(tempOutputPath); err != nil {
				// logutils.go の logger を使用
				logger.Printf("警告: 成功後の一時ファイル削除失敗 (%s): %v", tempOutputPath, err)
			}
			return fmt.Errorf("最終出力先 '%s' の確認エラー: %w", outputFile, err)
		}

		// 移動実行
		if err := os.Rename(tempOutputPath, outputFile); err != nil {
			// 移動失敗時は、一時ファイルを削除試行
			_ = os.Remove(tempOutputPath)
			// 最終出力先も削除試行（部分的に作成された可能性）
			_ = os.Remove(outputFile)
			return fmt.Errorf("一時ファイル移動失敗 (%s -> %s): %w", tempOutputPath, outputFile, err)
		}
		// logutils.go の logger を使用
		logger.Printf("ファイル移動完了: %s", filepath.Base(outputFile))

		// 元の入力ファイルはそのまま残る（Temp Mode）

	} else {
		// Quickモードの場合、リネームしたソースファイルを元の名前に戻す
		// logutils.go の debugLogPrintf を使用
		debugLogPrintf("Quick Mode: リネームしたソースを元に戻します: %s -> %s", renamedSourcePath, inputFile)
		if err := os.Rename(renamedSourcePath, inputFile); err != nil {
			// リネームバック失敗は警告に留める
			// logutils.go の logger を使用
			logger.Printf("警告 [Quick Mode]: ソースのリネームバック失敗 (%s -> %s): %v", renamedSourcePath, inputFile, err)
			// logutils.go の logger を使用
			logger.Printf("  手動で '%s' を '%s' に戻してください。", renamedSourcePath, inputFile)
		}
		// Quickモードでは tempOutputPath は outputFile なので、移動は不要（既にffmpegが出力している）
		// 成功時は一時ファイル (tempOutputPath) は存在しない
	}

	// logutils.go の logger を使用
	logger.Printf("処理完了: %s", filepath.Base(outputFile))
	return nil // 正常終了
}
