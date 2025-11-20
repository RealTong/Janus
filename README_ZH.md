# Janus
> Janus 是一个用于在 Linux 和 Windows 双系统之间切换的工具，支持通过 Telegram Bot 进行控制。

> 项目名称 `Janus` 由 Google Gemini 赐名，寓意为：「Janus 是罗马神话中的双面神，他有两副面孔，一副注视过去，一副注视未来。他是门户、开端、转变之神。」

## Install
Janus 的核心原理是通过 Redis 进行消息传递，当 Janus 接收到 Telegram Bot 的命令后，会通过 Redis 将命令发送给 Janus 进程，Janus 进程会根据命令执行相应的操作。

Janus 依赖 Grub 进行系统切换，因此需要确保 Grub 已经正确安装。并且将 Linux 系统作为 Grub 默认启动项。

切换到 Windows：当 Janus 接收到「切换到 Windows」命令后，会执行 `grub-reboot "Windows Boot Manager (on /dev/nvmexxxxx)"` 命令，然后重启系统。

切换到 Linux：当 Janus 接收到「切换到 Linux」命令后，会执行 `shutdown /r /t 0` 命令，然后重启系统，这时，Grub 会启动默认的启动项，即 Linux 系统。

### 安装前提
- 确保 Linux 系统作为 Grub 默认启动项。
- 将 Grub 的 Windows Boot Manager 的 entry 复制出来，并写入 `config.yaml`。

### Linux 
1. 根据系统架构下载最新版本的 [Janus](https://github.com/RealTong/janus/releases)
2. 将 `janus` 文件复制到 `/opt/janus/` 目录下
3. `wget https://raw.githubusercontent.com/RealTong/Janus/refs/heads/main/config.example.yaml -O /opt/janus/config.yaml`
4. 编辑 `config.yaml` 文件，根据实际情况修改配置。
5. `vim /etc/systemd/system/janus.service`
   ```ini
   [Unit]
   Description=Janus - System Control Service
   After=network.target
   Wants=network.target

   [Service]
   Type=simple
   User=root
   WorkingDirectory=/opt/janus
   ExecStart=/opt/janus/janus
   Restart=always
   RestartSec=10
   StandardOutput=journal
   StandardError=journal
   SyslogIdentifier=janus

   NoNewPrivileges=false
   PrivateTmp=false

   [Install]
   WantedBy=multi-user.target
   ```
6. `systemctl daemon-reload`
7. `systemctl enable janus`
8. `systemctl start janus`
9. 查看 Janus 服务状态：`systemctl status janus`
10. 停止 Janus 服务：`systemctl stop janus`
11. 重启 Janus 服务：`systemctl restart janus`

### Windows
1. 根据系统架构下载最新版本的 [Janus](https://github.com/RealTong/janus/releases)
2. 将 `janus` 文件复制到 `C:\Program Files\janus\` 目录下
3. `wget https://raw.githubusercontent.com/RealTong/Janus/refs/heads/main/config.example.yaml -O C:\Program Files\janus\config.yaml`
4. 编辑 `config.yaml` 文件，根据实际情况修改配置。
5. 下载 NSSM 并安装：`https://nssm.cc/release/nssm-2.24.zip`
6. 解压 NSSM 并且将 `nssm.exe` 复制到 `C:\Program Files\janus\` 目录下（确保 `nssm.exe` 和 `janus.exe` 在同一目录）
7. 打开命令行，执行 `nssm.exe install Janus "C:\Program Files\janus\janus.exe"`
8. 打开 `Services` 管理器，找到 `Janus` 服务，点击 `启动`，并设置为开机启动。