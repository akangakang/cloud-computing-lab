# go环境配置

## linux

1. Extract the archive you downloaded into /usr/local, creating a Go tree in /usr/local/go.

   **Important:** This step will remove a previous installation at /usr/local/go, if any, prior to extracting. Please back up any data before proceeding.

   For example, run the following as root or through `sudo`:

   ```
   rm -rf /usr/local/go && tar -C /usr/local -xzf go1.16.4.linux-amd64.tar.gz
   ```

2. Add /usr/local/go/bin to the

    

   ```
   PATH
   ```

    

   environment variable.

   You can do this by adding the following line to your $HOME/.profile or /etc/profile (for a system-wide installation):

   ```bash
   os@ubuntu:~/go/bin$ sudo vim  /etc/profile
   ```

   在最后两行加上

   ```
   export PATH=$PATH:/usr/local/go/bin
   export PATH=$PATH:$GOROOT/bin
   ```

   **Note:** Changes made to a profile file may not apply until the next time you log into your computer. To apply the changes immediately, just run the shell commands directly or execute them from the profile using a command such as `source $HOME/.profile`.

3. Verify that you've installed Go by opening a command prompt and typing the following command:

   ```
   $ go version
   ```

4. Confirm that the command prints the installed version of Go.



## vscode

1. 安装go插件只能高亮
2. 代码提示补全：
   1. Windows平台按下`Ctrl+Shift+P`在这个输入框中输入`>go:install`
   2. 下面会自动搜索相关命令，我们选择`Go:Install/Update Tools`这个命令，按下图选中并会回车执行该命令（或者使用鼠标点击该命令）
   3. 在弹出的窗口选中所有，并点击“确定”按钮，进行安装。