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
            // コントロールのインスタンス生成 (デザイナーが認識しやすいように先にまとめる)
            this.inputDirTextBox = new System.Windows.Forms.TextBox();
            this.inputDirButton = new System.Windows.Forms.Button();
            this.outputDirTextBox = new System.Windows.Forms.TextBox();
            this.outputDirButton = new System.Windows.Forms.Button();
            this.runButton = new System.Windows.Forms.Button();
            this.logRichTextBox = new System.Windows.Forms.RichTextBox();
            this.inputDirLabel = new System.Windows.Forms.Label();
            this.outputDirLabel = new System.Windows.Forms.Label();
            this.logLabel = new System.Windows.Forms.Label(); // ★ logLabel のインスタンス生成
            this.compressionGroupBox = new System.Windows.Forms.GroupBox();
            this.rbCompressionQuality = new System.Windows.Forms.RadioButton();
            this.rbCompressionStandard = new System.Windows.Forms.RadioButton();
            this.rbCompressionSize = new System.Windows.Forms.RadioButton();
            this.encoderGroupBox = new System.Windows.Forms.GroupBox();
            this.customEncoderTextBox = new System.Windows.Forms.TextBox();
            this.rbEncoderCustom = new System.Windows.Forms.RadioButton();
            this.rbEncoderCPU = new System.Windows.Forms.RadioButton();
            this.rbEncoderNVENC = new System.Windows.Forms.RadioButton();
            this.modeGroupBox = new System.Windows.Forms.GroupBox();
            this.rbModeForceStart = new System.Windows.Forms.RadioButton();
            this.rbModeRestart = new System.Windows.Forms.RadioButton();
            this.rbModeNormal = new System.Windows.Forms.RadioButton();
            this.cbQuickMode = new System.Windows.Forms.CheckBox();
            this.cbDebugMode = new System.Windows.Forms.CheckBox();
            this.ffmpegDirLabel = new System.Windows.Forms.Label();
            this.ffmpegDirTextBox = new System.Windows.Forms.TextBox();
            this.ffmpegDirButton = new System.Windows.Forms.Button();
            this.stopButton = new System.Windows.Forms.Button();
            this.cbShowLog = new System.Windows.Forms.CheckBox();
            this.timeoutLabel = new System.Windows.Forms.Label();
            this.timeoutNumericUpDown = new System.Windows.Forms.NumericUpDown();

            // SuspendLayout/ResumeLayout はグループボックスとフォームに対して行う
            this.compressionGroupBox.SuspendLayout();
            this.encoderGroupBox.SuspendLayout();
            this.modeGroupBox.SuspendLayout();
            ((System.ComponentModel.ISupportInitialize)(this.timeoutNumericUpDown)).BeginInit();
            this.SuspendLayout();

            //
            // inputDirTextBox
            //
            this.inputDirTextBox.Location = new System.Drawing.Point(167, 15);
            this.inputDirTextBox.Name = "inputDirTextBox";
            this.inputDirTextBox.Size = new System.Drawing.Size(500, 25);
            this.inputDirTextBox.TabIndex = 0;
            //
            // inputDirButton
            //
            this.inputDirButton.Location = new System.Drawing.Point(673, 15);
            this.inputDirButton.Name = "inputDirButton";
            this.inputDirButton.Size = new System.Drawing.Size(75, 25);
            this.inputDirButton.TabIndex = 1;
            this.inputDirButton.Text = "参照...";
            this.inputDirButton.UseVisualStyleBackColor = true;
            //
            // outputDirTextBox
            //
            this.outputDirTextBox.Location = new System.Drawing.Point(167, 50);
            this.outputDirTextBox.Name = "outputDirTextBox";
            this.outputDirTextBox.Size = new System.Drawing.Size(500, 25);
            this.outputDirTextBox.TabIndex = 2;
            //
            // outputDirButton
            //
            this.outputDirButton.Location = new System.Drawing.Point(673, 50);
            this.outputDirButton.Name = "outputDirButton";
            this.outputDirButton.Size = new System.Drawing.Size(75, 25);
            this.outputDirButton.TabIndex = 3;
            this.outputDirButton.Text = "参照...";
            this.outputDirButton.UseVisualStyleBackColor = true;
            //
            // runButton
            //
            this.runButton.Location = new System.Drawing.Point(609, 130); // 位置調整
            this.runButton.Name = "runButton";
            this.runButton.Size = new System.Drawing.Size(163, 87); // サイズ調整
            this.runButton.TabIndex = 7; // TabIndex調整
            this.runButton.Text = "変換開始";
            this.runButton.UseVisualStyleBackColor = true;
            //
            // logRichTextBox
            //
            this.logRichTextBox.DetectUrls = false; // URL検出無効
            this.logRichTextBox.Location = new System.Drawing.Point(20, 295);
            this.logRichTextBox.Name = "logRichTextBox";
            this.logRichTextBox.ReadOnly = true;
            this.logRichTextBox.Size = new System.Drawing.Size(750, 350);
            this.logRichTextBox.TabIndex = 17; // TabIndex調整
            this.logRichTextBox.Text = "";
            //
            // inputDirLabel
            //
            this.inputDirLabel.AutoSize = true;
            this.inputDirLabel.Location = new System.Drawing.Point(20, 20);
            this.inputDirLabel.Name = "inputDirLabel";
            this.inputDirLabel.Size = new System.Drawing.Size(141, 18);
            this.inputDirLabel.TabIndex = 19; // TabIndex調整 (ラベルは通常後ろの方)
            this.inputDirLabel.Text = "入力元ディレクトリ:";
            //
            // outputDirLabel
            //
            this.outputDirLabel.AutoSize = true;
            this.outputDirLabel.Location = new System.Drawing.Point(20, 55);
            this.outputDirLabel.Name = "outputDirLabel";
            this.outputDirLabel.Size = new System.Drawing.Size(141, 18);
            this.outputDirLabel.TabIndex = 18; // TabIndex調整
            this.outputDirLabel.Text = "出力先ディレクトリ:";
            //
            // logLabel
            // ★★★ logLabel のプロパティ設定 ★★★
            this.logLabel.AutoSize = true;
            this.logLabel.Location = new System.Drawing.Point(20, 270); // 位置確認
            this.logLabel.Name = "logLabel";
            this.logLabel.Size = new System.Drawing.Size(39, 18); // サイズ確認
            this.logLabel.TabIndex = 12; // TabIndex確認
            this.logLabel.Text = "ログ:";
            //
            // compressionGroupBox
            //
            this.compressionGroupBox.Controls.Add(this.rbCompressionQuality);
            this.compressionGroupBox.Controls.Add(this.rbCompressionStandard);
            this.compressionGroupBox.Controls.Add(this.rbCompressionSize);
            this.compressionGroupBox.Location = new System.Drawing.Point(20, 120);
            this.compressionGroupBox.Name = "compressionGroupBox";
            this.compressionGroupBox.Size = new System.Drawing.Size(150, 110);
            this.compressionGroupBox.TabIndex = 4;
            this.compressionGroupBox.TabStop = false;
            this.compressionGroupBox.Text = "圧縮オプション";
            //
            // rbCompressionQuality
            //
            this.rbCompressionQuality.AutoSize = true;
            this.rbCompressionQuality.Location = new System.Drawing.Point(15, 75);
            this.rbCompressionQuality.Name = "rbCompressionQuality";
            this.rbCompressionQuality.Size = new System.Drawing.Size(105, 22);
            this.rbCompressionQuality.TabIndex = 2;
            this.rbCompressionQuality.Text = "画質重視";
            this.rbCompressionQuality.UseVisualStyleBackColor = true;
            //
            // rbCompressionStandard
            //
            this.rbCompressionStandard.AutoSize = true;
            this.rbCompressionStandard.Checked = true;
            this.rbCompressionStandard.Location = new System.Drawing.Point(15, 50);
            this.rbCompressionStandard.Name = "rbCompressionStandard";
            this.rbCompressionStandard.Size = new System.Drawing.Size(69, 22);
            this.rbCompressionStandard.TabIndex = 1;
            this.rbCompressionStandard.TabStop = true;
            this.rbCompressionStandard.Text = "標準";
            this.rbCompressionStandard.UseVisualStyleBackColor = true;
            //
            // rbCompressionSize
            //
            this.rbCompressionSize.AutoSize = true;
            this.rbCompressionSize.Location = new System.Drawing.Point(15, 25);
            this.rbCompressionSize.Name = "rbCompressionSize";
            this.rbCompressionSize.Size = new System.Drawing.Size(105, 22);
            this.rbCompressionSize.TabIndex = 0;
            this.rbCompressionSize.Text = "圧縮重視";
            this.rbCompressionSize.UseVisualStyleBackColor = true;
            //
            // encoderGroupBox
            //
            this.encoderGroupBox.Controls.Add(this.customEncoderTextBox);
            this.encoderGroupBox.Controls.Add(this.rbEncoderCustom);
            this.encoderGroupBox.Controls.Add(this.rbEncoderCPU);
            this.encoderGroupBox.Controls.Add(this.rbEncoderNVENC);
            this.encoderGroupBox.Location = new System.Drawing.Point(226, 123); // 位置確認
            this.encoderGroupBox.Name = "encoderGroupBox";
            this.encoderGroupBox.Size = new System.Drawing.Size(200, 140); // サイズ確認
            this.encoderGroupBox.TabIndex = 5;
            this.encoderGroupBox.TabStop = false;
            this.encoderGroupBox.Text = "規定エンコーダ選択";
            //
            // customEncoderTextBox
            //
            this.customEncoderTextBox.Location = new System.Drawing.Point(15, 113); // 位置確認
            this.customEncoderTextBox.Name = "customEncoderTextBox";
            this.customEncoderTextBox.Size = new System.Drawing.Size(170, 25); // サイズ確認
            this.customEncoderTextBox.TabIndex = 3;
            //
            // rbEncoderCustom
            //
            this.rbEncoderCustom.AutoSize = true;
            this.rbEncoderCustom.Location = new System.Drawing.Point(15, 85); // 位置確認
            this.rbEncoderCustom.Name = "rbEncoderCustom";
            this.rbEncoderCustom.Size = new System.Drawing.Size(87, 22); // サイズ確認
            this.rbEncoderCustom.TabIndex = 2;
            this.rbEncoderCustom.Text = "カスタム";
            this.rbEncoderCustom.UseVisualStyleBackColor = true;
            //
            // rbEncoderCPU
            //
            this.rbEncoderCPU.AutoSize = true;
            this.rbEncoderCPU.Location = new System.Drawing.Point(15, 50);
            this.rbEncoderCPU.Name = "rbEncoderCPU";
            this.rbEncoderCPU.Size = new System.Drawing.Size(68, 22);
            this.rbEncoderCPU.TabIndex = 1;
            this.rbEncoderCPU.Text = "CPU";
            this.rbEncoderCPU.UseVisualStyleBackColor = true;
            //
            // rbEncoderNVENC
            //
            this.rbEncoderNVENC.AutoSize = true;
            this.rbEncoderNVENC.Checked = true;
            this.rbEncoderNVENC.Location = new System.Drawing.Point(15, 25);
            this.rbEncoderNVENC.Name = "rbEncoderNVENC";
            this.rbEncoderNVENC.Size = new System.Drawing.Size(90, 22);
            this.rbEncoderNVENC.TabIndex = 0;
            this.rbEncoderNVENC.TabStop = true;
            this.rbEncoderNVENC.Text = "NVENC";
            this.rbEncoderNVENC.UseVisualStyleBackColor = true;
            //
            // modeGroupBox
            //
            this.modeGroupBox.Controls.Add(this.rbModeForceStart);
            this.modeGroupBox.Controls.Add(this.rbModeRestart);
            this.modeGroupBox.Controls.Add(this.rbModeNormal);
            this.modeGroupBox.Controls.Add(this.cbQuickMode);
            this.modeGroupBox.Controls.Add(this.cbDebugMode);
            this.modeGroupBox.Location = new System.Drawing.Point(432, 116); // 位置確認
            this.modeGroupBox.Name = "modeGroupBox";
            this.modeGroupBox.Size = new System.Drawing.Size(171, 168); // サイズ確認
            this.modeGroupBox.TabIndex = 6;
            this.modeGroupBox.TabStop = false;
            this.modeGroupBox.Text = "動作モード";
            //
            // rbModeForceStart
            //
            this.rbModeForceStart.AutoSize = true;
            this.rbModeForceStart.Location = new System.Drawing.Point(15, 75);
            this.rbModeForceStart.Name = "rbModeForceStart";
            this.rbModeForceStart.Size = new System.Drawing.Size(105, 22);
            this.rbModeForceStart.TabIndex = 2;
            this.rbModeForceStart.Text = "強制開始";
            this.rbModeForceStart.UseVisualStyleBackColor = true;
            //
            // rbModeRestart
            //
            this.rbModeRestart.AutoSize = true;
            this.rbModeRestart.Location = new System.Drawing.Point(15, 50);
            this.rbModeRestart.Name = "rbModeRestart";
            this.rbModeRestart.Size = new System.Drawing.Size(69, 22);
            this.rbModeRestart.TabIndex = 1;
            this.rbModeRestart.Text = "再開";
            this.rbModeRestart.UseVisualStyleBackColor = true;
            //
            // rbModeNormal
            //
            this.rbModeNormal.AutoSize = true;
            this.rbModeNormal.Checked = true;
            this.rbModeNormal.Location = new System.Drawing.Point(15, 25);
            this.rbModeNormal.Name = "rbModeNormal";
            this.rbModeNormal.Size = new System.Drawing.Size(69, 22);
            this.rbModeNormal.TabIndex = 0;
            this.rbModeNormal.TabStop = true;
            this.rbModeNormal.Text = "通常";
            this.rbModeNormal.UseVisualStyleBackColor = true;
            //
            // cbQuickMode
            //
            this.cbQuickMode.AutoSize = true;
            this.cbQuickMode.Location = new System.Drawing.Point(15, 105);
            this.cbQuickMode.Name = "cbQuickMode";
            this.cbQuickMode.Size = new System.Drawing.Size(81, 22);
            this.cbQuickMode.TabIndex = 3;
            this.cbQuickMode.Text = "クイック";
            this.cbQuickMode.UseVisualStyleBackColor = true;
            //
            // cbDebugMode
            //
            this.cbDebugMode.AutoSize = true;
            this.cbDebugMode.Location = new System.Drawing.Point(15, 130);
            this.cbDebugMode.Name = "cbDebugMode";
            this.cbDebugMode.Size = new System.Drawing.Size(130, 22);
            this.cbDebugMode.TabIndex = 4;
            this.cbDebugMode.Text = "デバッグモード";
            this.cbDebugMode.UseVisualStyleBackColor = true;
            //
            // ffmpegDirLabel
            //
            this.ffmpegDirLabel.AutoSize = true;
            this.ffmpegDirLabel.Location = new System.Drawing.Point(20, 90);
            this.ffmpegDirLabel.Name = "ffmpegDirLabel";
            this.ffmpegDirLabel.Size = new System.Drawing.Size(136, 18);
            this.ffmpegDirLabel.TabIndex = 9; // TabIndex確認
            this.ffmpegDirLabel.Text = "ffmpegディレクトリ:";
            //
            // ffmpegDirTextBox
            //
            this.ffmpegDirTextBox.Location = new System.Drawing.Point(167, 85);
            this.ffmpegDirTextBox.Name = "ffmpegDirTextBox";
            this.ffmpegDirTextBox.Size = new System.Drawing.Size(500, 25);
            this.ffmpegDirTextBox.TabIndex = 10; // TabIndex確認
            //
            // ffmpegDirButton
            //
            this.ffmpegDirButton.Location = new System.Drawing.Point(673, 85);
            this.ffmpegDirButton.Name = "ffmpegDirButton";
            this.ffmpegDirButton.Size = new System.Drawing.Size(75, 25);
            this.ffmpegDirButton.TabIndex = 11; // TabIndex確認
            this.ffmpegDirButton.Text = "参照...";
            this.ffmpegDirButton.UseVisualStyleBackColor = true;
            //
            // stopButton
            //
            this.stopButton.Location = new System.Drawing.Point(652, 229); // 位置確認
            this.stopButton.Name = "stopButton";
            this.stopButton.Size = new System.Drawing.Size(120, 50); // サイズ確認
            this.stopButton.TabIndex = 8; // TabIndex確認
            this.stopButton.Text = "強制停止";
            this.stopButton.UseVisualStyleBackColor = true;
            //
            // cbShowLog
            //
            this.cbShowLog.AutoSize = true;
            this.cbShowLog.Checked = true;
            this.cbShowLog.CheckState = System.Windows.Forms.CheckState.Checked;
            this.cbShowLog.Location = new System.Drawing.Point(80, 270); // 位置確認
            this.cbShowLog.Name = "cbShowLog";
            this.cbShowLog.Size = new System.Drawing.Size(111, 22); // サイズ確認
            this.cbShowLog.TabIndex = 13; // TabIndex確認
            this.cbShowLog.Text = "ログを表示";
            this.cbShowLog.UseVisualStyleBackColor = true;
            //
            // timeoutLabel
            //
            this.timeoutLabel.AutoSize = true;
            this.timeoutLabel.Location = new System.Drawing.Point(20, 245);
            this.timeoutLabel.Name = "timeoutLabel";
            this.timeoutLabel.Size = new System.Drawing.Size(125, 18);
            this.timeoutLabel.TabIndex = 14; // TabIndex確認
            this.timeoutLabel.Text = "タイムアウト (秒):";
            //
            // timeoutNumericUpDown
            //
            this.timeoutNumericUpDown.Location = new System.Drawing.Point(140, 243);
            this.timeoutNumericUpDown.Maximum = new decimal(new int[] { 86400, 0, 0, 0 });
            this.timeoutNumericUpDown.Minimum = new decimal(new int[] { 0, 0, 0, 0 });
            this.timeoutNumericUpDown.Name = "timeoutNumericUpDown";
            this.timeoutNumericUpDown.Size = new System.Drawing.Size(80, 25);
            this.timeoutNumericUpDown.TabIndex = 15; // TabIndex確認
            this.timeoutNumericUpDown.Value = new decimal(new int[] { 7200, 0, 0, 0 });
            //
            // Form1
            //
            this.AutoScaleDimensions = new System.Drawing.SizeF(10F, 18F);
            this.AutoScaleMode = System.Windows.Forms.AutoScaleMode.Font;
            this.ClientSize = new System.Drawing.Size(784, 661); // ウィンドウサイズ確認
            // ★★★ フォームへのコントロール追加順序を確認・修正 ★★★
            this.Controls.Add(this.timeoutNumericUpDown);
            this.Controls.Add(this.timeoutLabel);
            this.Controls.Add(this.cbShowLog);
            this.Controls.Add(this.logLabel); // logLabel を追加
            this.Controls.Add(this.stopButton);
            this.Controls.Add(this.runButton);
            this.Controls.Add(this.ffmpegDirButton);
            this.Controls.Add(this.ffmpegDirTextBox);
            this.Controls.Add(this.ffmpegDirLabel);
            this.Controls.Add(this.modeGroupBox); // GroupBox を追加
            this.Controls.Add(this.encoderGroupBox); // GroupBox を追加
            this.Controls.Add(this.compressionGroupBox); // GroupBox を追加
            this.Controls.Add(this.outputDirButton);
            this.Controls.Add(this.outputDirTextBox);
            this.Controls.Add(this.outputDirLabel);
            this.Controls.Add(this.inputDirButton);
            this.Controls.Add(this.inputDirTextBox);
            this.Controls.Add(this.inputDirLabel);
            this.Controls.Add(this.logRichTextBox); // RichTextBox は最後の方でも良い
            this.Name = "Form1";
            this.Text = "TransAV1 GUI";
            this.Load += new System.EventHandler(this.Form1_Load);
            this.compressionGroupBox.ResumeLayout(false);
            this.compressionGroupBox.PerformLayout();
            this.encoderGroupBox.ResumeLayout(false);
            this.encoderGroupBox.PerformLayout();
            this.modeGroupBox.ResumeLayout(false);
            this.modeGroupBox.PerformLayout();
            ((System.ComponentModel.ISupportInitialize)(this.timeoutNumericUpDown)).EndInit();
            this.ResumeLayout(false);
            this.PerformLayout();

        }

        #endregion

        // デザイナーが生成するコントロールの変数宣言
        private System.Windows.Forms.TextBox inputDirTextBox;
        private System.Windows.Forms.Button inputDirButton;
        private System.Windows.Forms.TextBox outputDirTextBox;
        private System.Windows.Forms.Button outputDirButton;
        private System.Windows.Forms.Button runButton;
        private System.Windows.Forms.RichTextBox logRichTextBox;
        private System.Windows.Forms.Label inputDirLabel;
        private System.Windows.Forms.Label outputDirLabel;
        private System.Windows.Forms.Label logLabel; // ★ logLabel の宣言
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
        private System.Windows.Forms.RadioButton rbModeForceStart;
        private System.Windows.Forms.RadioButton rbModeRestart;
        private System.Windows.Forms.RadioButton rbModeNormal;
        private System.Windows.Forms.CheckBox cbQuickMode;
        private System.Windows.Forms.CheckBox cbDebugMode;
        private System.Windows.Forms.Label ffmpegDirLabel;
        private System.Windows.Forms.TextBox ffmpegDirTextBox;
        private System.Windows.Forms.Button ffmpegDirButton;
        private System.Windows.Forms.Button stopButton;
        private System.Windows.Forms.CheckBox cbShowLog;
        private System.Windows.Forms.Label timeoutLabel;
        private System.Windows.Forms.NumericUpDown timeoutNumericUpDown;
    }
}
