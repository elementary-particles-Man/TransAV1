package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"

	// "io" // 不要
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/windows" // Windows 特有の API 呼び出しに必要
)

// ffmpegResult: ffmpeg の実行結果を格納する構造体
type ffmpegResult struct {
	err      error // 発生したエラー
	timedOut bool  // タイムアウトしたかどうか
	exitCode int   // ffmpeg プロセスの終了コード (-1: 不明, -2: タイムアウト, -3: 実行時エラー)
}

// executeFFmpeg: ffmpeg プロセスを実行し、結果を返す
// ctx: タイムアウト制御のためのコンテキスト
// inputPath: 入力ファイルパス
// outputPath: 出力ファイルパス (QuickMode時は最終パス、TempMode時は一時パス)
// tempDir: 一時ディレクトリパス (ログなど用、現在は未使用)
// ffmpegPriority: プロセス優先度 (main.go で指定)
// encoder: 使用するエンコーダ名 (例: "av1_nvenc", "libsvtav1")
// encoderSpecificOptions: エンコーダ固有のオプション文字列 (例: "-cq 25 -preset p5")
func executeFFmpeg(ctx context.Context, inputPath string, outputPath string, tempDir string, ffmpegPriority string, encoder string, encoderSpecificOptions string) ffmpegResult {
	result := ffmpegResult{exitCode: -1} // 終了コードの初期値は不明(-1)

	// ffmpeg コマンドの基本パス (main.go で解決済み)
	baseCmd := ffmpegPath

	// ffmpeg に渡す引数リストを構築
	args := []string{
		"-hide_banner",  // バナー情報を非表示に
		"-nostats",      // 定期的な進捗状況の出力を抑制 (ログが見やすくなる)
		"-i", inputPath, // 入力ファイル指定
		"-c:v", encoder, // 映像エンコーダ指定
		"-c:a", "aac", // 音声エンコーダは AAC に固定 (必要ならオプション化)
		"-y", // 出力ファイルを常に上書き
		// ここにエンコーダ固有オプション、ログレベルが追加される
	}

	// エンコーダ固有オプションを追加 (スペースで分割して個別の引数にする)
	if encoderSpecificOptions != "" {
		// strings.Fields はスペース区切りの文字列をスライスに分割する
		opts := strings.Fields(encoderSpecificOptions)
		args = append(args, opts...)
		debugLogPrintf("エンコーダ (%s) 固有オプション追加: %v", encoder, opts)
	}

	// ログレベルを設定 (-debug フラグに応じて変更)
	if debugMode {
		// デバッグモード時は詳細なエラー情報が見たいので 'error' レベル
		args = append(args, "-loglevel", "error")
	} else {
		// 通常時は致命的なエラーのみ表示する 'fatal' レベルでログを最小限に
		args = append(args, "-loglevel", "fatal")
	}

	// 最後に出力ファイルパスを追加
	args = append(args, outputPath)

	// --- OS 別の優先度設定とコマンド構築 ---
	var finalArgs []string // 実際に exec に渡す引数リスト
	var cmd *exec.Cmd      // 実行するコマンドオブジェクト

	if runtime.GOOS == "windows" {
		// Windows: ffmpeg を直接実行 (優先度は後で設定)
		finalArgs = args
		cmd = exec.CommandContext(ctx, baseCmd, finalArgs...)
		debugLogPrintf("Windows コマンド準備: %s %v", baseCmd, finalArgs)
	} else if runtime.GOOS == "linux" || runtime.GOOS == "darwin" {
		// Unix (Linux/macOS): nice コマンド経由で優先度を設定
		niceArgs, err := getUnixNiceArgs(ffmpegPriority) // nice コマンド引数を取得
		if err != nil {
			// 優先度設定エラーは警告とし、nice なしで実行
			logger.Printf("警告: Unix 優先度 '%s' の設定に失敗: %v。デフォルト優先度で実行します。", ffmpegPriority, err)
			finalArgs = args
			cmd = exec.CommandContext(ctx, baseCmd, finalArgs...) // ffmpeg を直接実行
		} else {
			// nice コマンド経由で ffmpeg を実行
			// finalArgs = [ "nice", "-n", "10", "ffmpeg", "-i", ... ] のようになる
			finalArgs = append(niceArgs, baseCmd) // nice -n X ffmpeg
			finalArgs = append(finalArgs, args...)
			// exec.CommandContext の第一引数は実行ファイルパス、第二引数以降がその引数
			cmd = exec.CommandContext(ctx, finalArgs[0], finalArgs[1:]...) // finalArgs[0] は "nice"
			debugLogPrintf("Unix nice コマンド準備: %v", finalArgs)
		}
	} else {
		// その他の OS: 優先度設定はスキップ
		logger.Printf("警告: 未対応 OS (%s) のため、プロセス優先度設定はスキップされます。", runtime.GOOS)
		finalArgs = args
		cmd = exec.CommandContext(ctx, baseCmd, finalArgs...)
	}

	// --- プロセス属性設定 ---
	// SysProcAttr は OS 固有の設定を行うためのフィールド
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	setOSSpecificAttrs(cmd.SysProcAttr) // OS 固有の属性を設定 (例: Windows でウィンドウ非表示)

	// --- 標準エラー出力のパイプ設定 ---
	// ffmpeg は進捗やエラーを標準エラー出力 (stderr) に出すことが多い
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		result.err = fmt.Errorf("ffmpeg (%s) stderr パイプ作成エラー: %w", encoder, err)
		return result
	}
	// 標準出力 (stdout) も念のためキャプチャ (loglevel fatal ならほぼ出ないはず)
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		result.err = fmt.Errorf("ffmpeg (%s) stdout パイプ作成エラー: %w", encoder, err)
		return result
	}

	// --- ffmpeg プロセス実行開始 ---
	logger.Printf("ffmpeg 実行開始 (%s): %s", encoder, filepath.Base(outputPath))              // 通常ログはシンプルに
	debugLogPrintf("コマンド (%s): %s %s", encoder, cmd.Path, strings.Join(cmd.Args[1:], " ")) // デバッグ用にコマンド全体表示

	if err := cmd.Start(); err != nil {
		result.err = fmt.Errorf("ffmpeg (%s) プロセス開始エラー: %w", encoder, err)
		return result
	}

	// --- Windows プロセス優先度設定 (プロセス開始直後) ---
	if runtime.GOOS == "windows" {
		// プロセスが完全に初期化されるのを少し待つ (環境依存の可能性あり)
		time.Sleep(150 * time.Millisecond)
		// 優先度を設定
		if err := setWindowsPriorityAfterStart(cmd.Process, ffmpegPriority); err != nil {
			// 優先度設定失敗は警告ログのみ (setWindowsPriorityAfterStart 内でログ出力される)
		}
	}

	// --- 標準出力/エラー出力の非同期読み取り ---
	var ffmpegOutput strings.Builder // ffmpeg の出力を貯めるバッファ
	stderrScanner := bufio.NewScanner(stderrPipe)
	stdoutScanner := bufio.NewScanner(stdoutPipe)
	stderrChan := make(chan struct{}) // stderr 読み取り完了通知用チャネル
	stdoutChan := make(chan struct{}) // stdout 読み取り完了通知用チャネル

	// stderr 読み取りゴルーチン
	go func() {
		defer close(stderrChan) // ゴルーチン終了時にチャネルを閉じる
		for stderrScanner.Scan() {
			line := stderrScanner.Text()
			ffmpegOutput.WriteString(line + "\n") // バッファに追記
			// デバッグモード時、またはエラーっぽい行のみログに出力
			if debugMode || strings.Contains(strings.ToLower(line), "error") {
				logger.Printf("ffmpeg stderr (%s): %s", encoder, line)
			}
		}
		if err := stderrScanner.Err(); err != nil {
			// スキャン中にエラーが発生した場合
			logger.Printf("警告: ffmpeg (%s) stderr の読み取り中にエラー: %v", encoder, err)
		}
	}()

	// stdout 読み取りゴルーチン
	go func() {
		defer close(stdoutChan) // ゴルーチン終了時にチャネルを閉じる
		for stdoutScanner.Scan() {
			line := stdoutScanner.Text()
			ffmpegOutput.WriteString(line + "\n") // バッファに追記
			// 標準出力はデバッグモード時のみログに出力
			if debugMode {
				debugLogPrintf("ffmpeg stdout (%s): %s", encoder, line)
			}
		}
		if err := stdoutScanner.Err(); err != nil {
			// スキャン中にエラーが発生した場合
			logger.Printf("警告: ffmpeg (%s) stdout の読み取り中にエラー: %v", encoder, err)
		}
	}()

	// --- プロセス終了待機 ---
	// cmd.Wait() はプロセスが終了するまでブロックする
	err = cmd.Wait()

	// --- 出力読み取りゴルーチンの完全終了を待つ ---
	// パイプが閉じられ、ゴルーチン内のループが終了し、チャネルが閉じられるのを待つ
	<-stderrChan
	<-stdoutChan

	// --- 実行結果の判定 ---
	// 1. タイムアウト (コンテキストキャンセル) を確認
	if ctx.Err() == context.DeadlineExceeded {
		result.err = fmt.Errorf("ffmpeg (%s) タイムアウト (%d秒経過)", encoder, timeoutSeconds) // main.go の timeoutSeconds を使用
		result.timedOut = true
		result.exitCode = -2 // タイムアウトを示す内部コード
		// 念のためプロセスを Kill (既に終了している可能性もある)
		if cmd.Process != nil {
			_ = cmd.Process.Kill() // エラーは無視
		}
		logger.Printf("エラー: %v", result.err) // タイムアウトはエラーとしてログ出力
	} else if err != nil {
		// 2. Wait() がタイムアウト以外のエラーを返した場合
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			// プロセスは終了したが、ゼロ以外の終了コード
			result.exitCode = exitErr.ExitCode()
			// エラーメッセージには終了コードと ffmpeg の出力を付与
			errMsg := fmt.Sprintf("ffmpeg (%s) 失敗 (終了コード: %d)", encoder, result.exitCode)
			outputStr := strings.TrimSpace(ffmpegOutput.String())
			if outputStr != "" {
				errMsg += fmt.Sprintf("\n--- ffmpeg 出力 ---\n%s\n--- 出力終了 ---", outputStr)
			}
			result.err = errors.New(errMsg)  // エラー内容をセット
			logger.Printf("エラー: %s", errMsg) // エラーログを出力
		} else {
			// その他の実行時エラー (コマンドが見つからないなど)
			result.exitCode = -3 // 実行時エラーを示す内部コード
			result.err = fmt.Errorf("ffmpeg (%s) 実行時エラー: %w", encoder, err)
			logger.Printf("エラー: %v", result.err) // エラーログを出力
		}
	} else {
		// 3. 正常終了 (err == nil)
		result.exitCode = 0
		debugLogPrintf("ffmpeg (%s) 正常終了 (終了コード: 0)", encoder)
		// 正常終了時でも、デバッグモードなら出力をログに残す
		outputStr := strings.TrimSpace(ffmpegOutput.String())
		if debugMode && outputStr != "" {
			debugLogPrintf("ffmpeg (%s) 正常終了時の出力:\n--- ffmpeg 出力 ---\n%s\n--- 出力終了 ---", encoder, outputStr)
		}
	}

	return result
}

// setWindowsPriorityAfterStart: Windows でプロセス開始後に優先度を設定する
func setWindowsPriorityAfterStart(process *os.Process, priority string) error {
	if process == nil {
		return errors.New("プロセスが nil です")
	}

	// 文字列から Windows API の優先度定数に変換
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
		// 不明な優先度が指定された場合は警告ログを出して何もしない
		logger.Printf("警告: 無効な Windows 優先度指定 '%s'。デフォルト優先度を維持します。", priority)
		return nil // エラーとはしない
	}

	debugLogPrintf("Windows プロセス (PID: %d) の優先度を %s (0x%x) に設定試行...", process.Pid, priority, priorityClass)

	// プロセスハンドルを取得 (優先度設定に必要な権限を要求)
	handle, err := windows.OpenProcess(windows.PROCESS_SET_INFORMATION, false, uint32(process.Pid))
	if err != nil {
		errMsg := fmt.Sprintf("OpenProcess (PID: %d) 失敗: %v.", process.Pid, err)
		if errors.Is(err, windows.ERROR_ACCESS_DENIED) {
			errMsg += " プロセス優先度変更に必要な権限がない可能性があります (管理者権限で実行が必要な場合があります)。"
		}
		logger.Printf("警告: %s", errMsg) // 警告としてログ出力
		return nil                      // エラーは返さない
	}
	defer windows.CloseHandle(handle) // ハンドルを確実に閉じる

	// 優先度を設定
	err = windows.SetPriorityClass(handle, priorityClass)
	if err != nil {
		errMsg := fmt.Sprintf("SetPriorityClass (PID: %d, Priority: 0x%x) 失敗: %v.", process.Pid, priorityClass, err)
		logger.Printf("警告: %s", errMsg) // 警告としてログ出力
		return nil                      // エラーは返さない
	}

	debugLogPrintf("Windows プロセス (PID: %d) の優先度設定成功。", process.Pid)
	return nil
}

// getUnixNiceArgs: Unix 系 OS で nice コマンドの引数を生成する
func getUnixNiceArgs(priority string) ([]string, error) {
	var niceValue int
	switch strings.ToLower(priority) {
	case "idle": // 最低優先度
		niceValue = 19
	case "belownormal":
		niceValue = 10
	case "normal":
		niceValue = 0
	case "abovenormal": // より高い優先度 (低い nice 値)
		niceValue = -5 // 一般ユーザーで設定可能な範囲が多い (-20 まであるが root 権限が必要な場合あり)
	default:
		return nil, fmt.Errorf("無効な Unix 優先度指定: '%s' (idle, BelowNormal, Normal, AboveNormal のいずれか)", priority)
	}
	// nice コマンドとその引数を返す (例: ["nice", "-n", "10"])
	return []string{"nice", "-n", strconv.Itoa(niceValue)}, nil
}

// setOSSpecificAttrs: OS 固有のプロセス属性を設定する
func setOSSpecificAttrs(attr *syscall.SysProcAttr) {
	if runtime.GOOS == "windows" {
		// Windows で ffmpeg 実行時にコンソールウィンドウを表示しないようにする
		attr.HideWindow = true
	}
	// Linux/macOS では通常、追加の属性設定は不要
}

// processVideoFile: 1つの動画ファイルを処理するメインロジック
// inputFile: 入力動画ファイルのフルパス
// outputFile: 出力動画ファイルのフルパス
// tempDir: 一時ディレクトリのパス
// ffmpegPriority, hwEncoder, cpuEncoder, hwEncoderOptions, cpuEncoderOptions, timeoutSeconds, quickModeFlag: main から渡される設定値
func processVideoFile(inputFile string, outputFile string, tempDir string, ffmpegPriority string, hwEncoder string, cpuEncoder string, hwEncoderOptions string, cpuEncoderOptions string, timeoutSeconds int, quickModeFlag bool) error {
	logger.Printf("動画処理開始: %s", filepath.Base(inputFile))

	// --- 事前チェック ---
	// 出力ファイルが既に存在するかチェック (fileutils.go)
	if fileExists(outputFile) {
		logger.Printf("スキップ (出力ファイル既存): %s", filepath.Base(outputFile))
		return nil // 既に存在する場合は正常終了扱い
	}

	// 出力ディレクトリ作成 (fileutils.go)
	outputDir := filepath.Dir(outputFile)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		// 出力ディレクトリが作成できない場合は致命的エラー
		return fmt.Errorf("出力ディレクトリ '%s' の作成エラー: %w", outputDir, err)
	}

	// --- タイムアウト用コンテキスト設定 ---
	var ctx context.Context
	var cancel context.CancelFunc
	if timeoutSeconds > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), time.Duration(timeoutSeconds)*time.Second)
		debugLogPrintf("タイムアウト設定: %d 秒", timeoutSeconds)
	} else {
		// タイムアウト 0 または負数の場合はキャンセル可能なコンテキストのみ作成 (実質無制限)
		ctx, cancel = context.WithCancel(context.Background())
		debugLogPrintf("タイムアウト無効")
	}
	defer cancel() // 関数終了時に必ずキャンセルを呼び出し、リソースを解放

	// --- 入力ファイルの準備 (Quick/Temp モード分岐) ---
	var currentInputFile string  // ffmpeg に渡す実際の入力ファイルパス
	var tempOutputPath string    // Temp モード時の一時出力ファイルパス
	var renamedSourcePath string // Quick モード時のリネーム後ソースパス

	if quickModeFlag {
		// === Quick モード ===
		// ソースファイルをリネームして ffmpeg の入力とする
		renamedSourcePath = inputFile + ".processing" // リネーム後のパス
		logger.Printf("Quick Mode: ソースファイルを処理中名にリネーム: %s -> %s", filepath.Base(inputFile), filepath.Base(renamedSourcePath))
		if err := os.Rename(inputFile, renamedSourcePath); err != nil {
			// リネーム失敗は致命的エラー
			return fmt.Errorf("Quick Mode ソースファイルリネーム失敗 (%s -> %s): %w", inputFile, renamedSourcePath, err)
		}
		currentInputFile = renamedSourcePath // ffmpeg への入力はリネーム後のファイル
		tempOutputPath = outputFile          // Quick Mode では一時出力ファイルは使わず、直接最終出力パスに出力
		debugLogPrintf("Quick Mode: ffmpeg 入力: %s, ffmpeg 出力: %s", currentInputFile, tempOutputPath)
	} else {
		// === Temp モード (デフォルト) ===
		// ソースファイルを一時ディレクトリにコピーして ffmpeg の入力とする
		logger.Printf("Temp Mode: 一時ディレクトリにファイルをコピーします")
		// 一時ディレクトリ内の入力ファイルパス
		currentInputFile = filepath.Join(tempDir, filepath.Base(inputFile))
		// 一時ディレクトリ内の一時出力ファイルパス (ユニークな名前を付与)
		tempOutputFileName := fmt.Sprintf("%s_%d%s",
			strings.TrimSuffix(filepath.Base(inputFile), filepath.Ext(filepath.Base(inputFile))),
			time.Now().UnixNano(), // ナノ秒タイムスタンプで衝突回避
			outputSuffix)          // fileutils.go の outputSuffix
		tempOutputPath = filepath.Join(tempDir, tempOutputFileName)
		debugLogPrintf("Temp Mode: ffmpeg 入力: %s, ffmpeg 出力: %s", currentInputFile, tempOutputPath)

		// ファイルコピー実行 (fileutils.go)
		logger.Printf("一時コピー中: %s -> %s", filepath.Base(inputFile), filepath.Base(currentInputFile))
		if err := copyFileManually(inputFile, currentInputFile); err != nil {
			// コピー失敗時は作成された可能性のある一時ファイルを削除試行
			_ = os.Remove(currentInputFile)
			return fmt.Errorf("一時コピー失敗 (%s -> %s): %w", inputFile, currentInputFile, err)
		}
		debugLogPrintf("一時コピー完了: %s", currentInputFile)
	}

	// --- エンコード処理本体 ---
	var result ffmpegResult // ffmpeg の実行結果
	var usedEncoder string  // 実際に使用されたエンコーダ名 (ログ用)
	var usedOptions string  // 実際に使用されたオプション (ログ用)

	// 1. HW エンコーダ試行 (指定されている場合)
	if hwEncoder != "" {
		logger.Printf("HWエンコーダ (%s) で試行...", hwEncoder)
		usedEncoder = hwEncoder
		usedOptions = hwEncoderOptions
		result = executeFFmpeg(ctx, currentInputFile, tempOutputPath, tempDir, ffmpegPriority, usedEncoder, usedOptions)

		if result.err == nil && result.exitCode == 0 {
			// HW エンコード成功
			logger.Printf("HWエンコード成功 (%s)", hwEncoder)
			goto encodeSuccess // 成功時の後処理へ
		}

		// HW エンコード失敗
		logger.Printf("HWエンコード失敗 (%s, ExitCode: %d, TimedOut: %t): %v", hwEncoder, result.exitCode, result.timedOut, result.err)

		// タイムアウト以外の失敗で、かつ CPU エンコーダが指定されている場合 -> CPU で再試行
		if !result.timedOut && cpuEncoder != "" {
			logger.Printf("CPUエンコーダ (%s) で再試行...", cpuEncoder)
			// コンテキストはキャンセルされていないのでそのまま使用
			usedEncoder = cpuEncoder
			usedOptions = cpuEncoderOptions
			result = executeFFmpeg(ctx, currentInputFile, tempOutputPath, tempDir, ffmpegPriority, usedEncoder, usedOptions)

			if result.err == nil && result.exitCode == 0 {
				// CPU エンコード成功
				logger.Printf("CPUエンコード成功 (%s)", cpuEncoder)
				goto encodeSuccess // 成功時の後処理へ
			}

			// CPU エンコードも失敗
			logger.Printf("CPUエンコードも失敗 (%s, ExitCode: %d, TimedOut: %t): %v", cpuEncoder, result.exitCode, result.timedOut, result.err)
			goto encodeFailure // 失敗時の後処理へ
		} else if result.timedOut {
			// HW がタイムアウトした場合、CPU での再試行はスキップ
			logger.Printf("HWエンコードがタイムアウトしたため、CPUでの再試行はスキップします。")
			goto encodeFailure
		} else {
			// HW 失敗 & CPU 未指定の場合
			logger.Printf("CPUエンコーダが指定されていないため、再試行はスキップします。")
			goto encodeFailure
		}
	} else if cpuEncoder != "" {
		// 2. HW エンコーダ未指定 & CPU エンコーダ指定ありの場合
		logger.Printf("CPUエンコーダ (%s) で試行...", cpuEncoder)
		usedEncoder = cpuEncoder
		usedOptions = cpuEncoderOptions
		result = executeFFmpeg(ctx, currentInputFile, tempOutputPath, tempDir, ffmpegPriority, usedEncoder, usedOptions)

		if result.err == nil && result.exitCode == 0 {
			// CPU エンコード成功
			logger.Printf("CPUエンコード成功 (%s)", cpuEncoder)
			goto encodeSuccess // 成功時の後処理へ
		}

		// CPU エンコード失敗
		logger.Printf("CPUエンコード失敗 (%s, ExitCode: %d, TimedOut: %t): %v", cpuEncoder, result.exitCode, result.timedOut, result.err)
		goto encodeFailure // 失敗時の後処理へ

	} else {
		// 3. HW も CPU も未指定の場合 (通常 main でチェックされるはずだが念のため)
		return fmt.Errorf("エンコーダ (-hwenc または -cpuenc) が指定されていません。")
	}

encodeFailure: // --- エンコード失敗時の後処理 ---
	{ // goto ラベルと変数宣言スコープのためのブロック
		// マーカーファイルを作成 (fileutils.go)
		markerContent := fmt.Sprintf("Encoder: %s, Options: \"%s\", ExitCode: %d, TimedOut: %t, Error: %v", usedEncoder, usedOptions, result.exitCode, result.timedOut, result.err)
		markerSuffix := ".error" // デフォルト
		if result.timedOut {
			markerSuffix = ".timeout"
		} else if result.exitCode != 0 && result.exitCode != -1 && result.exitCode != -2 && result.exitCode != -3 {
			// ffmpeg が明確なエラーコードで終了した場合
			markerSuffix = fmt.Sprintf(".failed_%d", result.exitCode)
		} else if result.exitCode != 0 {
			// その他の失敗 (タイムアウト含む)
			markerSuffix = ".failed"
		}
		markerPath := outputFile + markerSuffix // 最終出力ファイルパスにサフィックスを追加
		createMarkerFile(markerPath, markerContent)

		// 失敗時のクリーンアップ処理 (fileutils.go)
		// QuickMode かどうかで処理内容が変わる
		return handleProcessingFailure(inputFile, outputFile, result, quickModeFlag, renamedSourcePath, tempOutputPath)
	}

encodeSuccess: // --- エンコード成功時の後処理 ---
	logger.Printf("エンコード成功: %s", filepath.Base(outputFile)) // 最終出力ファイル名でログ表示

	if !quickModeFlag {
		// === Temp モード成功時 ===
		// 一時出力ファイルを最終出力先に移動 (リネーム)
		debugLogPrintf("Temp Mode: 一時ファイルを最終出力先に移動: %s -> %s", tempOutputPath, outputFile)
		// 移動先にファイルが存在しないことを確認 (念のため)
		if _, err := os.Stat(outputFile); !os.IsNotExist(err) {
			// 存在する場合 (通常ありえないはずだが)、一時ファイルを削除して警告
			logger.Printf("警告: 移動先 '%s' にファイルが既に存在します。一時ファイル '%s' は削除されます。", outputFile, tempOutputPath)
			_ = os.Remove(tempOutputPath) // エラーは無視
			return nil                    // 成功として終了 (既存ファイルを上書きしない)
		}

		// リネーム実行
		if err := os.Rename(tempOutputPath, outputFile); err != nil {
			// リネーム失敗時のリカバリ試行
			_ = os.Remove(tempOutputPath) // 一時ファイルを削除
			_ = os.Remove(outputFile)     // 作成された可能性のある最終ファイルを削除
			// 失敗をエラーとして返す
			return fmt.Errorf("一時ファイル移動失敗 (%s -> %s): %w", tempOutputPath, outputFile, err)
		}
		logger.Printf("ファイル移動完了: %s", filepath.Base(outputFile))

		// Temp モードでは、一時ディレクトリにコピーした入力ファイル (currentInputFile) は
		// defer で一時ディレクトリごと削除される。元の入力ファイル (inputFile) はそのまま残る。

	} else {
		// === Quick モード成功時 ===
		// リネームしていたソースファイルを元の名前に戻す
		debugLogPrintf("Quick Mode: 処理中ファイル名を元に戻します: %s -> %s", renamedSourcePath, inputFile)
		if err := os.Rename(renamedSourcePath, inputFile); err != nil {
			// リネームバック失敗は警告ログに留める
			logger.Printf("警告 [Quick Mode]: ソースのリネームバック失敗 (%s -> %s): %v", renamedSourcePath, inputFile, err)
			logger.Printf("  手動で '%s' を '%s' に戻してください。", renamedSourcePath, inputFile)
			// 処理自体は成功しているのでエラーは返さない
		}
		// Quick モードでは ffmpeg が直接 outputFile に出力しているので、移動は不要。
	}

	// 正常終了
	logger.Printf("動画処理完了: %s", filepath.Base(outputFile))
	return nil
}
