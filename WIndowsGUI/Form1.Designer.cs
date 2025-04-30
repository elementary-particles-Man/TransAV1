namespace TransAV1
{
    partial class Form1
    {
        /// <summary>
        /// 必要なデザイナー変数です。
        /// </summary>
        private System.ComponentModel.IContainer components = null;

        /// <summary>
        /// 使用中のリソースをすべてクリーンアップします。
        /// </summary>
        /// <param name="disposing">マネージド リソースを破棄する場合は true を指定し、その他の場合は false を指定します。</param>
        protected override void Dispose(bool disposing)
        {
            if (disposing && (components != null))
            {
                components.Dispose();
            }
            base.Dispose(disposing);
        }

        #region Windows フォーム デザイナーで生成されたコード

        /// <summary>
        /// デザイナー サポートに必要なメソッドです。このメソッドの内容を
        /// コード エディターで変更しないでください。
        /// </summary>
        private void InitializeComponent()
        {
            // コントロールの作成

            // ディレクトリ/実行関連
            this.inputDirTextBox = new System.Windows.Forms.TextBox();
            this.inputDirButton = new System.Windows.Forms.Button();
            this.outputDirTextBox = new System.Windows.Forms.TextBox();
            this.outputDirButton = new System.Windows.Forms.Button();
            this.runButton = new System.Windows.Forms.Button();
            this.logRichTextBox = new System.Windows.Forms.RichTextBox();
            this.inputDirLabel = new System.Windows.Forms.Label();
            this.outputDirLabel = new System.Windows.Forms.Label();
            this.logLabel = new System.Windows.Forms.Label();

            // 新規追加コントロール

            // 圧縮オプション グループボックスとラジオボタン
            this.compressionGroupBox = new System.Windows.Forms.GroupBox();
            this.rbCompressionQuality = new System.Windows.Forms.RadioButton();
            this.rbCompressionStandard = new System.Windows.Forms.RadioButton();
            this.rbCompressionSize = new System.Windows.Forms.RadioButton();

            // 規定エンコーダ選択 グループボックスとラジオボタン、カスタム入力欄
            this.encoderGroupBox = new System.Windows.Forms.GroupBox();
            this.customEncoderTextBox = new System.Windows.Forms.TextBox();
            this.rbEncoderCustom = new System.Windows.Forms.RadioButton();
            this.rbEncoderCPU = new System.Windows.Forms.RadioButton();
            this.rbEncoderNVENC = new System.Windows.Forms.RadioButton();

            // 動作モード グループボックスとチェックボックス
            this.modeGroupBox = new System.Windows.Forms.GroupBox();
            this.cbQuickMode = new System.Windows.Forms.CheckBox();
            this.cbForceStart = new System.Windows.Forms.CheckBox();
            this.cbRestart = new System.Windows.Forms.CheckBox();

            // ffmpeg ディレクトリ指定関連
            this.ffmpegDirLabel = new System.Windows.Forms.Label(); // label1 をリネーム
            this.ffmpegDirTextBox = new System.Windows.Forms.TextBox(); // textBox1 をリネーム
            this.ffmpegDirButton = new System.Windows.Forms.Button(); // button1 をリネーム

            // 強制停止ボタン
            this.stopButton = new System.Windows.Forms.Button(); // button2 をリネーム

            // ログ表示切り替えチェックボックス
            this.cbShowLog = new System.Windows.Forms.CheckBox();

            // タイムアウト設定
            this.timeoutLabel = new System.Windows.Forms.Label();
            this.timeoutNumericUpDown = new System.Windows.Forms.NumericUpDown();

            // デバッグモードチェックボックス
            this.cbDebugMode = new System.Windows.Forms.CheckBox();


            this.compressionGroupBox.SuspendLayout();
            this.encoderGroupBox.SuspendLayout();
            this.modeGroupBox.SuspendLayout();
            ((System.ComponentModel.ISupportInitialize)(this.timeoutNumericUpDown)).BeginInit(); // NumericUpDownの初期化
            this.SuspendLayout(); // レイアウト一時停止

            // --- コントロールの設定と配置 ---

            // inputDirLabel
            this.inputDirLabel.AutoSize = true;
            this.inputDirLabel.Location = new System.Drawing.Point(20, 20);
            this.inputDirLabel.Name = "inputDirLabel";
            this.inputDirLabel.Size = new System.Drawing.Size(141, 18); // サイズ調整
            this.inputDirLabel.Text = "入力元ディレクトリ:";

            // inputDirTextBox
            this.inputDirTextBox.Location = new System.Drawing.Point(167, 15); // 位置調整
            this.inputDirTextBox.Name = "inputDirTextBox";
            this.inputDirTextBox.Size = new System.Drawing.Size(500, 25); // サイズ調整
            this.inputDirTextBox.TabIndex = 0;

            // inputDirButton
            this.inputDirButton.Location = new System.Drawing.Point(673, 15); // 位置調整
            this.inputDirButton.Name = "inputDirButton";
            this.inputDirButton.Size = new System.Drawing.Size(75, 25); // サイズ調整
            this.inputDirButton.TabIndex = 1;
            this.inputDirButton.Text = "参照...";
            this.inputDirButton.UseVisualStyleBackColor = true; // デフォルトスタイル

            // outputDirLabel
            this.outputDirLabel.AutoSize = true;
            this.outputDirLabel.Location = new System.Drawing.Point(20, 55); // 位置調整
            this.outputDirLabel.Name = "outputDirLabel";
            this.outputDirLabel.Size = new System.Drawing.Size(141, 18); // サイズ調整
            this.outputDirLabel.Text = "出力先ディレクトリ:";

            // outputDirTextBox
            this.outputDirTextBox.Location = new System.Drawing.Point(167, 50); // 位置調整
            this.outputDirTextBox.Name = "outputDirTextBox";
            this.outputDirTextBox.Size = new System.Drawing.Size(500, 25); // サイズ調整
            this.outputDirTextBox.TabIndex = 2;

            // outputDirButton
            this.outputDirButton.Location = new System.Drawing.Point(673, 50); // 位置調整
            this.outputDirButton.Name = "outputDirButton";
            this.outputDirButton.Size = new System.Drawing.Size(75, 25); // サイズ調整
            this.outputDirButton.TabIndex = 3;
            this.outputDirButton.Text = "参照...";
            this.outputDirButton.UseVisualStyleBackColor = true;

            // ffmpegDirLabel (label1 からリネーム)
            this.ffmpegDirLabel.AutoSize = true;
            this.ffmpegDirLabel.Location = new System.Drawing.Point(20, 90); // 位置調整
            this.ffmpegDirLabel.Name = "ffmpegDirLabel";
            this.ffmpegDirLabel.Size = new System.Drawing.Size(134, 18); // サイズ調整
            this.ffmpegDirLabel.TabIndex = 9; // TabIndex調整
            this.ffmpegDirLabel.Text = "ffmpegディレクトリ:";

            // ffmpegDirTextBox (textBox1 からリネーム)
            this.ffmpegDirTextBox.Location = new System.Drawing.Point(167, 85); // 位置調整
            this.ffmpegDirTextBox.Name = "ffmpegDirTextBox";
            this.ffmpegDirTextBox.Size = new System.Drawing.Size(500, 25); // サイズ調整
            this.ffmpegDirTextBox.TabIndex = 10; // TabIndex調整

            // ffmpegDirButton (button1 からリネーム)
            this.ffmpegDirButton.Location = new System.Drawing.Point(673, 85); // 位置調整
            this.ffmpegDirButton.Name = "ffmpegDirButton";
            this.ffmpegDirButton.Size = new System.Drawing.Size(75, 25); // サイズ調整
            this.ffmpegDirButton.TabIndex = 11; // TabIndex調整
            this.ffmpegDirButton.Text = "参照...";
            this.ffmpegDirButton.UseVisualStyleBackColor = true;

            // compressionGroupBox (圧縮オプション)
            this.compressionGroupBox.Controls.Add(this.rbCompressionQuality);
            this.compressionGroupBox.Controls.Add(this.rbCompressionStandard);
            this.compressionGroupBox.Controls.Add(this.rbCompressionSize);
            this.compressionGroupBox.Location = new System.Drawing.Point(20, 120); // 位置調整
            this.compressionGroupBox.Name = "compressionGroupBox";
            this.compressionGroupBox.Size = new System.Drawing.Size(150, 110); // サイズ調整
            this.compressionGroupBox.TabIndex = 4; // TabIndex調整
            this.compressionGroupBox.TabStop = false;
            this.compressionGroupBox.Text = "圧縮オプション";

            // rbCompressionSize (圧縮重視)
            this.rbCompressionSize.AutoSize = true;
            this.rbCompressionSize.Location = new System.Drawing.Point(15, 25); // グループボックス内の位置調整
            this.rbCompressionSize.Name = "rbCompressionSize";
            this.rbCompressionSize.Size = new System.Drawing.Size(105, 22); // サイズ調整
            this.rbCompressionSize.TabIndex = 0;
            this.rbCompressionSize.TabStop = true;
            this.rbCompressionSize.Text = "圧縮重視";
            this.rbCompressionSize.UseVisualStyleBackColor = true;

            // rbCompressionStandard (標準)
            this.rbCompressionStandard.AutoSize = true;
            this.rbCompressionStandard.Checked = true;
            this.rbCompressionStandard.Location = new System.Drawing.Point(15, 50); // グループボックス内の位置調整
            this.rbCompressionStandard.Name = "rbCompressionStandard";
            this.rbCompressionStandard.Size = new System.Drawing.Size(69, 22); // サイズ調整
            this.rbCompressionStandard.TabIndex = 1;
            this.rbCompressionStandard.TabStop = true;
            this.rbCompressionStandard.Text = "標準";
            this.rbCompressionStandard.UseVisualStyleBackColor = true;

            // rbCompressionQuality (画質重視)
            this.rbCompressionQuality.AutoSize = true;
            this.rbCompressionQuality.Location = new System.Drawing.Point(15, 75); // グループボックス内の位置調整
            this.rbCompressionQuality.Name = "rbCompressionQuality";
            this.rbCompressionQuality.Size = new System.Drawing.Size(105, 22); // サイズ調整
            this.rbCompressionQuality.TabIndex = 2;
            this.rbCompressionQuality.TabStop = true;
            this.rbCompressionQuality.Text = "画質重視";
            this.rbCompressionQuality.UseVisualStyleBackColor = true;


            // encoderGroupBox (規定エンコーダ選択)
            this.encoderGroupBox.Controls.Add(this.customEncoderTextBox);
            this.encoderGroupBox.Controls.Add(this.rbEncoderCustom);
            this.encoderGroupBox.Controls.Add(this.rbEncoderCPU);
            this.encoderGroupBox.Controls.Add(this.rbEncoderNVENC);
            this.encoderGroupBox.Location = new System.Drawing.Point(180, 120); // 位置調整
            this.encoderGroupBox.Name = "encoderGroupBox";
            this.encoderGroupBox.Size = new System.Drawing.Size(200, 140); // サイズ調整
            this.encoderGroupBox.TabIndex = 5; // TabIndex調整
            this.encoderGroupBox.TabStop = false;
            this.encoderGroupBox.Text = "規定エンコーダ選択";

            // rbEncoderNVENC (NVENC)
            this.rbEncoderNVENC.AutoSize = true;
            this.rbEncoderNVENC.Checked = true;
            this.rbEncoderNVENC.Location = new System.Drawing.Point(15, 25); // グループボックス内の位置調整
            this.rbEncoderNVENC.Name = "rbEncoderNVENC";
            this.rbEncoderNVENC.Size = new System.Drawing.Size(90, 22); // サイズ調整
            this.rbEncoderNVENC.TabIndex = 0;
            this.rbEncoderNVENC.TabStop = true;
            this.rbEncoderNVENC.Text = "NVENC";
            this.rbEncoderNVENC.UseVisualStyleBackColor = true;

            // rbEncoderCPU (CPU)
            this.rbEncoderCPU.AutoSize = true;
            this.rbEncoderCPU.Location = new System.Drawing.Point(15, 50); // グループボックス内の位置調整
            this.rbEncoderCPU.Name = "rbEncoderCPU";
            this.rbEncoderCPU.Size = new System.Drawing.Size(68, 22); // サイズ調整
            this.rbEncoderCPU.TabIndex = 1;
            this.rbEncoderCPU.TabStop = true;
            this.rbEncoderCPU.Text = "CPU";
            this.rbEncoderCPU.UseVisualStyleBackColor = true;

            // rbEncoderCustom (カスタム)
            this.rbEncoderCustom.AutoSize = true;
            this.rbEncoderCustom.Location = new System.Drawing.Point(15, 75); // グループボックス内の位置調整
            this.rbEncoderCustom.Name = "rbEncoderCustom";
            this.rbEncoderCustom.Size = new System.Drawing.Size(87, 22); // サイズ調整
            this.rbEncoderCustom.TabIndex = 2;
            this.rbEncoderCustom.TabStop = true;
            this.rbEncoderCustom.Text = "カスタム";
            this.rbEncoderCustom.UseVisualStyleBackColor = true;

            // customEncoderTextBox (カスタム入力欄)
            this.customEncoderTextBox.Location = new System.Drawing.Point(15, 100); // グループボックス内の位置調整
            this.customEncoderTextBox.Name = "customEncoderTextBox";
            this.customEncoderTextBox.Size = new System.Drawing.Size(170, 25); // サイズ調整
            this.customEncoderTextBox.TabIndex = 3;


            // modeGroupBox (動作モード)
            this.modeGroupBox.Controls.Add(this.cbQuickMode);
            this.modeGroupBox.Controls.Add(this.cbForceStart);
            this.modeGroupBox.Controls.Add(this.cbRestart);
            this.modeGroupBox.Location = new System.Drawing.Point(390, 120); // 位置調整
            this.modeGroupBox.Name = "modeGroupBox";
            this.modeGroupBox.Size = new System.Drawing.Size(150, 110); // サイズ調整
            this.modeGroupBox.TabIndex = 6; // TabIndex調整
            this.modeGroupBox.TabStop = false;
            this.modeGroupBox.Text = "動作モード";

            // cbRestart (再開)
            this.cbRestart.AutoSize = true;
            this.cbRestart.Location = new System.Drawing.Point(15, 25); // グループボックス内の位置調整
            this.cbRestart.Name = "cbRestart";
            this.cbRestart.Size = new System.Drawing.Size(70, 22); // サイズ調整
            this.cbRestart.TabIndex = 0;
            this.cbRestart.Text = "再開";
            this.cbRestart.UseVisualStyleBackColor = true;

            // cbForceStart (強制開始)
            this.cbForceStart.AutoSize = true;
            this.cbForceStart.Location = new System.Drawing.Point(15, 50); // グループボックス内の位置調整
            this.cbForceStart.Name = "cbForceStart";
            this.cbForceStart.Size = new System.Drawing.Size(106, 22); // サイズ調整
            this.cbForceStart.TabIndex = 1;
            this.cbForceStart.Text = "強制開始";
            this.cbForceStart.UseVisualStyleBackColor = true;

            // cbQuickMode (クイック)
            this.cbQuickMode.AutoSize = true;
            this.cbQuickMode.Location = new System.Drawing.Point(15, 75); // グループボックス内の位置調整
            this.cbQuickMode.Name = "cbQuickMode";
            this.cbQuickMode.Size = new System.Drawing.Size(81, 22); // サイズ調整
            this.cbQuickMode.TabIndex = 2;
            this.cbQuickMode.Text = "クイック";
            this.cbQuickMode.UseVisualStyleBackColor = true;

            // runButton
            this.runButton.Location = new System.Drawing.Point(550, 120); // 位置調整
            this.runButton.Name = "runButton";
            this.runButton.Size = new System.Drawing.Size(120, 50); // サイズ調整
            this.runButton.TabIndex = 7; // TabIndex調整
            this.runButton.Text = "変換開始";
            this.runButton.UseVisualStyleBackColor = true;

            // stopButton (button2 からリネーム)
            this.stopButton.Location = new System.Drawing.Point(550, 180); // 位置調整
            this.stopButton.Name = "stopButton";
            this.stopButton.Size = new System.Drawing.Size(120, 50); // サイズ調整
            this.stopButton.TabIndex = 8; // TabIndex調整
            this.stopButton.Text = "強制停止";
            this.stopButton.UseVisualStyleBackColor = true;

            // logLabel
            this.logLabel.AutoSize = true;
            this.logLabel.Location = new System.Drawing.Point(20, 270); // 位置調整
            this.logLabel.Name = "logLabel";
            this.logLabel.Size = new System.Drawing.Size(39, 18); // サイズ調整
            this.logLabel.TabIndex = 12; // TabIndex調整
            this.logLabel.Text = "ログ:";

            // cbShowLog (ログ表示切り替え)
            this.cbShowLog.AutoSize = true;
            this.cbShowLog.Checked = true; // デフォルトで表示
            this.cbShowLog.CheckState = System.Windows.Forms.CheckState.Checked;
            this.cbShowLog.Location = new System.Drawing.Point(80, 270); // 位置調整
            this.cbShowLog.Name = "cbShowLog";
            this.cbShowLog.Size = new System.Drawing.Size(103, 22); // サイズ調整
            this.cbShowLog.TabIndex = 13; // TabIndex調整
            this.cbShowLog.Text = "ログを表示";
            this.cbShowLog.UseVisualStyleBackColor = true;

            // timeoutLabel (タイムアウト)
            this.timeoutLabel.AutoSize = true;
            this.timeoutLabel.Location = new System.Drawing.Point(20, 245); // 位置調整
            this.timeoutLabel.Name = "timeoutLabel";
            this.timeoutLabel.Size = new System.Drawing.Size(112, 18); // サイズ調整
            this.timeoutLabel.TabIndex = 14; // TabIndex調整
            this.timeoutLabel.Text = "タイムアウト (秒):";

            // timeoutNumericUpDown (タイムアウト)
            this.timeoutNumericUpDown.Location = new System.Drawing.Point(140, 243); // 位置調整
            this.timeoutNumericUpDown.Maximum = new decimal(new int[] {
            -1,
            -1,
            0,
            0}); // 最大値を設定しない (または大きな値を設定)
            this.timeoutNumericUpDown.Name = "timeoutNumericUpDown";
            this.timeoutNumericUpDown.Size = new System.Drawing.Size(80, 25); // サイズ調整
            this.timeoutNumericUpDown.TabIndex = 15; // TabIndex調整
            this.timeoutNumericUpDown.Value = new decimal(new int[] {
            7200, // デフォルト値 7200秒
            0,
            0,
            0});

            // cbDebugMode (デバッグモード)
            this.cbDebugMode.AutoSize = true;
            this.cbDebugMode.Location = new System.Drawing.Point(240, 245); // 位置調整
            this.cbDebugMode.Name = "cbDebugMode";
            this.cbDebugMode.Size = new System.Drawing.Size(110, 22); // サイズ調整
            this.cbDebugMode.TabIndex = 16; // TabIndex調整
            this.cbDebugMode.Text = "デバッグモード";
            this.cbDebugMode.UseVisualStyleBackColor = true;


            // logRichTextBox
            this.logRichTextBox.Location = new System.Drawing.Point(20, 295); // 位置調整
            this.logRichTextBox.Name = "logRichTextBox";
            this.logRichTextBox.ReadOnly = true;
            this.logRichTextBox.Size = new System.Drawing.Size(750, 350); // サイズ調整
            this.logRichTextBox.TabIndex = 17; // TabIndex調整
            this.logRichTextBox.Text = "";
            this.logRichTextBox.Multiline = true;
            // ウィンドウサイズ変更に合わせて自動調整する場合、Anchorプロパティを設定すると便利です。
            // 例: this.logRichTextBox.Anchor = ((System.Windows.Forms.AnchorStyles)((((System.Windows.Forms.AnchorStyles.Top | System.Windows.Forms.AnchorStyles.Bottom) | System.Windows.Forms.AnchorStyles.Left) | System.Windows.Forms.AnchorStyles.Right)));


            // フォームにコントロールを追加
            // Controls.Add の順序は、コントロールの重なり順に影響します。
            // 通常、背景に近いものから手前に向かって追加します。
            this.Controls.Add(this.logRichTextBox); // ログエリア
            this.Controls.Add(this.cbDebugMode); // デバッグモードチェックボックス
            this.Controls.Add(this.timeoutNumericUpDown); // タイムアウトNumericUpDown
            this.Controls.Add(this.timeoutLabel); // タイムアウトラベル
            this.Controls.Add(this.cbShowLog); // ログ表示チェックボックス
            this.Controls.Add(this.logLabel); // ログラベル
            this.Controls.Add(this.stopButton); // 強制停止ボタン
            this.Controls.Add(this.runButton); // 実行ボタン
            this.Controls.Add(this.ffmpegDirButton); // ffmpeg参照ボタン
            this.Controls.Add(this.ffmpegDirTextBox); // ffmpegテキストボックス
            this.Controls.Add(this.ffmpegDirLabel); // ffmpegラベル
            this.Controls.Add(this.outputDirButton); // 出力参照ボタン
            this.Controls.Add(this.outputDirTextBox); // 出力テキストボックス
            this.Controls.Add(this.outputDirLabel); // 出力ラベル
            this.Controls.Add(this.inputDirButton); // 入力参照ボタン
            this.Controls.Add(this.inputDirTextBox); // 入力テキストボックス
            this.Controls.Add(this.inputDirLabel); // 入力ラベル

            // グループボックスをフォームに追加
            this.Controls.Add(this.compressionGroupBox);
            this.Controls.Add(this.encoderGroupBox);
            this.Controls.Add(this.modeGroupBox);


            // フォーム自体の設定
            this.AutoScaleDimensions = new System.Drawing.SizeF(10F, 18F);
            this.AutoScaleMode = System.Windows.Forms.AutoScaleMode.Font;
            this.ClientSize = new System.Drawing.Size(784, 661); // ウィンドウサイズを調整
            this.Name = "Form1";
            this.Text = "TransAV1 GUI";
            this.Load += new System.EventHandler(this.Form1_Load); // Loadイベントハンドラを追加

            // グループボックスの子コントロールのレイアウト再開
            this.compressionGroupBox.ResumeLayout(false);
            this.compressionGroupBox.PerformLayout();
            this.encoderGroupBox.ResumeLayout(false);
            this.encoderGroupBox.PerformLayout();
            this.modeGroupBox.ResumeLayout(false);
            this.modeGroupBox.PerformLayout();
            ((System.ComponentModel.ISupportInitialize)(this.timeoutNumericUpDown)).EndInit(); // NumericUpDownのレイアウト再開

            this.ResumeLayout(false); // フォーム全体のレイアウト再開
            this.PerformLayout(); // レイアウトの再計算

        }

        #endregion

        // デザイナーが生成するコントロールの変数宣言
        // 既存の宣言を削除し、以下に置き換えます。
        private System.Windows.Forms.TextBox inputDirTextBox;
        private System.Windows.Forms.Button inputDirButton;
        private System.Windows.Forms.TextBox outputDirTextBox;
        private System.Windows.Forms.Button outputDirButton;
        private System.Windows.Forms.Button runButton;
        private System.Windows.Forms.RichTextBox logRichTextBox;
        private System.Windows.Forms.Label inputDirLabel;
        private System.Windows.Forms.Label outputDirLabel;
        private System.Windows.Forms.Label logLabel;

        // 新規追加コントロールの変数宣言
        private System.Windows.Forms.GroupBox compressionGroupBox;
        private System.Windows.Forms.RadioButton rbCompressionQuality;
        private System.Windows.Forms.RadioButton rbCompressionStandard;
        private System.Windows.Forms.RadioButton rbCompressionSize;

        private System.Windows.Forms.GroupBox encoderGroupBox;
        private System.Windows.Forms.TextBox customEncoderTextBox;
        private System.Windows.Forms.RadioButton rbEncoderCustom;
        private System.Windows.Forms.RadioButton rbEncoderCPU;
        private System.Windows.Forms.RadioButton rbEncoderNVENC;

        private System.Windows.Forms.GroupBox modeGroupBox;
        private System.Windows.Forms.CheckBox cbQuickMode;
        private System.Windows.Forms.CheckBox cbForceStart;
        private System.Windows.Forms.CheckBox cbRestart;

        private System.Windows.Forms.Label ffmpegDirLabel; // label1 からリネーム
        private System.Windows.Forms.TextBox ffmpegDirTextBox; // textBox1 からリネーム
        private System.Windows.Forms.Button ffmpegDirButton; // button1 からリネーム

        private System.Windows.Forms.Button stopButton; // button2 からリネーム

        private System.Windows.Forms.CheckBox cbShowLog; // ログ表示切り替え

        private System.Windows.Forms.Label timeoutLabel; // タイムアウト
        private System.Windows.Forms.NumericUpDown timeoutNumericUpDown; // タイムアウト

        private System.Windows.Forms.CheckBox cbDebugMode; // デバッグモード
    }
}
