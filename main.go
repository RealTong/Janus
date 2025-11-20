package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"janus/config"
	"janus/pkg/osinfo"
	"janus/pkg/redis"
	"janus/pkg/telegram"

	redisv8 "github.com/go-redis/redis/v8"
)

func main() {
	// 1. 初始化配置
	if err := config.InitConfig(""); err != nil {
		log.Fatalf("❌ 配置初始化失败: %v", err)
	}

	// 2. 识别当前操作系统
	currentOS := runtime.GOOS // "linux" or "windows"
	log.Printf("🚀 Janus Agent starting on [%s]...", strings.ToUpper(currentOS))

	// 3. 初始化 Redis
	if err := redis.InitRedis(&config.GlobalConfig.Redis); err != nil {
		log.Fatalf("❌ Redis 连接失败: %v", err)
	}
	defer redis.Close()
	log.Println("✅ Redis connected.")

	// 4. 初始化 Telegram
	if err := telegram.InitTelegram(&config.GlobalConfig.Telegram); err != nil {
		log.Printf("⚠️ Telegram 初始化失败: %v", err)
	} else {
		// 启动 Telegram Bot 命令处理
		go telegram.StartInlineKeyBoard()
	}

	// 5. 启动 HTTP 服务器
	if config.GlobalConfig.HTTP.Enabled {
		go startHTTPServer(currentOS)
		log.Printf("🌐 HTTP 服务器启动在端口 %d", config.GlobalConfig.HTTP.Port)
	}

	// 6. 发送上线通知
	osInfo := osinfo.GetCurrentOSInfo()
	telegram.SendMessage(fmt.Sprintf("🖥️ *Janus Online*\nOS: %s\nIP: %s\nUser: %s\nTime: %s",
		strings.ToUpper(osInfo.OS), osInfo.PrivateIP, osInfo.UserInfo, time.Now().Format("2006-01-02 15:04:05")))

	// 7. 启动心跳轮询
	interval := time.Duration(config.GlobalConfig.System.CheckInterval) * time.Second
	if interval == 0 {
		interval = 3 * time.Second // 默认值
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		checkAndExecute(currentOS)
	}
}

// checkAndExecute 检查 Redis 指令并执行
func checkAndExecute(osType string) {
	// 读取指令
	cmd, err := redis.Get(config.GlobalConfig.System.CommandKey)
	if err != nil {
		// redis.Nil 表示键不存在，这是正常情况
		if !errors.Is(err, redisv8.Nil) {
			log.Printf("⚠️ Error reading redis: %v", err)
		}
		return
	}

	log.Printf("📥 Received command: %s", cmd)

	// 收到指令后，立即删除 Redis 中的 Key，防止重复执行
	redis.Delete(config.GlobalConfig.System.CommandKey)

	switch cmd {
	case "shutdown":
		telegram.SendMessage(fmt.Sprintf("💤 *Shutting down* %s...", strings.ToUpper(osType)))
		performShutdown(osType)

	case "switch":
		// 切换系统逻辑
		if osType == "linux" {
			telegram.SendMessage("🔄 *Switching to Windows* (Next Boot)...")
			// Linux 切 Windows: 设置 grub-reboot -> 重启
			grubEntry := config.GlobalConfig.System.Linux.GrubWinEntry
			if err := runCmd("sudo", "grub-reboot", grubEntry); err != nil {
				telegram.SendMessage(fmt.Sprintf("❌ Grub error: %v", err))
				return
			}
			// 使用配置中的重启命令
			cmdParts := strings.Fields(config.GlobalConfig.System.Linux.RebootCmd)
			if len(cmdParts) > 0 {
				runCmd(cmdParts[0], cmdParts[1:]...)
			} else {
				runCmd("sudo", "reboot")
			}

		} else if osType == "windows" {
			telegram.SendMessage("🔄 *Switching to Linux* (Rebooting)...")
			// Windows 切 Linux: 使用配置中的重启命令
			cmdParts := strings.Fields(config.GlobalConfig.System.Windows.RebootCmd)
			if len(cmdParts) > 0 {
				runCmd(cmdParts[0], cmdParts[1:]...)
			} else {
				runCmd("shutdown", "/r", "/t", "0")
			}
		}

	default:
		log.Printf("❓ Unknown command: %s", cmd)
	}
}

// performShutdown 执行关机
func performShutdown(osType string) {
	if osType == "windows" {
		// Windows 关机
		cmdParts := strings.Fields(config.GlobalConfig.System.Windows.ShutdownCmd)
		if len(cmdParts) > 0 {
			runCmd(cmdParts[0], cmdParts[1:]...)
		} else {
			runCmd("shutdown", "/s", "/t", "0")
		}
	} else {
		// Linux 关机
		cmdParts := strings.Fields(config.GlobalConfig.System.Linux.ShutdownCmd)
		if len(cmdParts) > 0 {
			runCmd(cmdParts[0], cmdParts[1:]...)
		} else {
			runCmd("sudo", "shutdown", "-h", "now")
		}
	}
}

// runCmd 执行系统命令的封装
func runCmd(name string, args ...string) error {
	log.Printf("Executing: %s %s", name, strings.Join(args, " "))
	cmd := exec.Command(name, args...)
	return cmd.Run()
}

// sendCommand 发送命令到 Redis
func sendCommand(cmd string) error {
	return redis.Set(config.GlobalConfig.System.CommandKey, cmd, 0)
}

// startHTTPServer 启动 HTTP 服务器
func startHTTPServer(currentOS string) {
	http.HandleFunc("/command", func(w http.ResponseWriter, r *http.Request) {
		// 检查方法
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// 权限验证
		password := r.URL.Query().Get("password")
		if password != config.GlobalConfig.HTTP.Password {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// 解析请求体
		var req struct {
			Command string `json:"command"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// 验证命令
		validCommands := map[string]bool{
			"shutdown": true,
			"switch":   true,
		}
		if !validCommands[req.Command] {
			http.Error(w, "Invalid command", http.StatusBadRequest)
			return
		}

		// 发送命令到 Redis
		if err := sendCommand(req.Command); err != nil {
			http.Error(w, fmt.Sprintf("Failed to send command: %v", err), http.StatusInternalServerError)
			return
		}

		// 返回成功响应
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "success",
			"message": fmt.Sprintf("Command '%s' sent successfully", req.Command),
			"os":      strings.ToUpper(currentOS),
		})
	})

	// 健康检查端点
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "ok",
			"os":     strings.ToUpper(currentOS),
		})
	})

	port := config.GlobalConfig.HTTP.Port
	if port == 0 {
		port = 8080
	}

	if err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil); err != nil {
		log.Printf("❌ HTTP 服务器启动失败: %v", err)
	}
}

// getLocalIP 获取本地 IP 用于展示 (简单实现)
func getLocalIP() string {
	// 简单粗暴的方法，实际生产中可能需要遍历网卡
	// 这里为了代码简洁，暂不实现复杂的 IP 获取，仅返回占位符
	// 你可以在这里添加 net.InterfaceAddrs() 的逻辑
	return "Localhost"
}
