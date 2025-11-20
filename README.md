# Janus
> Janus is a tool for switching between Linux and Windows dual-boot systems, with support for control via Telegram Bot.

> The project name `Janus` was bestowed by Google Gemini, symbolizing: "Janus is the two-faced god in Roman mythology, with one face looking to the past and the other to the future. He is the god of doorways, beginnings, and transitions."

## Install
The core principle of Janus is message passing through Redis. When Janus receives commands from the Telegram Bot, it sends the commands to the Janus process via Redis, which then executes the corresponding operations based on the commands.

Janus relies on Grub for system switching, so you need to ensure that Grub is properly installed. Linux system should be set as the default boot option in Grub.

Switch to Windows: When Janus receives the "Switch to Windows" command, it executes `grub-reboot "Windows Boot Manager (on /dev/nvmexxxxx)"` command, then reboots the system.

Switch to Linux: When Janus receives the "Switch to Linux" command, it executes `shutdown /r /t 0` command, then reboots the system. At this point, Grub will boot the default boot option, which is the Linux system.

### Prerequisites
- Ensure Linux system is set as the default boot option in Grub.
- Copy the Windows Boot Manager entry from Grub and write it into `config.yaml`.

### Linux
1. Download the latest version of [Janus](https://github.com/RealTong/janus/releases) according to your system architecture
2. Copy the `janus` file to the `/opt/janus/` directory
3. `wget https://raw.githubusercontent.com/RealTong/Janus/refs/heads/main/config.example.yaml -O /opt/janus/config.yaml`
4. Edit the `config.yaml` file and modify the configuration according to your actual situation.
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
9. Check Janus service status: `systemctl status janus`
10. Stop Janus service: `systemctl stop janus`
11. Restart Janus service: `systemctl restart janus`

### Windows
1. Download the latest version of [Janus](https://github.com/RealTong/janus/releases) according to your system architecture
2. Copy the `janus` file to the `C:\Program Files\janus\` directory
3. `wget https://raw.githubusercontent.com/RealTong/Janus/refs/heads/main/config.example.yaml -O C:\Program Files\janus\config.yaml`
4. Edit the `config.yaml` file and modify the configuration according to your actual situation.
5. Download and install NSSM: `https://nssm.cc/release/nssm-2.24.zip`
6. Extract NSSM and copy `nssm.exe` to the `C:\Program Files\janus\` directory (ensure `nssm.exe` and `janus.exe` are in the same directory)
7. Open command prompt and execute `nssm.exe install Janus "C:\Program Files\janus\janus.exe"`
8. Open the `Services` manager, find the `Janus` service, click `Start`, and set it to start automatically at boot.

