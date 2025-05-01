package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/fs"

	// "log" // log パッケージは logutils.go でのみ使用
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// --- パッケージレベル定数 (デフォルト値) ---
const (
	defaultFfmpegDir = "." // カレントディレクトリ
	defaultPriority  = "BelowNormal"
	defaultHwEnc     = "av1_nvenc"
	defaultCpuEnc    = "libsvtav1"
	defaultHwOpt     = "-cq 25 -preset p5"
	defaultCpuOpt    = "-crf 28 -preset 7"
	defaultTimeout   = 7200
	tempDirPrefix    = "go_transav1_" // 一時ディレクトリ名の接頭辞
)

// --- グローバル変数 ---
var (
	sourceDir string // 入力元ディレクトリパス
	destDir   string // 出力先ディレクトリパス
	ffmpegDir string // ffmpeg/ffprobe 格納ディレクトリパス

	// ffmpeg 実行関連 (ffmpeg.go で主に使用)
	ffmpegPriority    string // ffmpeg プロセスの優先度
	hwEncoder         string // ハードウェアエンコーダ名
	cpuEncoder        string // CPUエンコーダ名
	hwEncoderOptions  string // HWエンコーダ用オプション
	cpuEncoderOptions string // CPUエンコーダ用オプション
	timeoutSeconds    int    // ffmpeg 処理のタイムアウト秒数

	// 動作モード関連フラグ
	logToFile         bool   // ログをファイルにも書き出すか
	debugMode         bool   // デバッグログを有効にするか (logutils.goからも参照される)
	restart           bool   // 再開モード (マーカー/0バイトファイル削除)
	forceStart        bool   // 出力ディレクトリ強制削除モード
	quickModeFlag     bool   // 一時コピーなしの高速モード
	usingTempFileList bool   // 一時ファイルリストを使用するか
	tempFileListPath  string // 一時ファイルリストのパス

	// 実行パスと時間
	ffmpegPath  string    // 検出された ffmpeg のフルパス
	ffprobePath string    // 検出された ffprobe のフルパス (任意)
	startTime   time.Time // プログラム開始時刻

	// logger, debugLogger は logutils.go で定義・初期化
)

// --- printUsage 関数: ヘルプメッセージを表示 ---
func printUsage() {
	progName := filepath.Base(os.Args[0]) // プログラム名を取得

	// 標準エラー出力に Usage メッセージを書き出す
	fmt.Fprintf(os.Stderr, `概要 (Synopsis):
  %s は、指定されたディレクトリ内の動画ファイルをAV1コーデックに変換し、
  その他のファイル (画像など) をコピーするツールです。ffmpegを低優先度で実行し、
  ハードウェアエンコード失敗時にはCPUエンコードにフォールバックします。

使用法 (Usage):
  %s -s <入力元ディレクトリ|入力ファイル> -o <出力先ディレクトリ> [オプション...]
  引数なしで起動した場合、詳細な使用法を表示するには -h オプションを使用してください。

説明 (Description):
  入力元がディレクトリの場合、再帰的に検索し、動画とその他のファイルを処理します。
  入力元がファイルの場合、そのファイルのみを動画として処理します (出力先はディレクトリ指定必須)。

  - 動画ファイル (%s) は AV1 にエンコードされます。
    - まず -hwenc で指定されたHWエンコーダ (-hwopt オプション適用) を試行します。
    - 失敗時 (タイムアウト以外) は -cpuenc (-cpuopt オプション適用) で再試行します。
    - 音声は AAC に変換されます。
    - 出力ファイル名は元の名前に「%s」が付与されます (例: input.mp4 -> input_AV1.mp4)。
  - その他のファイル (画像 %s など) はそのまま出力先の対応するサブディレクトリにコピーされます。
  - 通常、エンコード処理は一時ディレクトリで行われます (-quick 指定時を除く)。
  - ffmpeg/ffprobe は -ffmpegdir で指定されたディレクトリ、または環境変数PATHから検索されます。
  - ffmpeg プロセスは指定された優先度で実行されます (Windows: SetPriorityClass, Linux/macOS: nice)。

必須引数:
`, progName, progName, getVideoExtList(), outputSuffix, getImageExtList()) // fileutils.go の関数を呼び出し

	// 各フラグの説明を出力
	fmt.Fprintf(os.Stderr, "  -s <パス>\n\t入力元ディレクトリ、または単一の入力動画ファイルパス。\n")
	fmt.Fprintf(os.Stderr, "  -o <パス>\n\t出力先ディレクトリ。\n")

	fmt.Fprintln(os.Stderr, "\nオプション:")
	fmt.Fprintf(os.Stderr, "  -ffmpegdir <パス>\n\tffmpeg と ffprobe が格納されているディレクトリ。\n\t(デフォルト: カレントディレクトリ「%s」または環境変数PATHから検索)\n", defaultFfmpegDir)
	fmt.Fprintf(os.Stderr, "  -priority <レベル>\n\tffmpeg プロセスの優先度。\n\t(idle, BelowNormal, Normal, AboveNormal)\n\t(デフォルト: \"%s\")\n", defaultPriority)
	fmt.Fprintf(os.Stderr, "  -hwenc <名前>\n\t優先して試行するハードウェアエンコーダ名 (例: av1_nvenc, hevc_videotoolbox)。\n\t(デフォルト: \"%s\")\n", defaultHwEnc)
	fmt.Fprintf(os.Stderr, "  -cpuenc <名前>\n\tフォールバック用CPUエンコーダ名 (例: libsvtav1, libx265)。空文字で無効。\n\t(デフォルト: \"%s\")\n", defaultCpuEnc)
	fmt.Fprintf(os.Stderr, "  -hwopt \"<オプション>\"\n\tHWエンコーダ用の追加ffmpegオプション (引用符で囲む)。\n\t(デフォルト: \"%s\")\n", defaultHwOpt)    // パッケージレベル定数を使用
	fmt.Fprintf(os.Stderr, "  -cpuopt \"<オプション>\"\n\tCPUエンコーダ用の追加ffmpegオプション (引用符で囲む)。\n\t(デフォルト: \"%s\")\n", defaultCpuOpt) // パッケージレベル定数を使用
	fmt.Fprintf(os.Stderr, "  -timeout <秒>\n\tffmpeg 各処理のタイムアウト秒数 (0で無効)。\n\t(デフォルト: %d)\n", defaultTimeout)                 // パッケージレベル定数を使用
	fmt.Fprintf(os.Stderr, "  -quick\n\t高速モード: 一時コピーを行わず入力元ファイルを直接エンコード (非推奨)。\n\t処理失敗時に元ファイルが破損するリスクがあります。\n\t(デフォルト: false)\n")
	fmt.Fprintf(os.Stderr, "  -usetemp\n\t多数の動画ファイルを処理する場合に一時ファイルリストを使用します。\n\tメモリ使用量を抑えられますが、ディスクI/Oが増加します。\n\t(デフォルト: false - メモリ内リストを使用)\n")
	fmt.Fprintf(os.Stderr, "  -log\n\tログを出力ディレクトリ内のファイル (GoTransAV1_Log_*.log) にも書き出します。\n\t(デフォルト: false)\n")
	fmt.Fprintf(os.Stderr, "  -debug\n\t詳細なデバッグログ (ffmpegの出力など) を有効にします。\n\t(デフォルト: false)\n")
	fmt.Fprintf(os.Stderr, "  -restart\n\t処理開始前に出力先のマーカーファイル (*.failed, *.timeout など) と\n\tサイズ 0 の動画ファイルを削除します。中断からの再開時に便利です。\n\t(デフォルト: false)\n")
	fmt.Fprintf(os.Stderr, "  -force\n\t処理開始前に出力先ディレクトリを対話的に確認した後、\n\t完全に削除します。注意して使用してください。\n\t(デフォルト: false)\n")
	fmt.Fprintf(os.Stderr, "  -h, --help\n\tこのヘルプメッセージを表示します。\n")

	fmt.Fprintln(os.Stderr, `
注意事項:
  - ffmpeg および ffprobe (任意) がシステムにインストールされている必要があります。
    -ffmpegdir でパスを指定するか、環境変数PATHに登録してください。
  - ハードウェアエンコード (-hwenc) を使用するには、対応するGPUとドライバが必要です。
    エンコーダ名は ffmpeg -encoders で確認できます。
  - -force オプションは出力先を完全に削除するため、実行前に確認メッセージが表示されます。
  - -quick モードは処理中に問題が発生した場合、入力ファイルが不完全な状態になる可能性があります。
`)
}

// --- main 関数 ---
func main() {
	startTime = time.Now()  // プログラム開始時刻を記録
	flag.Usage = printUsage // Usage表示関数を設定

	// --- 引数がない場合の処理 ---
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "エラー: 引数が指定されていません。使用法を表示するには -h オプションを使用してください。")
		os.Exit(1)
	}

	// --- コマンドライン引数の定義 ---
	// flag 変数定義 (グローバル変数へのポインタを渡す)
	flag.StringVar(&sourceDir, "s", "", "入力元ディレクトリまたはファイル (必須)")
	flag.StringVar(&destDir, "o", "", "出力先ディレクトリ (必須)")
	flag.StringVar(&ffmpegDir, "ffmpegdir", defaultFfmpegDir, "ffmpeg/ffprobe 格納ディレクトリ")
	flag.StringVar(&ffmpegPriority, "priority", defaultPriority, "プロセス優先度 (idle|BelowNormal|Normal|AboveNormal)")
	flag.StringVar(&hwEncoder, "hwenc", defaultHwEnc, "優先HWエンコーダ名")
	flag.StringVar(&cpuEncoder, "cpuenc", defaultCpuEnc, "フォールバックCPUエンコーダ名")
	flag.StringVar(&hwEncoderOptions, "hwopt", defaultHwOpt, "HWエンコーダ用ffmpegオプション")
	flag.StringVar(&cpuEncoderOptions, "cpuopt", defaultCpuOpt, "CPUエンコーダ用追加ffmpegオプション")
	flag.IntVar(&timeoutSeconds, "timeout", defaultTimeout, "タイムアウト秒数 (0で無効)")
	flag.BoolVar(&quickModeFlag, "quick", false, "高速モード: 一時コピーを行わず直接エンコード")
	flag.BoolVar(&logToFile, "log", false, "ログをファイルにも書き出す")
	flag.BoolVar(&debugMode, "debug", false, "詳細ログ出力") // グローバル変数 debugMode に直接設定
	flag.BoolVar(&restart, "restart", false, "マーカー/0バイト動画削除")
	flag.BoolVar(&forceStart, "force", false, "出力Dirを強制削除 (確認あり)")
	flag.BoolVar(&usingTempFileList, "usetemp", false, "一時ファイルリストを使用")

	// --- 引数のパース ---
	flag.Parse()

	// --- ロギング設定 ---
	// logutils.go の setupLogging を呼び出し
	// setupLogging はグローバル変数 debugMode を参照してデバッグロガーを設定する
	setupLogging(destDir, startTime, logToFile, debugMode) // debugMode を引数で渡す必要はなくなった

	// --- 必須引数のチェック ---
	if sourceDir == "" || destDir == "" {
		logger.Println("エラー: -s (入力元) と -o (出力先) は必須です。")
		flag.Usage() // ヘルプを表示
		os.Exit(1)
	}

	// --- パスの正規化と検証 ---
	var err error
	sourceDir, err = filepath.Abs(filepath.Clean(sourceDir))
	if err != nil {
		logger.Fatalf("エラー: 入力元パス '%s' の正規化に失敗: %v", flag.Lookup("s").Value, err) // 元の入力値も表示
	}
	destDir, err = filepath.Abs(filepath.Clean(destDir))
	if err != nil {
		logger.Fatalf("エラー: 出力先パス '%s' の正規化に失敗: %v", flag.Lookup("o").Value, err) // 元の入力値も表示
	}
	logger.Printf("入力元 (正規化後): %s", sourceDir)
	logger.Printf("出力先 (正規化後): %s", destDir)
	if sourceDir == destDir {
		logger.Fatalf("エラー: 入力元と出力先が同じディレクトリです。")
	}

	// --- ffmpeg/ffprobe パスの検索と設定 ---
	ffmpegBase := "ffmpeg"
	ffprobeBase := "ffprobe"
	if runtime.GOOS == "windows" {
		ffmpegBase += ".exe"
		ffprobeBase += ".exe"
	}

	// ffmpeg のパス解決
	ffmpegPath = filepath.Join(ffmpegDir, ffmpegBase) // 指定ディレクトリを優先
	if _, err := exec.LookPath(ffmpegPath); err != nil {
		ffmpegPathFromPath, errPath := exec.LookPath(ffmpegBase)
		if errPath != nil {
			logger.Fatalf("エラー: ffmpeg が見つかりません。-ffmpegdir で指定されたパス '%s' にもなく、環境変数PATHにもありません。", ffmpegDir)
		}
		logger.Printf("情報: ffmpeg を環境変数PATHから使用します: %s", ffmpegPathFromPath)
		ffmpegPath = ffmpegPathFromPath
	} else {
		ffmpegPath, _ = filepath.Abs(ffmpegPath)
		logger.Printf("情報: ffmpeg を指定ディレクトリから使用します: %s", ffmpegPath)
	}

	// ffprobe のパス解決 (任意)
	ffprobePath = filepath.Join(ffmpegDir, ffprobeBase) // 指定ディレクトリを優先
	if _, err := exec.LookPath(ffprobePath); err != nil {
		ffprobePathFromPath, errPath := exec.LookPath(ffprobeBase)
		if errPath != nil {
			logger.Printf("警告: ffprobe が見つかりません (-ffmpegdir '%s' または PATH)。一部機能が制限される可能性があります。", ffmpegDir)
			ffprobePath = ""
		} else {
			logger.Printf("情報: ffprobe を環境変数PATHから使用します: %s", ffprobePathFromPath)
			ffprobePath = ffprobePathFromPath
		}
	} else {
		ffprobePath, _ = filepath.Abs(ffprobePath)
		logger.Printf("情報: ffprobe を指定ディレクトリから使用します: %s", ffprobePath)
	}

	// --- 入力元がファイルかディレクトリか判定 ---
	sourceInfo, err := os.Stat(sourceDir)
	if err != nil {
		logger.Fatalf("エラー: 入力元 '%s' の情報取得に失敗: %v", sourceDir, err)
	}
	isSingleFileMode := !sourceInfo.IsDir()

	// --- 出力先の検証と準備 ---
	destInfo, err := os.Stat(destDir)
	if err != nil {
		if os.IsNotExist(err) {
			if isSingleFileMode {
				logger.Fatalf("エラー: 単一ファイルモード(-s でファイルを指定)では、出力先ディレクトリ '%s' が事前に存在する必要があります。", destDir)
			} else {
				logger.Printf("情報: 出力先ディレクトリ '%s' が存在しないため、作成します。", destDir)
			}
		} else {
			logger.Fatalf("エラー: 出力先 '%s' の情報取得に失敗: %v", destDir, err)
		}
	} else if !destInfo.IsDir() {
		logger.Fatalf("エラー: 指定された出力先 '%s' はディレクトリではありません。", destDir)
	}

	// --- -force オプション処理 ---
	if forceStart {
		if isSingleFileMode {
			logger.Println("警告: 単一ファイルモードでは -force オプションは無視されます。")
		} else {
			logger.Printf("!!! 警告: -force オプションが指定されました。出力ディレクトリ '%s' を完全に削除します。", destDir)
			fmt.Print("本当に実行しますか？ (yes/no): ")
			reader := bufio.NewReader(os.Stdin)
			input, err := reader.ReadString('\n')
			if err != nil {
				logger.Fatalf("エラー: 確認入力の読み取りに失敗しました: %v", err)
			}
			input = strings.TrimSpace(strings.ToLower(input))

			if input == "yes" {
				logger.Printf("出力ディレクトリ '%s' を削除しています...", destDir)
				if err := os.RemoveAll(destDir); err != nil {
					logger.Fatalf("エラー: 出力ディレクトリ削除失敗: %v", err)
				}
				logger.Println("出力ディレクトリを削除しました。")
			} else {
				logger.Println("削除をキャンセルしました。処理を中断します。")
				os.Exit(0)
			}
		}
	}

	// --- 出力ディレクトリ作成 ---
	if err := os.MkdirAll(destDir, 0755); err != nil {
		logger.Fatalf("エラー: 出力ディレクトリ '%s' の作成に失敗: %v", destDir, err)
	}

	// --- -restart オプション処理 ---
	if restart {
		if isSingleFileMode {
			logger.Println("警告: 単一ファイルモードでは -restart オプションは無視されます。")
		} else {
			if err := removeRestartFiles(destDir); err != nil {
				logger.Fatalf("エラー: -restart 処理中にエラーが発生: %v", err)
			}
		}
	}

	// --- 一時ディレクトリ作成 ---
	tempDir, err := os.MkdirTemp("", tempDirPrefix)
	if err != nil {
		logger.Fatalf("エラー: 一時ディレクトリの作成に失敗: %v", err)
	}
	logger.Printf("一時ディレクトリ: %s", tempDir)
	defer func() {
		debugLogPrintf("一時ディレクトリ削除: %s", tempDir)
		if err := os.RemoveAll(tempDir); err != nil {
			logger.Printf("警告: 一時ディレクトリ '%s' の削除に失敗: %v", tempDir, err)
		}
	}()

	// --- メイン処理の分岐 ---
	var allErrors []string // 処理中のエラーを格納するスライス

	if isSingleFileMode {
		// === 単一ファイル処理モード ===
		logger.Println("--- 単一ファイル処理モード開始 ---")
		inputFile := sourceDir

		inputFilename := filepath.Base(inputFile)
		inputExt := strings.ToLower(filepath.Ext(inputFilename))

		if _, isVideo := videoExtensions[inputExt]; !isVideo {
			logger.Fatalf("エラー: 入力ファイル '%s' はサポートされている動画拡張子ではありません。", inputFile)
		}

		outputBaseName := strings.TrimSuffix(inputFilename, filepath.Ext(inputFilename)) + outputSuffix
		outputFile := filepath.Join(destDir, outputBaseName)

		logger.Printf("処理対象: %s -> %s", inputFile, outputFile)

		if err := processVideoFile(inputFile, outputFile, tempDir, ffmpegPriority, hwEncoder, cpuEncoder, hwEncoderOptions, cpuEncoderOptions, timeoutSeconds, quickModeFlag); err != nil {
			allErrors = append(allErrors, fmt.Sprintf("%s: %v", inputFilename, err))
		}
		logger.Println("--- 単一ファイル処理モード終了 ---")

	} else {
		// === ディレクトリ処理モード ===
		logger.Println("--- ディレクトリ処理モード開始 ---")

		// --- ファイルリスト作成 ---
		logger.Println("--- ファイルリスト作成開始 ---")
		var videoFiles []string
		var otherFiles []string // 動画以外のファイルを格納
		fileCount := 0
		walkErr := filepath.WalkDir(sourceDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				logger.Printf("警告: ディレクトリ/ファイル '%s' へのアクセスエラー: %v。スキップします。", path, err)
				if d != nil && d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if d.IsDir() {
				return nil
			}
			fileCount++
			if fileCount%1000 == 0 && fileCount > 0 {
				logger.Printf("ファイルリスト作成中... %d 件スキャン済み", fileCount)
			}
			ext := strings.ToLower(filepath.Ext(path))
			if _, isVideo := videoExtensions[ext]; isVideo {
				videoFiles = append(videoFiles, path)
			} else {
				otherFiles = append(otherFiles, path)
			}
			return nil
		})
		if walkErr != nil {
			logger.Fatalf("エラー: ファイルリスト作成中に予期せぬエラーが発生: %v", walkErr)
		}
		logger.Printf("ファイルリスト作成完了。 動画: %d件, その他: %d件 (総ファイル: %d件)", len(videoFiles), len(otherFiles), fileCount)

		// --- 一時ファイルリスト書き出し (-usetemp 指定時) ---
		if usingTempFileList && len(videoFiles) > 0 {
			tempFileName := fmt.Sprintf("GoTransAV1_FileList_%s.txt", startTime.Format("20060102_150405"))
			tempFileListPath = filepath.Join(tempDir, tempFileName)
			logger.Printf("-usetemp 指定のため、動画ファイルリストを一時ファイルに書き出します: %s", tempFileListPath)
			file, err := os.Create(tempFileListPath)
			if err != nil {
				logger.Printf("警告: 一時リスト作成失敗(%s): %v。メモリ上のリストを使用します。", tempFileListPath, err)
				usingTempFileList = false
			} else {
				writer := bufio.NewWriter(file)
				for _, vf := range videoFiles {
					if _, err := writer.WriteString(vf + "\n"); err != nil {
						file.Close()
						logger.Printf("警告: 一時リスト書き込み失敗 (%s): %v。メモリ上のリストを使用します。", tempFileListPath, err)
						usingTempFileList = false
						_ = os.Remove(tempFileListPath)
						break
					}
				}
				if usingTempFileList {
					if err := writer.Flush(); err != nil {
						file.Close()
						logger.Printf("警告: 一時リストフラッシュ失敗 (%s): %v。メモリ上のリストを使用します。", tempFileListPath, err)
						usingTempFileList = false
						_ = os.Remove(tempFileListPath)
					} else if err := file.Close(); err != nil {
						logger.Printf("警告: 一時リストクローズ失敗 (%s): %v。メモリ上のリストを使用します。", tempFileListPath, err)
						usingTempFileList = false
						_ = os.Remove(tempFileListPath)
					} else {
						videoFiles = nil
						logger.Printf("一時リスト書き込み完了。")
					}
				}
			}
		} else if usingTempFileList && len(videoFiles) == 0 {
			logger.Println("-usetemp 指定ですが、処理対象の動画ファイルがないため一時リストは作成しません。")
			usingTempFileList = false
		} else {
			logger.Printf("メモリ上のリストを使用します (-usetemp 未指定または動画なし)。")
		}

		// --- その他のファイルコピー処理 ---
		logger.Println("--- その他のファイルコピー処理開始 ---")
		var otherCopyErrors []string
		otherCount := len(otherFiles)
		if otherCount > 0 {
			logger.Printf("%d 件のその他のファイルをコピーします...", otherCount)
			for i, otherFile := range otherFiles {
				relPath, err := filepath.Rel(sourceDir, otherFile)
				if err != nil {
					errMsg := fmt.Sprintf("その他ファイル相対パス計算失敗 (%s): %v", otherFile, err)
					logger.Printf("エラー (%d/%d): %s", i+1, otherCount, errMsg)
					otherCopyErrors = append(otherCopyErrors, errMsg)
					continue
				}
				otherOutputPath := filepath.Join(destDir, relPath)
				logger.Printf("コピー中 (%d/%d): %s", i+1, otherCount, filepath.Base(otherFile))
				// ★★★ 修正点: copyImageFile -> copyOtherFile ★★★
				if err := copyOtherFile(otherFile, otherOutputPath); err != nil {
					// copyOtherFile 内でスキップログは出さないので、エラーのみ記録
					errMsg := fmt.Sprintf("その他ファイルコピー失敗(%s): %v", filepath.Base(otherFile), err)
					logger.Printf("エラー (%d/%d): %s", i+1, otherCount, errMsg)
					otherCopyErrors = append(otherCopyErrors, errMsg)
				}
			}
		} else {
			logger.Println("コピー対象のその他のファイルはありません。")
		}
		logger.Println("--- その他のファイルコピー処理終了 ---")
		allErrors = append(allErrors, otherCopyErrors...)

		// --- 動画エンコード処理 ---
		logger.Println("--- 動画エンコード処理開始 ---")
		var videoProcessingErrors []string
		if usingTempFileList {
			logger.Printf("一時リスト %s から動画パスを読み込んで処理します。", tempFileListPath)
			file, err := os.Open(tempFileListPath)
			if err != nil {
				logger.Fatalf("エラー: 一時リスト '%s' の読み込みに失敗: %v", tempFileListPath, err)
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
				if err := processVideoFile(filePath, outputPath, tempDir, ffmpegPriority, hwEncoder, cpuEncoder, hwEncoderOptions, cpuEncoderOptions, timeoutSeconds, quickModeFlag); err != nil {
					videoProcessingErrors = append(videoProcessingErrors, fmt.Sprintf("%s: %v", filepath.Base(filePath), err))
				}
			}
			if err := scanner.Err(); err != nil {
				logger.Printf("エラー: 一時リストのスキャン中にエラーが発生: %v", err)
				videoProcessingErrors = append(videoProcessingErrors, fmt.Sprintf("一時リストスキャンエラー: %v", err))
			}
		} else {
			videoCount := len(videoFiles)
			if videoCount > 0 {
				logger.Printf("メモリ上のリストから %d 件の動画を処理します。", videoCount)
				for i, vidFile := range videoFiles {
					logger.Printf("--- 動画エンコード (%d/%d): %s ---", i+1, videoCount, filepath.Base(vidFile))
					outputPath, pathErr := getOutputPath(vidFile, sourceDir, destDir)
					if pathErr != nil {
						errMsg := fmt.Sprintf("動画出力パス計算失敗 (%s): %v", vidFile, pathErr)
						logger.Printf("エラー: %s", errMsg)
						videoProcessingErrors = append(videoProcessingErrors, errMsg)
						continue
					}
					if err := processVideoFile(vidFile, outputPath, tempDir, ffmpegPriority, hwEncoder, cpuEncoder, hwEncoderOptions, cpuEncoderOptions, timeoutSeconds, quickModeFlag); err != nil {
						videoProcessingErrors = append(videoProcessingErrors, fmt.Sprintf("%s: %v", filepath.Base(vidFile), err))
					}
				}
			} else {
				logger.Println("エンコード対象の動画ファイルはありません。")
			}
		}
		logger.Println("--- 動画エンコード処理終了 ---")
		allErrors = append(allErrors, videoProcessingErrors...)

		logger.Println("--- ディレクトリ処理モード終了 ---")
	}

	// --- 終了処理 ---
	endTime := time.Now()
	elapsedTime := endTime.Sub(startTime)
	logger.Printf("総処理時間: %v", elapsedTime.Round(time.Second))

	if len(allErrors) > 0 {
		logger.Printf("--- 処理中に %d 件のエラーが発生しました ---", len(allErrors))
		limit := 20
		for i, e := range allErrors {
			if i >= limit {
				logger.Printf("  ...他 %d 件のエラー (詳細はログファイルを確認してください)", len(allErrors)-limit)
				break
			}
			logger.Printf("  [%d] %s", i+1, e)
		}
		os.Exit(1) // エラー終了コード
	} else {
		logger.Println("全ての処理が正常に完了しました。")
	}
}
