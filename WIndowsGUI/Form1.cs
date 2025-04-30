using System;
using System.Diagnostics; // Process クラス用
using System.IO; // Directory.Exists, File.Exists 用
using System.Text; // StringBuilder 用
using System.Threading.Tasks; // 非同期処理用
using System.Windows.Forms; // Windows Forms 用
using System.Runtime.InteropServices; // P/Invoke 用 (Ctrl-C送信に使用)

namespace TransAV1
{
    public partial class Form1 : Form // partial class は Designer.cs と連携するために必要
    {
        // 実行中のTransAV1 CUIプロセスを保持するフィールド
        private Process _transAV1Process;

        // Ctrl-C シグナルを送信するための Windows API (GenerateConsoleCtrlEvent)
        // CUIプロセスに Ctrl+C シグナルを送信するために必要
        [DllImport("kernel32.dll", SetLastError = true)]
        private static extern bool GenerateConsoleCtrlEvent(uint dwCtrlEvent, uint dwProcessGroupId);

        [DllImport("kernel32.dll", SetLastError = true)]
        private static extern bool AttachConsole(uint dwProcessId);

        [DllImport("kernel32.dll", SetLastError = true)]
        private static extern bool FreeConsole();

        // Ctrl-C イベントタイプ
        private const uint CTRL_C_EVENT = 0;


        public Form1()
        {
            // Designer.cs で定義された UI 要素を初期化
            InitializeComponent();

            // コントロールの初期状態を設定
            SetControlsState(false); // 起動時は強制停止ボタンを無効にする
            customEncoderTextBox.Enabled = false; // 起動時はカスタムエンコーダ入力欄を無効にする
        }

        // フォームロード時の処理 (必要に応じて初期設定などを追加)
        private void Form1_Load(object sender, EventArgs e)
        {
            // 例: 設定ファイルから前回のディレクトリパスを読み込むなど
            // inputDirTextBox.Text = Properties.Settings.Default.LastInputDir;
            // outputDirTextBox.Text = Properties.Settings.Default.LastOutputDir;
            // ffmpegDirTextBox.Text = Properties.Settings.Default.LastFfmpegDir;

            // ラジオボタンの CheckedChanged イベントハンドラを設定
            // これにより、カスタムエンコーダ選択時にテキストボックスを有効/無効にする
            rbEncoderNVENC.CheckedChanged += rbEncoder_CheckedChanged;
            rbEncoderCPU.CheckedChanged += rbEncoder_CheckedChanged;
            rbEncoderCustom.CheckedChanged += rbEncoder_CheckedChanged;

            // 初期状態でカスタムエンコーダ入力欄の状態を設定
            UpdateCustomEncoderTextBoxState();

            // ログ表示チェックボックスのイベントハンドラを設定
            cbShowLog.CheckedChanged += cbShowLog_CheckedChanged;
            // 初期状態でログ表示エリアの状態を設定
            UpdateLogRichTextBoxVisibility();

            // 各ボタンのクリックイベントハンドラを関連付け
            inputDirButton.Click += inputDirButton_Click;
            outputDirButton.Click += outputDirButton_Click;
            ffmpegDirButton.Click += ffmpegDirButton_Click;
            runButton.Click += runButton_Click;
            stopButton.Click += stopButton_Click;

            // フォームが閉じられるときのイベントハンドラを関連付け
            this.FormClosing += Form1_FormClosing;
        }

        // ラジオボタン選択状態変更時の処理
        private void rbEncoder_CheckedChanged(object sender, EventArgs e)
        {
            UpdateCustomEncoderTextBoxState();
        }

        // カスタムエンコーダ入力欄の有効/無効を更新する
        private void UpdateCustomEncoderTextBoxState()
        {
            customEncoderTextBox.Enabled = rbEncoderCustom.Checked;
        }

        // ログ表示チェックボックスの状態変更時の処理
        private void cbShowLog_CheckedChanged(object sender, EventArgs e)
        {
            UpdateLogRichTextBoxVisibility();
        }

        // ログ表示エリアの表示/非表示を更新する
        private void UpdateLogRichTextBoxVisibility()
        {
            logRichTextBox.Visible = cbShowLog.Checked;
            logLabel.Visible = cbShowLog.Checked; // ラベルも一緒に表示/非表示
        }


        // コントロールの有効/無効状態を設定するヘルパーメソッド
        private void SetControlsState(bool isRunning)
        {
            // 実行中はディレクトリ選択、オプション設定、実行ボタンを無効にする
            inputDirTextBox.Enabled = !isRunning;
            inputDirButton.Enabled = !isRunning;
            outputDirTextBox.Enabled = !isRunning;
            outputDirButton.Enabled = !isRunning;
            ffmpegDirTextBox.Enabled = !isRunning;
            ffmpegDirButton.Enabled = !isRunning;
            runButton.Enabled = !isRunning;

            compressionGroupBox.Enabled = !isRunning;
            encoderGroupBox.Enabled = !isRunning;
            modeGroupBox.Enabled = !isRunning;
            timeoutNumericUpDown.Enabled = !isRunning;
            cbDebugMode.Enabled = !isRunning;

            // カスタムエンコーダ入力欄は、isRunningに関わらずラジオボタンの状態に依存
            if (!isRunning)
            {
                UpdateCustomEncoderTextBoxState(); // 実行終了時はラジオボタンの状態に合わせて更新
            }
            else
            {
                customEncoderTextBox.Enabled = false; // 実行中は強制的に無効
            }


            // 実行中は強制停止ボタンを有効にする
            stopButton.Enabled = isRunning;

            // ログ表示チェックボックスは常に有効にしておくか、実行中は無効にするか検討
            // ここでは実行中も有効にしておく
            cbShowLog.Enabled = true;
        }


        // ディレクトリ選択ボタンのクリックイベントハンドラ (入力元)
        private void inputDirButton_Click(object sender, EventArgs e)
        {
            using (var dialog = new FolderBrowserDialog())
            {
                dialog.Description = "入力元ディレクトリを選択してください";
                dialog.ShowNewFolderButton = true;
                dialog.SelectedPath = inputDirTextBox.Text;

                if (dialog.ShowDialog() == DialogResult.OK)
                {
                    inputDirTextBox.Text = dialog.SelectedPath;
                }
            }
        }

        // ディレクトリ選択ボタンのクリックイベントハンドラ (出力先)
        private void outputDirButton_Click(object sender, EventArgs e)
        {
            using (var dialog = new FolderBrowserDialog())
            {
                dialog.Description = "出力先ディレクトリを選択してください";
                dialog.ShowNewFolderButton = true;
                dialog.SelectedPath = outputDirTextBox.Text;

                if (dialog.ShowDialog() == DialogResult.OK)
                {
                    outputDirTextBox.Text = dialog.SelectedPath;
                }
            }
        }

        // ディレクトリ選択ボタンのクリックイベントハンドラ (ffmpeg ディレクトリ)
        private void ffmpegDirButton_Click(object sender, EventArgs e)
        {
            using (var dialog = new FolderBrowserDialog())
            {
                dialog.Description = "ffmpeg が含まれるディレクトリを選択してください";
                dialog.ShowNewFolderButton = true;
                dialog.SelectedPath = ffmpegDirTextBox.Text;

                if (dialog.ShowDialog() == DialogResult.OK)
                {
                    ffmpegDirTextBox.Text = dialog.SelectedPath;
                }
            }
        }


        // TransAV1 実行ボタンのクリックイベントハンドラ
        private async void runButton_Click(object sender, EventArgs e) // async を付けて非同期処理を可能にする
        {
            // 入力値の基本的な検証
            if (string.IsNullOrWhiteSpace(inputDirTextBox.Text))
            {
                MessageBox.Show("入力元ディレクトリを指定してください。", "エラー", MessageBoxButtons.OK, MessageBoxIcon.Error);
                return;
            }
            if (string.IsNullOrWhiteSpace(outputDirTextBox.Text))
            {
                MessageBox.Show("出力先ディレクトリを指定してください。", "エラー", MessageBoxButtons.OK, MessageBoxIcon.Error);
                return;
            }
            if (rbEncoderCustom.Checked && string.IsNullOrWhiteSpace(customEncoderTextBox.Text))
            {
                MessageBox.Show("カスタムエンコーダ名を指定してください。", "エラー", MessageBoxButtons.OK, MessageBoxIcon.Error);
                return;
            }


            // 実行確認ダイアログを表示
            DialogResult confirmResult = MessageBox.Show(
                "TransAV1 の変換処理を開始しますか？",
                "実行確認",
                MessageBoxButtons.YesNo,
                MessageBoxIcon.Question
            );

            if (confirmResult == DialogResult.No)
            {
                // 「いいえ」が選択されたら処理を中断
                return;
            }

            // ログエリアをクリア
            logRichTextBox.Clear();
            logRichTextBox.AppendText("TransAV1 実行中...\r\n");

            // コントロールの状態を実行中に設定
            SetControlsState(true);

            // コマンドライン引数を構築
            string arguments = GetCommandLineArguments();

            if (arguments == null) // 引数構築中にエラーがあった場合
            {
                SetControlsState(false); // 状態を戻す
                return;
            }


            // TransAV1 CUIの実行ファイルパス
            // GUI実行ファイルと同じディレクトリにあると仮定
            string transAV1Cmd = "TransAV1_CUI.exe";

            // TransAV1 CUI実行ファイルの存在チェック
            if (!File.Exists(transAV1Cmd))
            {
                logRichTextBox.AppendText($"エラー: '{transAV1Cmd}' が見つかりません。\r\n");
                SetControlsState(false); // 状態を戻す
                return;
            }


            // ProcessStartInfo の設定
            ProcessStartInfo startInfo = new ProcessStartInfo
            {
                FileName = transAV1Cmd,
                Arguments = arguments,
                UseShellExecute = false,      // シェルを使わない (出力をリダイレクトするため必須)
                RedirectStandardOutput = true, // 標準出力をリダイレクト
                RedirectStandardError = true,  // 標準エラー出力をリダイレクト
                CreateNoWindow = true         // 新しいコンソールウィンドウを作成しない
            };

            // プロセスオブジェクトを作成
            _transAV1Process = new Process { StartInfo = startInfo };

            // 標準出力と標準エラー出力のイベントハンドラを設定
            _transAV1Process.OutputDataReceived += (s, args) =>
            {
                if (!string.IsNullOrEmpty(args.Data))
                {
                    // UIスレッド以外からのUI更新はInvokeが必要
                    logRichTextBox.Invoke((MethodInvoker)delegate
                    {
                        logRichTextBox.AppendText(args.Data + "\r\n");
                    });
                }
            };

            _transAV1Process.ErrorDataReceived += (s, args) =>
            {
                if (!string.IsNullOrEmpty(args.Data))
                {
                    // UIスレッド以外からのUI更新はInvokeが必要
                    logRichTextBox.Invoke((MethodInvoker)delegate
                    {
                        logRichTextBox.AppendText("エラー出力: " + args.Data + "\r\n"); // エラー出力は区別するなど
                    });
                }
            };

            // プロセス終了イベントハンドラを設定
            _transAV1Process.EnableRaisingEvents = true; // Exited イベントを有効にする
            _transAV1Process.Exited += (s, args) =>
            {
                // UIスレッド以外からのUI更新はInvokeが必要
                logRichTextBox.Invoke((MethodInvoker)delegate
                {
                    logRichTextBox.AppendText($"TransAV1 終了 (終了コード: {_transAV1Process.ExitCode}).\r\n");
                    _transAV1Process = null; // プロセス参照をクリア
                    SetControlsState(false); // 状態を戻す
                });
            };


            try
            {
                // プロセスを開始
                _transAV1Process.Start();

                // 標準出力と標準エラー出力の読み取りを開始 (非同期)
                _transAV1Process.BeginOutputReadLine();
                _transAV1Process.BeginErrorReadLine();

                // プロセス終了を待機 (UIスレッドをブロックしないように async/await を使用)
                // .NET Framework の場合、WaitForExitAsync はないので Task.Run を使う
                await Task.Run(() => _transAV1Process.WaitForExit());

                // Exited イベントハンドラで終了メッセージや状態更新は処理されるため、ここでは特別な処理は不要

            }
            catch (Exception ex)
            {
                // プロセス開始失敗などの例外をキャッチ
                logRichTextBox.Invoke((MethodInvoker)delegate
                {
                    logRichTextBox.AppendText($"エラー: プロセス開始または実行中に例外が発生しました: {ex.Message}\r\n");
                    _transAV1Process = null; // プロセス参照をクリア
                    SetControlsState(false); // 状態を戻す
                });
            }
        }

        // コマンドライン引数を構築するヘルパーメソッド
        private string GetCommandLineArguments()
        {
            StringBuilder args = new StringBuilder();

            // 入力元ディレクトリ (必須) - runButton_Click でチェック済み
            args.Append($"-s \"{inputDirTextBox.Text}\" ");

            // 出力先ディレクトリ (必須) - runButton_Click でチェック済み
            args.Append($"-o \"{outputDirTextBox.Text}\" ");

            // ffmpeg ディレクトリ (オプション)
            if (!string.IsNullOrWhiteSpace(ffmpegDirTextBox.Text))
            {
                args.Append($"-ffmpegdir \"{ffmpegDirTextBox.Text}\" ");
            }

            // 圧縮オプション
            if (rbCompressionSize.Checked)
            {
                // 圧縮重視のオプションを追加 (例: CRF高め, Preset遅め)
                // Go CUIのデフォルトを参考に調整
                args.Append("-hwopt \"-cq 30 -preset p6\" -cpuopt \"-crf 30 -preset 8\" ");
            }
            else if (rbCompressionStandard.Checked)
            {
                // 標準のオプションを追加 (Go CUIのデフォルト)
                args.Append("-hwopt \"-cq 25 -preset p5\" -cpuopt \"-crf 28 -preset 7\" ");
            }
            else if (rbCompressionQuality.Checked)
            {
                // 画質重視のオプションを追加 (例: CRF低め, Preset遅め)
                // Go CUIのデフォルトを参考に調整
                args.Append("-hwopt \"-cq 20 -preset p4\" -cpuopt \"-crf 25 -preset 6\" ");
            }

            // エンコーダ選択
            if (rbEncoderNVENC.Checked)
            {
                args.Append("-hwenc \"av1_nvenc\" ");
                // CPUエンコーダはフォールバックとして Go CUI に任せるか、ここで指定するか
                // Go CUI のロジックに合わせて、HW指定時は CPU は指定しないでおく
            }
            else if (rbEncoderCPU.Checked)
            {
                args.Append("-cpuenc \"libsvtav1\" ");
                // HWエンコーダは指定しない
            }
            else if (rbEncoderCustom.Checked)
            {
                // カスタムエンコーダ名が HW なのか CPU なのか判断できないため、
                // ここではシンプルに -hwenc と -cpuenc の両方に同じカスタム名を渡す。
                // Go CUI のロジックに合わせて調整が必要かもしれません。
                // 例: args.Append($"-hwenc \"{customEncoderTextBox.Text}\" -cpuenc \"{customEncoderTextBox.Text}\" ");
                // あるいは、カスタムは -hwenc のみとして、CPUフォールバックはデフォルトにする (前回のコードの仮定)。
                // ここでは前回の仮定を踏襲し、カスタムは -hwenc のみとする。
                if (!string.IsNullOrWhiteSpace(customEncoderTextBox.Text)) // runButton_Click でチェック済み
                {
                    args.Append($"-hwenc \"{customEncoderTextBox.Text}\" ");
                    // CPUフォールバックは Go CUI のデフォルト (-cpuenc "libsvtav1") に任せる
                }
            }
            // どちらも選択されていない場合は Go CUI のデフォルトに任せる

            // 動作モード
            if (cbRestart.Checked)
            {
                args.Append("-restart ");
            }
            if (cbForceStart.Checked)
            {
                args.Append("-force ");
                // -force は対話的な確認が必要なので、GUIでは注意が必要。
                // CUI側で確認プロンプトが出ると、GUI側でそれを処理する必要がある。
                // CUI側で -force が指定されたら確認なしで強制実行される仕様なら問題ない。
                // TransAV1 CUIの -force オプションの正確な動作を確認してください。
                // もし CUI 側で確認プロンプトが出る場合、GUI側でそれを自動応答するか、
                // -force オプションを使わない代替手段を検討する必要があります。
                // ここでは CUI 側で確認なしで強制実行されると仮定します。
            }
            if (cbQuickMode.Checked)
            {
                args.Append("-quick ");
            }

            // タイムアウト
            args.Append($"-timeout {timeoutNumericUpDown.Value} ");

            // デバッグモード
            if (cbDebugMode.Checked)
            {
                args.Append("-debug ");
            }

            // 末尾のスペースを削除して返す
            return args.ToString().TrimEnd();
        }


        // 強制停止ボタンのクリックイベントハンドラ
        private void stopButton_Click(object sender, EventArgs e)
        {
            if (_transAV1Process != null && !_transAV1Process.HasExited)
            {
                // 実行中のプロセスがある場合
                DialogResult confirmResult = MessageBox.Show(
                    "TransAV1 処理を強制停止しますか？\r\n(ファイルの破損や不完全な出力が発生する可能性があります)",
                    "強制停止確認",
                    MessageBoxButtons.YesNo,
                    MessageBoxIcon.Warning
                );

                if (confirmResult == DialogResult.Yes)
                {
                    // 「はい」が選択されたらプロセスを終了させる

                    // Ctrl-C シグナルを送信して終了を試みる
                    // これが最もクリーンな終了方法に近い (CUIがシグナルを捕捉できる場合)
                    // AttachConsole は、対象プロセスがコンソールを持っている場合にのみ成功します。
                    // CreateNoWindow = true で起動した場合、コンソールを持たない可能性があります。
                    // その場合、AttachConsole は失敗します。
                    // より確実に終了させるには Kill() が確実ですが、クリーンアップは行われません。
                    // ここではまず AttachConsole & Ctrl+C を試み、失敗したら Kill() にフォールバックします。

                    bool attached = false;
                    try
                    {
                        // プロセスIDでコンソールをアタッチ
                        attached = AttachConsole((uint)_transAV1Process.Id);

                        if (attached)
                        {
                            // Ctrl-C シグナルを送信
                            GenerateConsoleCtrlEvent(CTRL_C_EVENT, 0);
                        }
                    }
                    finally
                    {
                        if (attached)
                        {
                            FreeConsole(); // アタッチしたコンソールを解放
                        }
                    }

                    // プロセスが終了するのを少し待つ
                    _transAV1Process.WaitForExit(5000); // 5秒待つ

                    // Ctrl-C で終了しなかった場合、または AttachConsole が失敗した場合、強制終了
                    if (!_transAV1Process.HasExited)
                    {
                        try
                        {
                            _transAV1Process.Kill(); // プロセスを強制終了
                            logRichTextBox.AppendText("TransAV1 プロセスを強制終了しました。\r\n");
                        }
                        catch (Exception ex)
                        {
                            // プロセスが既に終了している場合など、Kill() が例外を投げることがある
                            logRichTextBox.AppendText($"警告: プロセス強制終了中に例外が発生しました: {ex.Message}\r\n");
                        }
                    }
                    // Exited イベントハンドラで状態は更新される
                }
            }
            else
            {
                logRichTextBox.AppendText("TransAV1 プロセスは実行されていません。\r\n");
            }
        }

        // フォームが閉じられるときの処理 (実行中のプロセスがあれば終了させる)
        private void Form1_FormClosing(object sender, FormClosingEventArgs e)
        {
            if (_transAV1Process != null && !_transAV1Process.HasExited)
            {
                DialogResult confirmResult = MessageBox.Show(
                   "TransAV1 処理が実行中です。\r\nフォームを閉じると処理は中断されます。\r\n閉じてもよろしいですか？",
                   "終了確認",
                   MessageBoxButtons.YesNo,
                   MessageBoxIcon.Warning
               );

                if (confirmResult == DialogResult.No)
                {
                    e.Cancel = true; // フォームを閉じるのをキャンセル
                }
                else
                {
                    // 強制停止処理を実行 (UIスレッドではない可能性もあるため、Invokeは不要だがログ出力はできない)
                    // ここではシンプルに Kill() を使用
                    try
                    {
                        if (AttachConsole((uint)_transAV1Process.Id))
                        {
                            GenerateConsoleCtrlEvent(CTRL_C_EVENT, 0);
                            FreeConsole();
                            _transAV1Process.WaitForExit(5000); // 5秒待つ
                        }
                        if (!_transAV1Process.HasExited)
                        {
                            _transAV1Process.Kill();
                        }
                    }
                    catch (Exception ex)
                    {
                        // 終了処理中のエラーはログに出せない可能性が高い
                        System.Diagnostics.Debug.WriteLine($"終了処理中のエラー: {ex.Message}");
                    }
                }
            }
        }
    }
}
