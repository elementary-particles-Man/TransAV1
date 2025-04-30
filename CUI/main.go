package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/fs"
	// "log" // log パッケージは logutils.go でのみ使用するため削除
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// --- グローバル変数 ---
var (
	sourceDir string
	destDir   string
	ffmpegDir string
	// ffmpegPriority は ffmpeg.go で使用
	hwEncoder         string
	cpuEncoder        string
	hwEncoderOptions  string
	cpuEncoderOptions string
	timeoutSeconds    int
	logToFile         bool
	debugMode         bool
	restart           bool
	forceStart        bool
	quickModeFlag     bool
	usingTempFileList bool   // 一時ファイルリストを使用するかどうか
	tempFileListPath  string // 一時ファイルリストのパス

	ffmpegPath  string
	ffprobePath string
	startTime   time.Time
	// logger, debugLogger は logutils.go で定義
)

// main 関数内で使用される定数やヘルパー関数は main.go に残す
const (
	// outputSuffix は fileutils.go に移動済みのため削除
	tempDirPrefix = "go_transav1_"
)

var (
	// videoExtensions は fileutils.go に移動済みのため削除
	// imageExtensions は fileutils.go に移動済みのため削除
	// failedMarkersToDelete は fileutils.go に移動済みのため削除
)


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
`, progName, progName, getVideoExtList(), outputSuffix, getImageExtList()) // fileutils.go の関数を呼び出し

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
	fmt.Fprintf(os.Stderr, "  -hwopt \"<オプション>\"\n\tHWエンコーダ用の追加ffmpegオプション (引用符で囲む)。\n\t(デフォルト: \"%s\")\n", "-cq 25 -preset p5")
	fmt.Fprintf(os.Stderr, "  -cpuopt \"<オプション>\"\n\tCPUエンコーダ用の追加ffmpegオプション (引用符で囲む)。\n\t(デフォルト: \"%s\")\n", "-crf 28 -preset 7")
	fmt.Fprintf(os.Stderr, "  -timeout <秒>\n\tffmpeg 各処理のタイムアウト秒数 (0で無効)。\n\t(デフォルト: %d)\n", 7200) // flag定義のデフォルト値を反映
	fmt.Fprintf(os.Stderr, "  -quick\n\t高速モード: 一時コピーを行わず直接エンコード (非推奨)\n\t(デフォルト: false)\n") // Quickモードの説明を追加
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

// --- main 関数 ---
func main() {
	startTime = time.Now()
	flag.Usage = printUsage

	// ffmpegPriority は main.go で定義し、ffmpeg.go に渡す
	var ffmpegPriority string
	flag.StringVar(&sourceDir, "s", "", "入力元ディレクトリ (必須)")
	flag.StringVar(&destDir, "o", "", "出力先ディレクトリ (必須)")
	defaultFfmpegDir := getDefaultFfmpegDir()
	flag.StringVar(&ffmpegDir, "ffmpegdir", defaultFfmpegDir, "ffmpeg/ffprobe 格納ディレクトリ")
	flag.StringVar(&ffmpegPriority, "priority", "Idle", "プロセス優先度 (idle|BelowNormal|Normal|AboveNormal)")
	flag.StringVar(&hwEncoder, "hwenc", "av1_nvenc", "優先HWエンコーダ名")
	flag.StringVar(&cpuEncoder, "cpuenc", "libsvtav1", "フォールバックCPUエンコーダ名")
	const defaultHwOpt = "-cq 25 -preset p5"
	const defaultCpuOpt = "-crf 28 -preset 7"
	const defaultTimeout = 7200
	flag.StringVar(&hwEncoderOptions, "hwopt", defaultHwOpt, "HWエンコーダ用ffmpegオプション")
	flag.StringVar(&cpuEncoderOptions, "cpuopt", defaultCpuOpt, "CPUエンコーda用追加ffmpegオプション")
	flag.IntVar(&timeoutSeconds, "timeout", defaultTimeout, "タイムアウト秒数 (0で無効)")
	flag.BoolVar(&quickModeFlag, "quick", false, "高速モード: 一時コピーを行わず直接エンコード")
	flag.BoolVar(&logToFile, "log", false, "ログをファイルにも書き出す")
	flag.BoolVar(&debugMode, "debug", false, "詳細ログ出力")
	flag.BoolVar(&restart, "restart", false, "マーカー/0バイト動画削除")
	flag.BoolVar(&forceStart, "force", false, "出力Dirを強制削除 (確認あり)")
	flag.BoolVar(&usingTempFileList, "usetemp", false, "一時ファイルリストを使用")

	flag.Parse()

	// logutils.go の setupLogging を呼び出し
	setupLogging(destDir, startTime, logToFile, debugMode)

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

	// ffmpeg/ffprobe パスの検索ロジック
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
			ffprobePath = "" // 見つからなければ空文字列
		} else {
			logger.Printf("ffprobe をPATHから使用: %s", ffprobePathFromPath)
			ffprobePath = ffprobePathFromPath
		}
	} else {
		ffprobePath, _ = filepath.Abs(ffprobePath)
		logger.Printf("ffprobe を使用: %s", ffprobePath)
	}


	sourceInfo, err := os.Stat(sourceDir)
	if err != nil {
		logger.Fatalf("エラー: 入力元 '%s' の情報取得に失敗: %v", sourceDir, err)
	}

	isSingleFileMode := !sourceInfo.IsDir()

	destInfo, err := os.Stat(destDir)
	if err != nil {
		if os.IsNotExist(err) && !isSingleFileMode {
			logger.Printf("情報: 出力先ディレクトリ '%s' が存在しません。作成を試みます。", destDir)
		} else if os.IsNotExist(err) && isSingleFileMode {
			logger.Fatalf("エラー: 単一ファイルモードでは出力先ディレクトリ '%s' が存在する必要があります。", destDir)
		} else {
			logger.Fatalf("エラー: 出力先 '%s' の情報取得に失敗: %v", destDir, err)
		}
	} else if !destInfo.IsDir() {
		logger.Fatalf("エラー: 出力先 '%s' はディレクトリではありません。", destDir)
	}

	if forceStart {
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

	// 出力ディレクトリが存在しない場合は作成
	if err := os.MkdirAll(destDir, 0755); err != nil {
		logger.Fatalf("エラー: 出力Dir作成失敗 (%s): %v", destDir, err)
	}

	if restart {
		// fileutils.go の removeRestartFiles を呼び出し
		if err := removeRestartFiles(destDir); err != nil {
			logger.Fatalf("エラー: -restart 処理エラー: %v", err)
		}
	}

	// 一時ディレクトリ作成
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

	var allErrors []string

	if isSingleFileMode {
		logger.Println("--- 単一ファイル処理モード開始 ---")
		inputFile := sourceDir

		inputFilename := filepath.Base(inputFile)
		inputExt := filepath.Ext(inputFilename)

		// fileutils.go の videoExtensions を使用
		if _, isVideo := videoExtensions[strings.ToLower(inputExt)]; !isVideo {
			logger.Fatalf("エラー: 入力ファイル '%s' はサポートされている動画ファイルではありません。", inputFile)
		}

		// fileutils.go の getOutputPath を呼び出し
		outputFile, err := getOutputPath(inputFile, filepath.Dir(inputFile), destDir)
		if err != nil {
			logger.Fatalf("エラー: 出力パス計算失敗 (%s): %v", inputFile, err)
		}

		logger.Printf("処理対象: %s -> %s", inputFile, outputFile)

		// ffmpeg.go の processVideoFile を呼び出し
		if err := processVideoFile(inputFile, outputFile, tempDir, ffmpegPriority, hwEncoder, cpuEncoder, hwEncoderOptions, cpuEncoderOptions, timeoutSeconds, quickModeFlag); err != nil {
			allErrors = append(allErrors, fmt.Sprintf("%s: %v", inputFilename, err))
		}
		logger.Println("--- 単一ファイル処理モード終了 ---")

	} else {
		// ディレクトリ処理モード
		logger.Println("--- ディレクトリ処理モード開始 ---")

		// ファイルリスト作成
		logger.Println("--- ファイルリスト作成開始 ---")
		var videoFiles []string
		var imageFiles []string
		var otherFiles []string // その他のファイルを格納するスライス
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
			// fileutils.go の videoExtensions を使用
			if _, isVideo := videoExtensions[ext]; isVideo {
				videoFiles = append(videoFiles, path)
			} else if _, isImage := imageExtensions[ext]; isImage { // fileutils.go の imageExtensions を使用
				imageFiles = append(imageFiles, path)
			} else {
				otherFiles = append(otherFiles, path) // その他のファイルパスをotherFiles に追加
				debugLogPrintf("その他のファイルとしてリストアップ: %s", path)
			}
			if fileCount%1000 == 0 && fileCount > 0 {
				logger.Printf("リスト作成中... %d件スキャン済み", fileCount)
			}
			return nil
		})
		if err != nil {
			logger.Fatalf("エラー: ファイルリスト作成中にエラー: %v", err)
		}
		logger.Printf("リスト作成完了。 動画: %d件, 画像: %d件, その他: %d件 (総ファイル: %d件)", len(videoFiles), len(imageFiles), len(otherFiles), fileCount)

		// 一時ファイルリスト書き出し判定
		if usingTempFileList {
			tempFileName := fmt.Sprintf("GoTransAV1_FileList_%s.txt", startTime.Format("20060102_150405"))
			tempFileListPath = filepath.Join(tempDir, tempFileName)
			logger.Printf("-usetemp 指定のため、一時リストを使用: %s", tempFileListPath)
			file, err := os.Create(tempFileListPath)
			if err != nil {
				logger.Fatalf("エラー: 一時リスト作成失敗(%s): %v", tempFileListPath, err)
				usingTempFileList = false // 失敗したらメモリモードにフォールバック
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
				videoFiles = nil // メモリ上のリストを解放
				logger.Printf("一時リスト書込完了。")
			}
		} else {
			logger.Printf("メモリ上のリストを使用 (-usetemp 未指定)")
		}

		// 画像コピー処理
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
				// fileutils.go の copyImageFile を呼び出し
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
		allErrors = append(allErrors, imageCopyErrors...) // エラーを集計

		// --- その他のファイルコピー処理 ---
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

				// fileutils.go の copyImageFile を再利用 (ファイルコピーロジックは同じため)
				if err := copyImageFile(otherFile, otherOutputPath); err != nil {
					errMsg := fmt.Sprintf("その他ファイルコピー失敗(%s): %v", filepath.Base(otherFile), err)
					logger.Printf("エラー: %s", errMsg)
					otherCopyErrors = append(otherCopyErrors, errMsg)
				}
			}
		} else {
			logger.Println("コピー対象のその他のファイルはありません。")
		}
		logger.Println("--- その他のファイルコピー処理終了 ---")
		allErrors = append(allErrors, otherCopyErrors...) // エラーを集計

		// --- 動画エンコード処理 ---
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
				// fileutils.go の getOutputPath を呼び出し
				outputPath, pathErr := getOutputPath(filePath, sourceDir, destDir)
				if pathErr != nil {
					errMsg := fmt.Sprintf("動画出力パス計算失敗 (%s): %v", filePath, pathErr)
					logger.Printf("エラー: %s", errMsg)
					videoProcessingErrors = append(videoProcessingErrors, errMsg)
					continue
				}
				// ffmpeg.go の processVideoFile を呼び出し
				if err := processVideoFile(filePath, outputPath, tempDir, ffmpegPriority, hwEncoder, cpuEncoder, hwEncoderOptions, cpuEncoderOptions, timeoutSeconds, quickModeFlag); err != nil {
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
					// fileutils.go の getOutputPath を呼び出し
					outputPath, pathErr := getOutputPath(vidFile, sourceDir, destDir)
					if pathErr != nil {
						errMsg := fmt.Sprintf("動画出力パス計算失敗 (%s): %v", vidFile, pathErr)
						logger.Printf("エラー: %s", errMsg)
						videoProcessingErrors = append(videoProcessingErrors, errMsg)
						continue
					}
					// ffmpeg.go の processVideoFile を呼び出し
					if err := processVideoFile(vidFile, outputPath, tempDir, ffmpegPriority, hwEncoder, cpuEncoder, hwEncoderOptions, cpuEncoderOptions, timeoutSeconds, quickModeFlag); err != nil {
						videoProcessingErrors = append(videoProcessingErrors, fmt.Sprintf("%s: %v", filepath.Base(vidFile), err))
					}
				}
			} else {
				logger.Println("エンコード対象の動画ファイルはありません。")
			}
		}
		logger.Println("--- 動画エンコード処理終了 ---")
		allErrors = append(allErrors, videoProcessingErrors...) // エラーを集計

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
				logger.Printf("  ...他 %d 件のエラー (ログを確認してください)", len(allErrors)-limit)
				break
			}
			logger.Printf("  [%d] %s", i+1, e)
		}
		os.Exit(1) // エラー終了
	} else {
		logger.Println("全ての処理が正常に完了しました。")
	}
}
