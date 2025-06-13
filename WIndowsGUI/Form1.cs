using System;
using System.Diagnostics; // Process, Debug クラス用
using System.IO; // Directory, File, Path 用
using System.Text; // StringBuilder, Encoding 用
using System.Text.RegularExpressions; // Regex用
using System.Threading; // Mutex, Thread.Sleep 用 (今回は未使用だが参考)
using System.Threading.Tasks; // 非同期処理用
using System.Windows.Forms; // Windows Forms 用
using System.Runtime.InteropServices; // P/Invoke 用

namespace TransAV1
{
    public partial class Form1 : Form
    {
        private volatile Process _transAV1Process; // volatile を追加
        private readonly string _iniFileName = "TransAV1_GUI.ini";
        private string _iniFilePath;
        private readonly object _processLock = new object(); // プロセス操作用のロックオブジェクト

        // --- ★ ログバッファリング用変数 Start ★ ---
        private readonly StringBuilder _logBuffer = new StringBuilder();
        private System.Windows.Forms.Timer _logUpdateTimer;
        private readonly object _logBufferLock = new object(); // ログバッファアクセス用ロック
        private const int LogUpdateInterval = 500; // ログUI更新間隔 (ミリ秒) - 500ms = 0.5秒
        // --- ★ ログバッファリング用変数 End ★ ---

        // --- ★ 制御文字除去用 Regex (改行を除く) ---
        private static readonly Regex _controlCharRegex = new Regex(@"[\x00-\x08\x0B\x0C\x0E-\x1F\x7F]");


        #region P/Invoke for INI Files
        [DllImport("kernel32.dll", CharSet = CharSet.Unicode, SetLastError = true)]
        private static extern long WritePrivateProfileString(string section, string key, string val, string filePath);

        [DllImport("kernel32.dll", CharSet = CharSet.Unicode, SetLastError = true)]
        private static extern int GetPrivateProfileString(string section, string key, string def, StringBuilder retVal, int size, string filePath);

        private int GetPrivateProfileInt(string section, string key, int defaultValue, string filePath)
        {
            StringBuilder temp = new StringBuilder(255);
            GetPrivateProfileString(section, key, defaultValue.ToString(), temp, 255, filePath);
            if (int.TryParse(temp.ToString(), out int result)) { return result; }
            return defaultValue;
        }

        private bool GetPrivateProfileBool(string section, string key, bool defaultValue, string filePath)
        {
            StringBuilder temp = new StringBuilder(255);
            GetPrivateProfileString(section, key, defaultValue.ToString(), temp, 255, filePath);
            if (bool.TryParse(temp.ToString(), out bool result)) { return result; }
            if (temp.ToString().Equals("1", StringComparison.OrdinalIgnoreCase) || temp.ToString().Equals("true", StringComparison.OrdinalIgnoreCase)) { return true; }
            return defaultValue;
        }
        #endregion

        public Form1()
        {
            InitializeComponent();
            _iniFilePath = Path.Combine(AppDomain.CurrentDomain.BaseDirectory, _iniFileName);
            InitializeLogUpdateTimer(); // ★ ログ更新タイマー初期化
            SetControlsState(false);
            customEncoderTextBox.Enabled = false;
        }

        // --- ★ ログタイマー関連 Start ★ ---
        private void InitializeLogUpdateTimer()
        {
            _logUpdateTimer = new System.Windows.Forms.Timer();
            _logUpdateTimer.Interval = LogUpdateInterval;
            _logUpdateTimer.Tick += LogUpdateTimer_Tick;
            _logUpdateTimer.Start();
        }

        private void LogUpdateTimer_Tick(object sender, EventArgs e)
        {
            FlushLogBuffer(); // タイマーTickでバッファをUIに反映
        }
        // --- ★ ログタイマー関連 End ★ ---


        private void Form1_Load(object sender, EventArgs e)
        {
            LoadSettingsFromIni();

            // イベントハンドラ設定
            rbEncoderNVENC.CheckedChanged += rbEncoder_CheckedChanged;
            rbEncoderCPU.CheckedChanged += rbEncoder_CheckedChanged;
            rbEncoderCustom.CheckedChanged += rbEncoder_CheckedChanged;
            cbShowLog.CheckedChanged += cbShowLog_CheckedChanged;
            inputDirButton.Click += inputDirButton_Click;
            outputDirButton.Click += outputDirButton_Click;
            ffmpegDirButton.Click += ffmpegDirButton_Click;
            runButton.Click += runButton_Click;
            stopButton.Click += stopButton_Click;
            this.FormClosing += Form1_FormClosing;

            UpdateCustomEncoderTextBoxState();
            UpdateLogRichTextBoxVisibility();
        }

        #region INI File Handling Methods
        private void LoadSettingsFromIni()
        {
            LogToBuffer($"INIファイル読み込み開始: {_iniFilePath}"); // AppendLog -> LogToBuffer
            if (!File.Exists(_iniFilePath))
            {
                LogToBuffer("INIファイルが見つかりません。デフォルト設定を使用します。"); // AppendLog -> LogToBuffer
                FlushLogBuffer(); // すぐ表示
                return;
            }
            try
            {
                StringBuilder temp = new StringBuilder(1024);
                GetPrivateProfileString("Paths", "InputDirectory", "", temp, temp.Capacity, _iniFilePath); inputDirTextBox.Text = temp.ToString();
                GetPrivateProfileString("Paths", "OutputDirectory", "", temp, temp.Capacity, _iniFilePath); outputDirTextBox.Text = temp.ToString();
                GetPrivateProfileString("Paths", "FfmpegDirectory", "", temp, temp.Capacity, _iniFilePath); ffmpegDirTextBox.Text = temp.ToString();
                GetPrivateProfileString("Compression", "SelectedOption", "Standard", temp, temp.Capacity, _iniFilePath); string compressionOption = temp.ToString(); if (compressionOption.Equals("Size", StringComparison.OrdinalIgnoreCase)) rbCompressionSize.Checked = true; else if (compressionOption.Equals("Quality", StringComparison.OrdinalIgnoreCase)) rbCompressionQuality.Checked = true; else rbCompressionStandard.Checked = true;
                GetPrivateProfileString("Encoder", "SelectedOption", "NVENC", temp, temp.Capacity, _iniFilePath); string encoderOption = temp.ToString(); GetPrivateProfileString("Encoder", "CustomEncoderName", "", temp, temp.Capacity, _iniFilePath); customEncoderTextBox.Text = temp.ToString(); if (encoderOption.Equals("CPU", StringComparison.OrdinalIgnoreCase)) rbEncoderCPU.Checked = true; else if (encoderOption.Equals("Custom", StringComparison.OrdinalIgnoreCase)) rbEncoderCustom.Checked = true; else rbEncoderNVENC.Checked = true;
                GetPrivateProfileString("Mode", "StartMode", "Normal", temp, temp.Capacity, _iniFilePath); string startMode = temp.ToString(); if (startMode.Equals("Restart", StringComparison.OrdinalIgnoreCase)) rbModeRestart.Checked = true; else if (startMode.Equals("ForceStart", StringComparison.OrdinalIgnoreCase)) rbModeForceStart.Checked = true; else rbModeNormal.Checked = true;
                cbQuickMode.Checked = GetPrivateProfileBool("Mode", "QuickMode", false, _iniFilePath);
                int timeoutSeconds = GetPrivateProfileInt("Options", "TimeoutSeconds", 7200, _iniFilePath); if (timeoutSeconds >= timeoutNumericUpDown.Minimum && timeoutSeconds <= timeoutNumericUpDown.Maximum) timeoutNumericUpDown.Value = timeoutSeconds; else { timeoutNumericUpDown.Value = 7200; LogToBuffer($"警告: INI TimeoutSeconds範囲外"); } // AppendLog -> LogToBuffer
                cbDebugMode.Checked = GetPrivateProfileBool("Options", "DebugMode", false, _iniFilePath);
                cbShowLog.Checked = GetPrivateProfileBool("Options", "ShowLog", true, _iniFilePath);
                LogToBuffer("INIファイルからの設定読み込み完了。"); // AppendLog -> LogToBuffer
            }
            catch (Exception ex)
            {
                LogToBuffer($"エラー: INI読み込み例外: {ex.Message}"); // AppendLog -> LogToBuffer
                MessageBox.Show($"設定ファイル ({_iniFileName}) 読み込みエラー。\nデフォルト設定で起動します。\n\n詳細: {ex.Message}", "設定読み込みエラー", MessageBoxButtons.OK, MessageBoxIcon.Warning);
            }
            finally
            {
                FlushLogBuffer(); // 読み込み完了またはエラー時に表示
            }
        }

        private void SaveSettingsToIni()
        {
            LogToBuffer($"INIファイル保存開始: {_iniFilePath}"); // AppendLog -> LogToBuffer
            try
            {
                WritePrivateProfileString("Paths", "InputDirectory", inputDirTextBox.Text, _iniFilePath); WritePrivateProfileString("Paths", "OutputDirectory", outputDirTextBox.Text, _iniFilePath); WritePrivateProfileString("Paths", "FfmpegDirectory", ffmpegDirTextBox.Text, _iniFilePath);
                string compressionOption = rbCompressionSize.Checked ? "Size" : (rbCompressionQuality.Checked ? "Quality" : "Standard"); WritePrivateProfileString("Compression", "SelectedOption", compressionOption, _iniFilePath);
                string encoderOption = rbEncoderCPU.Checked ? "CPU" : (rbEncoderCustom.Checked ? "Custom" : "NVENC"); WritePrivateProfileString("Encoder", "SelectedOption", encoderOption, _iniFilePath); WritePrivateProfileString("Encoder", "CustomEncoderName", customEncoderTextBox.Text, _iniFilePath);
                string startMode = "Normal"; if (rbModeRestart.Checked) startMode = "Restart"; else if (rbModeForceStart.Checked) startMode = "ForceStart"; WritePrivateProfileString("Mode", "StartMode", startMode, _iniFilePath);
                WritePrivateProfileString("Mode", "QuickMode", cbQuickMode.Checked.ToString(), _iniFilePath);
                WritePrivateProfileString("Options", "TimeoutSeconds", timeoutNumericUpDown.Value.ToString(), _iniFilePath); WritePrivateProfileString("Options", "DebugMode", cbDebugMode.Checked.ToString(), _iniFilePath); WritePrivateProfileString("Options", "ShowLog", cbShowLog.Checked.ToString(), _iniFilePath);
                LogToBuffer("INIファイルへの設定保存完了。"); // AppendLog -> LogToBuffer
            }
            catch (Exception ex)
            {
                string errorMsg = $"エラー: INI保存例外: {ex.Message}"; LogToBuffer(errorMsg); Debug.WriteLine(errorMsg); // AppendLog -> LogToBuffer
            }
            finally
            {
                FlushLogBuffer(); // 保存試行後にバッファをフラッシュ
            }
        }
        #endregion

        private void rbEncoder_CheckedChanged(object sender, EventArgs e) => UpdateCustomEncoderTextBoxState();
        private void UpdateCustomEncoderTextBoxState() { try { if (this.IsDisposed) return; customEncoderTextBox.Enabled = rbEncoderCustom.Checked && runButton.Enabled; } catch { /* ignore */ } }
        private void cbShowLog_CheckedChanged(object sender, EventArgs e) => UpdateLogRichTextBoxVisibility();

        // UpdateLogRichTextBoxVisibility メソッド (logLabel への参照を削除済み)
        private void UpdateLogRichTextBoxVisibility()
        {
            try
            {
                if (this.IsDisposed) return;
                bool visible = cbShowLog.Checked;
                logRichTextBox.Visible = visible;
                // logLabel.Visible = visible; // <- この行は削除済み
            }
            catch (Exception ex)
            {
                // エラー処理が必要な場合はここに追加 (例: Debug.WriteLine)
                // 通常、フォーム破棄時の例外は無視して問題ないことが多い
                /* ignore */
                Debug.WriteLine($"Error in UpdateLogRichTextBoxVisibility: {ex.Message}"); // デバッグ用にログ出力
            }
        }


        private void SetControlsState(bool isRunning)
        {
            if (this.InvokeRequired) { this.BeginInvoke((MethodInvoker)delegate { SetControlsStateInternal(isRunning); }); }
            else { SetControlsStateInternal(isRunning); }
        }
        private void SetControlsStateInternal(bool isRunning)
        {
            try
            {
                if (this.IsDisposed) return;
                bool enableControls = !isRunning;
                inputDirTextBox.Enabled = enableControls; inputDirButton.Enabled = enableControls; outputDirTextBox.Enabled = enableControls; outputDirButton.Enabled = enableControls; ffmpegDirTextBox.Enabled = enableControls; ffmpegDirButton.Enabled = enableControls;
                runButton.Enabled = enableControls; compressionGroupBox.Enabled = enableControls; encoderGroupBox.Enabled = enableControls; modeGroupBox.Enabled = enableControls; timeoutNumericUpDown.Enabled = enableControls; // cbDebugMode は modeGroupBox 内なので不要
                stopButton.Enabled = isRunning; cbShowLog.Enabled = true;
                UpdateCustomEncoderTextBoxState();
            }
            catch (ObjectDisposedException) { /* ignore */ }
            catch (Exception ex) { Debug.WriteLine($"SetControlsStateInternal Error: {ex.Message}"); }
        }

        private void inputDirButton_Click(object sender, EventArgs e) => SelectDirectory(inputDirTextBox, "入力元ディレクトリを選択");
        private void outputDirButton_Click(object sender, EventArgs e) => SelectDirectory(outputDirTextBox, "出力先ディレクトリを選択");
        private void ffmpegDirButton_Click(object sender, EventArgs e) => SelectDirectory(ffmpegDirTextBox, "ffmpeg ディレクトリを選択");
        private void SelectDirectory(TextBox targetTextBox, string description) { using (var dialog = new FolderBrowserDialog()) { dialog.Description = description; dialog.ShowNewFolderButton = true; if (!string.IsNullOrWhiteSpace(targetTextBox.Text) && Directory.Exists(targetTextBox.Text)) { dialog.SelectedPath = targetTextBox.Text; } if (dialog.ShowDialog() == DialogResult.OK) { targetTextBox.Text = dialog.SelectedPath; } } }

        private void runButton_Click(object sender, EventArgs e) // ★ async void の警告は残るが動作優先
        {
            // --- 入力検証 ---
            if (!Directory.Exists(inputDirTextBox.Text)) { MessageBox.Show("入力元ディレクトリが存在しません。", "エラー", MessageBoxButtons.OK, MessageBoxIcon.Error); return; }
            if (string.IsNullOrWhiteSpace(outputDirTextBox.Text)) { MessageBox.Show("出力先ディレクトリを指定してください。", "エラー", MessageBoxButtons.OK, MessageBoxIcon.Error); return; }
            if (!Directory.Exists(outputDirTextBox.Text)) { var confirmCreate = MessageBox.Show($"出力先ディレクトリ '{outputDirTextBox.Text}' が存在しません。\n作成しますか？", "確認", MessageBoxButtons.YesNo, MessageBoxIcon.Question); if (confirmCreate == DialogResult.Yes) { try { Directory.CreateDirectory(outputDirTextBox.Text); } catch (Exception ex) { MessageBox.Show($"出力先ディレクトリの作成失敗:\n{ex.Message}", "エラー", MessageBoxButtons.OK, MessageBoxIcon.Error); return; } } else { return; } }
            if (rbEncoderCustom.Checked && string.IsNullOrWhiteSpace(customEncoderTextBox.Text)) { MessageBox.Show("カスタムエンコーダ名を指定してください。", "エラー", MessageBoxButtons.OK, MessageBoxIcon.Error); return; }
            if (!string.IsNullOrWhiteSpace(ffmpegDirTextBox.Text) && !Directory.Exists(ffmpegDirTextBox.Text)) { MessageBox.Show("指定された ffmpeg ディレクトリが存在しません。", "警告", MessageBoxButtons.OK, MessageBoxIcon.Warning); }
            // --- /入力検証 ---

            if (MessageBox.Show("TransAV1 の変換処理を開始しますか？", "実行確認", MessageBoxButtons.YesNo, MessageBoxIcon.Question) == DialogResult.No) return;

            logRichTextBox.Clear();
            lock (_logBufferLock) { _logBuffer.Clear(); } // バッファもクリア
            LogToBuffer("TransAV1 実行準備中..."); // AppendLog -> LogToBuffer
            SetControlsState(true);

            string arguments = GetCommandLineArguments();
            if (arguments == null) { LogToBuffer("エラー: 引数構築失敗"); FlushLogBuffer(); SetControlsState(false); return; } // AppendLog -> LogToBuffer
            LogToBuffer($"実行コマンド: TransAV1_CUI.exe {arguments}"); // AppendLog -> LogToBuffer

            string transAV1CmdPath = Path.Combine(AppDomain.CurrentDomain.BaseDirectory, "TransAV1_CUI.exe");
            if (!File.Exists(transAV1CmdPath)) { LogToBuffer($"エラー: '{transAV1CmdPath}' が見つかりません。"); FlushLogBuffer(); SetControlsState(false); return; } // AppendLog -> LogToBuffer

            ProcessStartInfo startInfo = new ProcessStartInfo { FileName = transAV1CmdPath, Arguments = arguments, UseShellExecute = false, RedirectStandardOutput = true, RedirectStandardError = true, CreateNoWindow = true, StandardOutputEncoding = Encoding.UTF8, StandardErrorEncoding = Encoding.UTF8 };

            lock (_processLock)
            {
                DisposeProcess();
                try
                {
                    _transAV1Process = new Process { StartInfo = startInfo, EnableRaisingEvents = true };
                    _transAV1Process.OutputDataReceived += Process_OutputDataReceived;
                    _transAV1Process.ErrorDataReceived += Process_ErrorDataReceived;
                    _transAV1Process.Exited += Process_Exited;

                    LogToBuffer("TransAV1 プロセスを開始します..."); // AppendLog -> LogToBuffer
                    FlushLogBuffer(); // 開始直前のログを表示
                    _transAV1Process.Start();
                    _transAV1Process.BeginOutputReadLine();
                    _transAV1Process.BeginErrorReadLine();
                }
                catch (Exception ex)
                {
                    LogToBuffer($"エラー: プロセス開始失敗: {ex.Message}"); // AppendLog -> LogToBuffer
                    FlushLogBuffer();
                    DisposeProcess();
                    SetControlsState(false);
                }
            }
        }

        // --- プロセスイベントハンドラ ---
        private void Process_OutputDataReceived(object sender, DataReceivedEventArgs e) => LogToBuffer(e.Data); // AppendLog -> LogToBuffer
        private void Process_ErrorDataReceived(object sender, DataReceivedEventArgs e) => LogToBuffer($"[エラー出力] {e.Data}"); // AppendLog -> LogToBuffer

        private void Process_Exited(object sender, EventArgs e)
        {
            try
            {
                Process processSnapshot = sender as Process;
                int exitCode = -999;
                try { if (processSnapshot != null && processSnapshot.HasExited) { exitCode = processSnapshot.ExitCode; } }
                catch (Exception ex) when (ex is InvalidOperationException || ex is NotSupportedException) { LogToBuffer($"警告: ExitCode取得失敗: {ex.GetType().Name}"); } // AppendLog -> LogToBuffer
                catch (Exception ex) { LogToBuffer($"警告: ExitCode取得中エラー: {ex.Message}"); } // AppendLog -> LogToBuffer

                LogToBuffer($"TransAV1 プロセスが終了しました (終了コード: {exitCode})。"); // AppendLog -> LogToBuffer

                this.BeginInvoke((MethodInvoker)delegate {
                    try
                    {
                        if (this.IsDisposed) return;
                        lock (_processLock)
                        {
                            if (_transAV1Process == processSnapshot) { _transAV1Process = null; }
                        }
                        processSnapshot?.Dispose();
                        SetControlsState(false);
                        FlushLogBuffer(); // 最後のログ表示
                    }
                    catch (Exception uiEx) { Debug.WriteLine($"UI Update Error in Process_Exited callback: {uiEx}"); }
                });
            }
            catch (Exception outerEx)
            {
                LogToBuffer($"重大エラー: Process_Exited 例外: {outerEx.Message}"); Debug.WriteLine($"Critical Error in Process_Exited: {outerEx}"); // AppendLog -> LogToBuffer
                if (!this.IsDisposed && this.IsHandleCreated) { try { this.BeginInvoke((MethodInvoker)delegate { SetControlsState(false); }); } catch { } }
                FlushLogBuffer(); // エラーログ表示試行
            }
        }
        // --- /プロセスイベントハンドラ ---

        // --- ★ ログ出力ヘルパー (バッファリング方式) Start ★ ---
        private void LogToBuffer(string message)
        {
            if (string.IsNullOrEmpty(message)) return;

            lock (_logBufferLock) // StringBuilderへのアクセスをロック
            {
                // 制御文字を除去してからバッファに追加
                string sanitizedMessage = message != null ? _controlCharRegex.Replace(message, "") : "";
                if (!string.IsNullOrEmpty(sanitizedMessage))
                {
                    _logBuffer.AppendLine(sanitizedMessage); // 改行を追加してバッファへ
                }
            }
        }

        private void FlushLogBuffer()
        {
            if (this.IsDisposed || !this.IsHandleCreated || !logRichTextBox.IsHandleCreated) return;

            string logsToAppend;
            lock (_logBufferLock)
            {
                if (_logBuffer.Length == 0) return;
                logsToAppend = _logBuffer.ToString();
                _logBuffer.Clear();
            }

            // UI スレッドで RichTextBox を更新
            if (logRichTextBox.InvokeRequired)
            {
                logRichTextBox.BeginInvoke((MethodInvoker)delegate { AppendLogInternal(logsToAppend); });
            }
            else
            {
                AppendLogInternal(logsToAppend);
            }
        }

        private void AppendLogInternal(string messageChunk) // 内部的な追加処理 (改行は含まれている前提)
        {
            try
            {
                if (logRichTextBox.IsDisposed) return;
                // ここでは制御文字除去は不要 (LogToBufferで実施済み)
                if (string.IsNullOrEmpty(messageChunk)) return;

                const int MAX_LOG_LENGTH = 60000; const int TRUNCATE_KEEP_LENGTH = 40000;

                if (logRichTextBox.TextLength + messageChunk.Length > MAX_LOG_LENGTH)
                {
                    int overflow = (logRichTextBox.TextLength + messageChunk.Length) - MAX_LOG_LENGTH;
                    int removeLength = Math.Max(overflow, logRichTextBox.TextLength - TRUNCATE_KEEP_LENGTH);
                    if (removeLength > 0 && removeLength < logRichTextBox.TextLength)
                    {
                        logRichTextBox.Select(0, removeLength);
                        logRichTextBox.SelectedText = "";
                    }
                    else if (removeLength >= logRichTextBox.TextLength)
                    {
                        logRichTextBox.Clear();
                    }
                }

                logRichTextBox.AppendText(messageChunk); // バッファの内容を追加 (改行は LogToBuffer で付与済み)
                // logRichTextBox.ScrollToCaret(); // コメントアウト継続
            }
            catch (ObjectDisposedException) { /* ignore */ }
            catch (Exception ex) { Debug.WriteLine($"AppendLogInternal Error: {ex.Message}"); }
        }
        // --- ★ ログ出力ヘルパー (バッファリング方式) End ★ ---

        private string GetCommandLineArguments()
        {
            StringBuilder args = new StringBuilder();
            args.Append($"-s \"{inputDirTextBox.Text}\" "); args.Append($"-o \"{outputDirTextBox.Text}\" "); if (!string.IsNullOrWhiteSpace(ffmpegDirTextBox.Text)) args.Append($"-ffmpegdir \"{ffmpegDirTextBox.Text}\" ");
            if (rbCompressionSize.Checked) args.Append("-hwopt \"-cq 30 -preset p6\" -cpuopt \"-crf 30 -preset 8\" "); else if (rbCompressionStandard.Checked) args.Append("-hwopt \"-cq 25 -preset p5\" -cpuopt \"-crf 28 -preset 7\" "); else if (rbCompressionQuality.Checked) args.Append("-hwopt \"-cq 20 -preset p4\" -cpuopt \"-crf 25 -preset 6\" ");
            if (rbEncoderNVENC.Checked) args.Append("-hwenc \"av1_nvenc\" "); else if (rbEncoderCPU.Checked) args.Append("-cpuenc \"libsvtav1\" "); else if (rbEncoderCustom.Checked && !string.IsNullOrWhiteSpace(customEncoderTextBox.Text)) args.Append($"-hwenc \"{customEncoderTextBox.Text}\" ");
            if (rbModeRestart.Checked) args.Append("-restart "); if (rbModeForceStart.Checked) args.Append("-force "); if (cbQuickMode.Checked) args.Append("-quick ");
            args.Append($"-timeout {timeoutNumericUpDown.Value} "); if (cbDebugMode.Checked) args.Append("-debug "); return args.ToString().TrimEnd();
        }

        private void stopButton_Click(object sender, EventArgs e)
        {
            Process processToStop = null;
            lock (_processLock) { processToStop = _transAV1Process; }

            if (processToStop == null || processToStop.HasExited)
            {
                LogToBuffer("TransAV1 プロセスは実行されていません。"); FlushLogBuffer(); SetControlsState(false); return; // AppendLog -> LogToBuffer
            }

            if (MessageBox.Show("TransAV1 処理を強制停止しますか？\n(ファイルの破損等の可能性があります)", "強制停止確認", MessageBoxButtons.YesNo, MessageBoxIcon.Warning) == DialogResult.No) return;

            LogToBuffer("プロセス停止を試みます..."); FlushLogBuffer(); // AppendLog -> LogToBuffer
            try { if (!this.IsDisposed) rbModeRestart.Checked = true; } catch { /* ignore */ } // 再開チェック

            Task.Run(async () => // Kill処理はバックグラウンドで
            {
                Process currentProcess = null;
                lock (_processLock) { currentProcess = _transAV1Process; }

                if (currentProcess == null || currentProcess.HasExited || currentProcess != processToStop)
                { LogToBuffer("停止タスク開始時にはプロセスは既に終了または変更されていました。"); FlushLogBuffer(); return; } // AppendLog -> LogToBuffer

                try
                {
                    // ★★★ 修正点: 待機時間を 10 秒に変更 ★★★
                    LogToBuffer("プロセスの終了を最大 10 秒間待機します..."); FlushLogBuffer(); // AppendLog -> LogToBuffer
                    await Task.Delay(10000); // 1000ms -> 10000ms

                    lock (_processLock) { currentProcess = _transAV1Process; }
                    if (currentProcess != null && !currentProcess.HasExited && currentProcess == processToStop)
                    {
                        try
                        {
                            LogToBuffer("プロセスが応答しないため、強制終了 (Kill) します。"); FlushLogBuffer(); // AppendLog -> LogToBuffer
                            currentProcess.Kill();
                            LogToBuffer("Kill() を呼び出しました。Exitedイベントで終了を検知します。"); // AppendLog -> LogToBuffer
                        }
                        catch (InvalidOperationException) { LogToBuffer("Kill試行時にプロセスは既に終了していました。"); } // AppendLog -> LogToBuffer
                        catch (Exception killEx) { LogToBuffer($"エラー: Kill中に例外: {killEx.Message}"); } // AppendLog -> LogToBuffer
                    }
                    else { LogToBuffer("待機中にプロセスが終了したか、参照が変わりました。"); } // AppendLog -> LogToBuffer
                }
                catch (Exception taskEx) { LogToBuffer($"重大エラー: 停止タスク例外: {taskEx.Message}"); Debug.WriteLine($"Critical Error in Stop Button Task: {taskEx}"); if (!this.IsDisposed && this.IsHandleCreated) { try { this.BeginInvoke((MethodInvoker)delegate { SetControlsState(false); }); } catch { } } } // AppendLog -> LogToBuffer
                finally { FlushLogBuffer(); } // タスク終了時にログ表示
            });
        }

        private void Form1_FormClosing(object sender, FormClosingEventArgs e)
        {
            LogToBuffer("フォームクロージング処理開始..."); FlushLogBuffer(); // AppendLog -> LogToBuffer

            // ★ タイマー停止・破棄
            if (_logUpdateTimer != null)
            {
                _logUpdateTimer.Stop();
                _logUpdateTimer.Dispose();
                _logUpdateTimer = null;
                LogToBuffer("ログ更新タイマーを停止・破棄しました。"); // AppendLog -> LogToBuffer
            }

            Process processSnapshot = null;
            lock (_processLock) { processSnapshot = _transAV1Process; _transAV1Process = null; }

            if (processSnapshot != null)
            {
                bool processHasExited = false;
                try { processHasExited = processSnapshot.HasExited; } catch { /* ignore */ }

                if (!processHasExited)
                {
                    LogToBuffer("実行中のプロセスがあります。終了確認を行います。"); FlushLogBuffer(); // AppendLog -> LogToBuffer
                    DialogResult confirmResult = MessageBox.Show("TransAV1 処理が実行中です。\nフォームを閉じると処理は中断されます。\n閉じてもよろしいですか？", "終了確認", MessageBoxButtons.YesNo, MessageBoxIcon.Warning);

                    if (confirmResult == DialogResult.No)
                    {
                        LogToBuffer("フォームクローズはキャンセルされました。"); FlushLogBuffer(); // AppendLog -> LogToBuffer
                        e.Cancel = true;
                        lock (_processLock) { _transAV1Process = processSnapshot; } // プロセス参照を戻す
                        InitializeLogUpdateTimer(); // ★ タイマー再開
                        return;
                    }
                    else
                    {
                        LogToBuffer("ユーザーがフォームクローズを承認。プロセスを強制停止します..."); FlushLogBuffer(); // AppendLog -> LogToBuffer
                        try
                        {
                            if (!processSnapshot.HasExited) { processSnapshot.Kill(); LogToBuffer("終了時にプロセスを強制終了 (Kill) しました。"); } // AppendLog -> LogToBuffer
                        }
                        catch (Exception ex) { string errorMsg = $"エラー: 終了時Kill例外: {ex.Message}"; LogToBuffer(errorMsg); Debug.WriteLine(errorMsg); } // AppendLog -> LogToBuffer
                    }
                }
                try
                {
                    processSnapshot.OutputDataReceived -= Process_OutputDataReceived;
                    processSnapshot.ErrorDataReceived -= Process_ErrorDataReceived;
                    processSnapshot.Exited -= Process_Exited;
                    processSnapshot.Dispose();
                    LogToBuffer("終了時のプロセスオブジェクトを破棄しました。"); // AppendLog -> LogToBuffer
                }
                catch (Exception ex) { LogToBuffer($"終了時のプロセス破棄中にエラー: {ex.Message}"); } // AppendLog -> LogToBuffer
                FlushLogBuffer();
            }

            LogToBuffer("フォームクローズ処理: INIファイル保存を呼び出します。"); FlushLogBuffer(); // AppendLog -> LogToBuffer
            SaveSettingsToIni();
            LogToBuffer("フォームクロージング処理完了。"); FlushLogBuffer(); // AppendLog -> LogToBuffer
        }

        private void DisposeProcess()
        {
            Process processToDispose = null;
            lock (_processLock) { processToDispose = _transAV1Process; _transAV1Process = null; }

            if (processToDispose != null)
            {
                LogToBuffer("古いプロセスオブジェクトの破棄を開始します..."); // AppendLog -> LogToBuffer
                try
                {
                    processToDispose.OutputDataReceived -= Process_OutputDataReceived;
                    processToDispose.ErrorDataReceived -= Process_ErrorDataReceived;
                    processToDispose.Exited -= Process_Exited;
                    bool needsKill = false;
                    try { needsKill = !processToDispose.HasExited; } catch { /* ignore */ }
                    if (needsKill) { try { processToDispose.Kill(); LogToBuffer("破棄時に古いプロセスをKillしました。"); } catch { /* 無視 */ } } // AppendLog -> LogToBuffer
                    processToDispose.Dispose();
                    LogToBuffer("古いプロセスオブジェクトを破棄しました。"); // AppendLog -> LogToBuffer
                }
                catch (Exception ex) { LogToBuffer($"古いプロセスオブジェクトの破棄中にエラー: {ex.Message}"); } // AppendLog -> LogToBuffer
                FlushLogBuffer();
            }
        }
    }
}
