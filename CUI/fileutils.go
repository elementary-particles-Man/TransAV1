package main

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// --- 定数と変数 (ファイル関連) ---
// main.go から移動
var (
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
	failedMarkersToDelete = []string{".failed", ".timeout", ".error", ".unreadable"}
)

// main.go から移動
const (
	outputSuffix = "_AV1.mp4"
	// tempDirPrefix は main.go で使用
)


// getOutputPath: 入力パスに対応する出力パスを生成
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
	outputBaseName := baseNameWithoutExt + outputSuffix // outputSuffix は fileutils.go で定義

	// 出力先のサブディレクトリパスを計算
	outputRelDir := filepath.Dir(relPath) // 元の相対ディレクトリ構造
	finalDestDir := filepath.Join(dstRoot, outputRelDir)

	// 最終的な出力フルパスを結合
	return filepath.Join(finalDestDir, filepath.Base(outputBaseName)), nil
}

// copyImageFile: 画像ファイルをコピーする関数
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
	// fileExists 関数を使用
	if fileExists(outputFile) {
		// ファイルが存在する場合、スキップ
		logger.Printf("スキップ (既存): %s", filepath.Base(outputFile)) // logutils.go の logger を使用
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
	logger.Printf("コピー中: %s -> %s", filepath.Base(inputFile), filepath.Base(outputFile)) // logutils.go の logger を使用

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

// copyFileManually: 手動でファイルをコピーするヘルパー関数
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

	destination, err := os.OpenFile(dst, os.O_RDWR|os.O_CREATE|os.O_TRUNC, sourceFileStat.Mode())
	if err != nil {
		return fmt.Errorf("コピー先作成エラー (%s): %w", dst, err)
	}
	defer destination.Close()

	bytesCopied, err := io.Copy(destination, source)
	if err != nil {
		_ = os.Remove(dst) // コピー失敗時は作成したファイルを削除試行
		return fmt.Errorf("io.Copyエラー (%s -> %s): %w", src, dst, err)
	}
	debugLogPrintf("%d バイトコピー完了: %s -> %s", bytesCopied, src, dst) // logutils.go の debugLogPrintf を使用

	err = destination.Sync()
	if err != nil {
		_ = os.Remove(dst)
		return fmt.Errorf("コピー先Syncエラー (%s): %w", dst, err)
	}

	err = destination.Close()
	if err != nil {
		_ = os.Remove(dst)
		return fmt.Errorf("コピー先クローズエラー (%s): %w", dst, err)
	}

	return nil
}

// handleProcessingFailure: ffmpeg 処理失敗時の共通ハンドラ
// originalInputFile: 変換前の元のファイルパス
// finalOutputFile: 本来の出力先ファイルパス
// result: ffmpeg実行結果 (ffmpeg.go で定義された struct)
// isQuickMode: Quickモードで実行されていたか (main.go の変数)
// renamedSourcePath: Quickモード時のリネーム後ソースパス (Tempモード時は空文字)
// tempOutputPath: Tempモード時の一時出力パス (Quickモード時は空文字)
func handleProcessingFailure(originalInputFile string, finalOutputFile string, result ffmpegResult, isQuickMode bool, renamedSourcePath string, tempOutputPath string) error {
	if result.err != nil {
		logger.Printf("エラー: %v", result.err) // logutils.go の logger を使用
	} else {
		logger.Printf("エラー: ffmpeg処理失敗 (ExitCode: %d)", result.exitCode) // logutils.go の logger を使用
	}
	logger.Printf("メッセージ: このファイル変換時に異常終了しました。") // logutils.go の logger を使用
	logger.Printf("  元ファイルパス: %s", filepath.Dir(originalInputFile))
	logger.Printf("  元ファイル名: %s", filepath.Base(originalInputFile))

	if isQuickMode {
		// fileExists 関数を使用
		if renamedSourcePath != "" && fileExists(renamedSourcePath) {
			debugLogPrintf("QuickMode失敗、ソースを元に戻します: %s -> %s", renamedSourcePath, originalInputFile) // logutils.go の debugLogPrintf を使用
			if err := os.Rename(renamedSourcePath, originalInputFile); err != nil {
				logger.Printf("警告 [Quick Mode]: ソースのリネームバック失敗 (%s -> %s): %v", renamedSourcePath, originalInputFile, err) // logutils.go の logger を使用
				logger.Printf("  手動で '%s' を '%s' に戻してください。", renamedSourcePath, originalInputFile) // logutils.go の logger を使用
			}
		} else if renamedSourcePath != "" {
			debugLogPrintf("リネーム後のソースファイル '%s' が見つからないため、リネームバックはスキップします。", renamedSourcePath) // logutils.go の debugLogPrintf を使用
		}
		// fileExists 関数を使用
		if fileExists(finalOutputFile) {
			debugLogPrintf("QuickMode失敗、部分出力削除試行: %s", finalOutputFile) // logutils.go の debugLogPrintf を使用
			if err := os.Remove(finalOutputFile); err != nil {
				logger.Printf("警告: QuickMode失敗後の部分出力削除失敗 (%s): %v", finalOutputFile, err) // logutils.go の logger を使用
				logger.Printf("  手動で '%s' を確認・削除してください。", finalOutputFile, err) // logutils.go の logger を使用
			}
		}
	} else {
		// fileExists 関数を使用
		if tempOutputPath != "" && fileExists(tempOutputPath) {
			debugLogPrintf("TempMode失敗、一時出力削除試行: %s", tempOutputPath) // logutils.go の debugLogPrintf を使用
			if err := os.Remove(tempOutputPath); err != nil {
				logger.Printf("警告: TempMode失敗後の一時出力削除失敗 (%s): %v", tempOutputPath, err) // logutils.go の logger を使用
			}
		}
	}

	if result.err != nil {
		return result.err
	}
	return fmt.Errorf("ffmpeg処理失敗 (ExitCode: %d, TimedOut: %t)", result.exitCode, result.timedOut)
}

// fileExists: ファイルが存在するかどうかをチェック
func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false // 存在しない
	}
	if err != nil {
		// Statで他のエラー (権限など)
		logger.Printf("警告: ファイル状態確認エラー (%s): %v", filename, err) // logutils.go の logger を使用
		return false // 存在するか不明な場合は false とする
	}
	// 存在し、かつディレクトリではない
	return !info.IsDir()
}

// createMarkerFile: マーカーファイルを作成する関数
func createMarkerFile(markerPath string, content string) {
	debugLogPrintf("マーカーファイル作成: %s (内容: %s)", markerPath, content) // logutils.go の debugLogPrintf を使用
	markerDir := filepath.Dir(markerPath)
	if err := os.MkdirAll(markerDir, 0755); err != nil {
		logger.Printf("警告: マーカー用ディレクトリ作成失敗 (%s): %v", markerDir, err) // logutils.go の logger を使用
	}
	fileContent := content
	maxLen := 200
	if len(fileContent) > maxLen {
		fileContent = fileContent[:maxLen] + "..."
	}
	if err := os.WriteFile(markerPath, []byte(fileContent), 0644); err != nil {
		logger.Printf("警告: マーカーファイル書き込み失敗 (%s): %v", markerPath, err) // logutils.go の logger を使用
	}
}

// removeRestartFiles: -Restart オプション用のファイル削除関数
func removeRestartFiles(dir string) error {
	logger.Printf("-Restart: %s 内のエラーマーカーと0バイト動画ファイルを削除します...", dir) // logutils.go の logger を使用
	filesRemoved := 0
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			logger.Printf("警告: ディレクトリ '%s' 走査エラー: %v。スキップ。", path, err) // logutils.go の logger を使用
			if d != nil && d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		fileNameLower := strings.ToLower(d.Name())
		// failedMarkersToDelete を使用
		for _, markerSuffix := range failedMarkersToDelete {
			if strings.HasSuffix(fileNameLower, markerSuffix) {
				logger.Printf("-Restart: マーカー削除: %s", path) // logutils.go の logger を使用
				if err := os.Remove(path); err != nil {
					logger.Printf("警告: マーカー削除失敗 (%s): %v", path, err) // logutils.go の logger を使用
				} else {
					filesRemoved++
				}
				return nil
			}
		}
		ext := strings.ToLower(filepath.Ext(fileNameLower))
		// videoExtensions を使用
		if _, isVideo := videoExtensions[ext]; isVideo {
			info, infoErr := os.Stat(path)
			if infoErr != nil {
				logger.Printf("警告: ファイル情報取得エラー (%s): %v。スキップ。", path, infoErr) // logutils.go の logger を使用
				return nil
			}
			if info.Size() == 0 {
				logger.Printf("-Restart: 0バイト動画削除: %s", path) // logutils.go の logger を使用
				if err := os.Remove(path); err != nil {
					logger.Printf("警告: 0バイト動画削除失敗 (%s): %v", path, err) // logutils.go の logger を使用
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
	logger.Printf("-Restart: %d 個削除。", filesRemoved) // logutils.go の logger を使用
	return nil
}

// getVideoExtList: 動画拡張子リストを文字列で返す (Usage用)
func getVideoExtList() string {
	keys := make([]string, 0, len(videoExtensions)) // videoExtensions を使用
	for k := range videoExtensions {
		keys = append(keys, k)
	}
	sort.Strings(keys) // アルファベット順にソート
	return strings.Join(keys, ", ")
}

// getImageExtList: 画像拡張子リストを文字列で返す (Usage用)
func getImageExtList() string {
	keys := make([]string, 0, len(imageExtensions)) // imageExtensions を使用
	for k := range imageExtensions {
		keys = append(keys, k)
	}
	sort.Strings(keys) // アルファベット順にソート
	return strings.Join(keys, ", ")
}

