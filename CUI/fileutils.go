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
var (
	// 処理対象とする動画ファイルの拡張子マップ (キーは小文字)
	videoExtensions = map[string]struct{}{
		".mp4": {}, ".avi": {}, ".mov": {}, ".mkv": {}, ".wmv": {}, ".flv": {},
		".webm": {}, ".m4v": {}, ".mpeg": {}, ".mpg": {}, ".ts": {}, ".mts": {},
		".m2ts": {}, ".3gp": {}, ".asf": {}, ".divx": {},
	}
	// コピー対象とする画像ファイルの拡張子マップ (キーは小文字)
	// 注意: ここに含まれないファイルも「その他のファイル」としてコピーされる
	imageExtensions = map[string]struct{}{
		".jpg": {}, ".jpeg": {}, ".png": {}, ".gif": {}, ".bmp": {}, ".tif": {},
		".tiff": {}, ".webp": {}, ".heic": {}, ".heif": {}, ".raw": {}, ".cr2": {},
		".nef": {}, ".orf": {}, ".sr2": {}, ".svg": {}, ".avif": {},
	}
	// -restart オプションで削除対象とするマーカーファイルのサフィックス
	failedMarkersToDelete = []string{".failed", ".timeout", ".error", ".unreadable", ".failed_"} // .failed_NN も対象に含める
)

// 出力ファイル名のサフィックス
const outputSuffix = "_AV1.mp4"

// getOutputPath: 入力ファイルパスに対応する出力ファイルパスを生成する
// inputFile: 入力ファイルのフルパス
// srcRoot: 入力元のルートディレクトリパス
// dstRoot: 出力先のルートディレクトリパス
func getOutputPath(inputFile, srcRoot, dstRoot string) (string, error) {
	// 入力元ルートからの相対パスを計算
	relPath, err := filepath.Rel(srcRoot, inputFile)
	if err != nil {
		// srcRoot が inputFile の親でない場合などにエラー
		return "", fmt.Errorf("相対パス計算失敗 ('%s' は '%s' 内にありませんか?): %w", inputFile, srcRoot, err)
	}

	// 拡張子を除去して新しいサフィックスを付与
	ext := filepath.Ext(relPath)
	baseNameWithoutExt := strings.TrimSuffix(relPath, ext) // 拡張子を除去
	outputBaseName := baseNameWithoutExt + outputSuffix    // 新しいサフィックスを追加

	// 最終的な出力フルパスを結合 (元のディレクトリ構造を維持)
	return filepath.Join(dstRoot, outputBaseName), nil
}

// copyOtherFile: 動画以外のファイル (画像など) をコピーする関数
// inputFile: コピー元ファイルのパス
// outputFile: コピー先ファイルのパス
func copyOtherFile(inputFile, outputFile string) error {
	// 入力ファイルの情報を取得 (パーミッション維持のため)
	srcInfo, err := os.Stat(inputFile)
	if err != nil {
		return fmt.Errorf("入力ファイル '%s' 情報取得エラー: %w", inputFile, err)
	}
	if srcInfo.IsDir() {
		return fmt.Errorf("入力パス '%s' はディレクトリです", inputFile)
	}

	// 出力先にファイルが存在するかチェック
	if fileExists(outputFile) {
		// 既に存在する場合はスキップログを出力して正常終了
		// logger.Printf("スキップ (既存): %s", filepath.Base(outputFile)) // main 側でログを出すのでここでは不要かも
		return nil // 正常終了扱い
	}
	// Stat で IsNotExist 以外のエラーが出た場合は問題あり
	_, err = os.Stat(outputFile)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("出力ファイル '%s' 確認エラー: %w", outputFile, err)
	}

	// 出力ディレクトリ作成 (MkdirAll は存在してもエラーにならない)
	outputDir := filepath.Dir(outputFile)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("出力ディレクトリ '%s' 作成エラー: %w", outputDir, err)
	}

	// コピー実行 (手動コピー関数を呼び出す)
	// logger.Printf("コピー中: %s -> %s", filepath.Base(inputFile), filepath.Base(outputFile)) // main 側でログを出す
	if err := copyFileManually(inputFile, outputFile); err != nil {
		// コピー失敗時は作成された可能性のある出力ファイルを削除試行
		_ = os.Remove(outputFile)
		return fmt.Errorf("ファイルコピー失敗 (%s -> %s): %w", inputFile, outputFile, err)
	}

	return nil
}

// copyFileManually: 低レベルなファイルコピー処理 (io.Copy を使用)
// src: コピー元ファイルパス
// dst: コピー先ファイルパス
func copyFileManually(src, dst string) error {
	// コピー元のファイル情報を取得
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("コピー元 stat エラー (%s): %w", src, err)
	}
	// 通常ファイル以外はエラー
	if !sourceFileStat.Mode().IsRegular() {
		return fmt.Errorf("コピー元 '%s' は通常ファイルではありません", src)
	}

	// コピー元ファイルを開く
	source, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("コピー元オープンエラー (%s): %w", src, err)
	}
	defer source.Close() // 関数終了時にファイルを閉じる

	// コピー先ファイルを作成 (存在する場合は上書き、パーミッションはコピー元に合わせる)
	destination, err := os.OpenFile(dst, os.O_RDWR|os.O_CREATE|os.O_TRUNC, sourceFileStat.Mode())
	if err != nil {
		return fmt.Errorf("コピー先作成エラー (%s): %w", dst, err)
	}
	// defer destination.Close() // Close はコピー完了後、エラーチェックと共に行う

	// データをコピー
	bytesCopied, err := io.Copy(destination, source)
	if err != nil {
		destination.Close() // エラーでも Close を試みる
		_ = os.Remove(dst)  // コピー失敗時は作成したファイルを削除試行
		return fmt.Errorf("io.Copy エラー (%s -> %s): %w", src, dst, err)
	}
	debugLogPrintf("%d バイトコピー完了: %s -> %s", bytesCopied, filepath.Base(src), filepath.Base(dst))

	// コピー先ファイルのデータをディスクに書き込む (Sync)
	// なくても動くことが多いが、信頼性を高めるため
	// err = destination.Sync()
	// if err != nil {
	// 	destination.Close()
	// 	_ = os.Remove(dst)
	// 	return fmt.Errorf("コピー先 Sync エラー (%s): %w", dst, err)
	// }

	// コピー先ファイルを閉じる (エラーチェックも行う)
	err = destination.Close()
	if err != nil {
		_ = os.Remove(dst) // Close 失敗でもファイルを削除試行
		return fmt.Errorf("コピー先クローズエラー (%s): %w", dst, err)
	}

	return nil
}

// handleProcessingFailure: ffmpeg 処理失敗時の後処理 (クリーンアップ)
// originalInputFile: 処理対象だった元のファイルパス
// finalOutputFile: 本来の出力先ファイルパス
// result: ffmpeg 実行結果 (ffmpegResult 構造体)
// isQuickMode: Quick モードで実行されたか
// renamedSourcePath: Quick モード時のリネーム後ソースパス (なければ空文字)
// tempOutputPath: Temp モード時の一時出力パス (なければ空文字)
func handleProcessingFailure(originalInputFile string, finalOutputFile string, result ffmpegResult, isQuickMode bool, renamedSourcePath string, tempOutputPath string) error {
	// エラーログは呼び出し元 (processVideoFile) で出力済みなので、ここではクリーンアップ処理に専念
	debugLogPrintf("失敗後処理開始: Original: %s, QuickMode: %t", originalInputFile, isQuickMode)

	if isQuickMode {
		// === Quick モード失敗時 ===
		// 1. リネームしたソースファイルを元に戻す試行
		if renamedSourcePath != "" && fileExists(renamedSourcePath) {
			debugLogPrintf("QuickMode失敗: ソースを元に戻します: %s -> %s", renamedSourcePath, originalInputFile)
			if err := os.Rename(renamedSourcePath, originalInputFile); err != nil {
				// リネームバック失敗は警告ログに留める
				logger.Printf("警告 [Quick Mode]: ソースのリネームバック失敗 (%s -> %s): %v", renamedSourcePath, originalInputFile, err)
				logger.Printf("  手動で '%s' を '%s' に戻してください。", renamedSourcePath, originalInputFile)
			}
		} else if renamedSourcePath != "" {
			// リネーム後のファイルが見つからない場合 (通常ありえないはず)
			debugLogPrintf("リネーム後のソースファイル '%s' が見つからないため、リネームバックはスキップします。", renamedSourcePath)
		}

		// 2. 不完全に生成された可能性のある出力ファイルを削除試行
		if fileExists(finalOutputFile) {
			debugLogPrintf("QuickMode失敗: 不完全な出力ファイルを削除試行: %s", finalOutputFile)
			if err := os.Remove(finalOutputFile); err != nil {
				logger.Printf("警告 [Quick Mode]: 不完全な出力ファイルの削除失敗 (%s): %v", finalOutputFile, err)
				logger.Printf("  手動で '%s' を確認・削除してください。", finalOutputFile)
			}
		}
	} else {
		// === Temp モード失敗時 ===
		// 1. 一時ディレクトリにコピーした入力ファイルを削除試行 (tempDir ごと削除されるので不要かも)
		// tempInputPath := filepath.Join(filepath.Dir(tempOutputPath), filepath.Base(originalInputFile)) // これでいいか要確認
		// if fileExists(tempInputPath) { ... os.Remove ... }

		// 2. 一時出力ファイルを削除試行
		if tempOutputPath != "" && fileExists(tempOutputPath) {
			debugLogPrintf("TempMode失敗: 一時出力ファイルを削除試行: %s", tempOutputPath)
			if err := os.Remove(tempOutputPath); err != nil {
				logger.Printf("警告 [Temp Mode]: 一時出力ファイルの削除失敗 (%s): %v", tempOutputPath, err)
			}
		}
		// Temp モードでは元の入力ファイル (originalInputFile) は変更されない
	}

	// 最終的なエラーメッセージを生成して返す
	if result.err != nil {
		// ffmpegResult に詳細なエラーが含まれている場合
		return result.err // そのまま返す
	}
	// result.err が nil だが失敗した場合 (ExitCode が 0 以外など)
	return fmt.Errorf("ffmpeg 処理失敗 (終了コード: %d, タイムアウト: %t)", result.exitCode, result.timedOut)
}

// fileExists: 指定されたパスがファイルとして存在するかどうかをチェック
func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return false // 存在しない
		}
		// Stat で他のエラー (権限など) が発生した場合
		logger.Printf("警告: ファイル状態確認エラー (%s): %v", filename, err)
		return false // 存在するか不明な場合は false 扱い
	}
	// 存在し、かつディレクトリではない場合に true
	return !info.IsDir()
}

// createMarkerFile: 処理結果を示すマーカーファイルを作成する
// markerPath: 作成するマーカーファイルのフルパス
// content: マーカーファイルに書き込む内容 (エラー詳細など)
func createMarkerFile(markerPath string, content string) {
	debugLogPrintf("マーカーファイル作成試行: %s", markerPath)
	// マーカーファイル用のディレクトリが存在しない場合は作成
	markerDir := filepath.Dir(markerPath)
	if err := os.MkdirAll(markerDir, 0755); err != nil {
		logger.Printf("警告: マーカー用ディレクトリ作成失敗 (%s): %v", markerDir, err)
		// ディレクトリが作れなくてもファイルの書き込みは試行する
	}

	// 書き込む内容が長すぎる場合は切り詰める (ファイルシステム制限対策)
	const maxContentLength = 512 // 最大文字数 (バイト数ではない)
	fileContent := content
	if len(fileContent) > maxContentLength {
		fileContent = fileContent[:maxContentLength] + "...(truncated)"
	}

	// ファイルに内容を書き込む (既存ファイルは上書き)
	if err := os.WriteFile(markerPath, []byte(fileContent), 0644); err != nil {
		logger.Printf("警告: マーカーファイル書き込み失敗 (%s): %v", markerPath, err)
	} else {
		debugLogPrintf("マーカーファイル作成成功: %s", markerPath)
	}
}

// removeRestartFiles: -restart オプション実行時に、出力ディレクトリ内の不要ファイルを削除する
// dir: 対象の出力ディレクトリパス
func removeRestartFiles(dir string) error {
	logger.Printf("-Restart: ディレクトリ '%s' 内のエラーマーカーと0バイト動画ファイルを削除します...", dir)
	filesRemoved := 0
	walkErr := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// ディレクトリ走査中にエラーが発生した場合
			logger.Printf("警告: ディレクトリ '%s' 走査エラー: %v。スキップします。", path, err)
			// エラーが発生したのがディレクトリなら、その中身は処理しない
			if d != nil && d.IsDir() {
				return filepath.SkipDir
			}
			return nil // ファイルエラーなら次の要素へ
		}
		// ディレクトリ自体は処理しない
		if d.IsDir() {
			return nil
		}

		// ファイル名を小文字にして比較
		fileNameLower := strings.ToLower(d.Name())

		// 1. マーカーファイルの削除チェック
		for _, markerSuffix := range failedMarkersToDelete {
			// failed_ もチェックするため HasPrefix を使う
			if strings.HasSuffix(fileNameLower, markerSuffix) || (markerSuffix == ".failed_" && strings.Contains(fileNameLower, ".failed_")) {
				debugLogPrintf("-Restart: マーカーファイル削除: %s", path)
				if err := os.Remove(path); err != nil {
					logger.Printf("警告: マーカーファイル削除失敗 (%s): %v", path, err)
				} else {
					filesRemoved++
				}
				return nil // マーカーファイルならここで処理終了
			}
		}

		// 2. 0バイト動画ファイルの削除チェック
		ext := strings.ToLower(filepath.Ext(fileNameLower))
		if _, isVideo := videoExtensions[ext]; isVideo { // 動画拡張子かチェック
			info, infoErr := d.Info() // fs.DirEntry からファイル情報を取得
			if infoErr != nil {
				logger.Printf("警告: ファイル情報取得エラー (%s): %v。スキップします。", path, infoErr)
				return nil
			}
			if info.Size() == 0 { // ファイルサイズが0かチェック
				debugLogPrintf("-Restart: 0バイト動画ファイル削除: %s", path)
				if err := os.Remove(path); err != nil {
					logger.Printf("警告: 0バイト動画ファイル削除失敗 (%s): %v", path, err)
				} else {
					filesRemoved++
				}
				return nil // 0バイト動画ならここで処理終了
			}
		}
		return nil // 上記以外は削除しない
	})

	if walkErr != nil {
		// WalkDir 自体のエラー
		return fmt.Errorf("-Restart 処理中に予期せぬエラー: %w", walkErr)
	}
	logger.Printf("-Restart: %d 個のマーカーファイルまたは0バイト動画ファイルを削除しました。", filesRemoved)
	return nil
}

// getVideoExtList: サポートする動画拡張子のリストを文字列で返す (Usage 表示用)
func getVideoExtList() string {
	keys := make([]string, 0, len(videoExtensions))
	for k := range videoExtensions {
		keys = append(keys, k)
	}
	sort.Strings(keys) // アルファベット順にソート
	return strings.Join(keys, ", ")
}

// getImageExtList: コピー対象の画像拡張子のリストを文字列で返す (Usage 表示用)
func getImageExtList() string {
	keys := make([]string, 0, len(imageExtensions))
	for k := range imageExtensions {
		keys = append(keys, k)
	}
	sort.Strings(keys) // アルファベット順にソート
	return strings.Join(keys, ", ")
}
