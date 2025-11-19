package telegram

import (
	"fmt"
	"log"
	"runtime"
	"strings"
	"time"

	"janus/config"
	"janus/pkg/redis"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type TelegramService struct {
	bot     *tgbotapi.BotAPI
	chatID  int64
	enabled bool
}

var Service *TelegramService

// InitTelegram 初始化 Telegram 服务
func InitTelegram(cfg *config.TelegramConfig) error {
	if !cfg.Enabled {
		log.Println("Telegram 通知未启用")
		return nil
	}

	if cfg.BotToken == "" || cfg.ChatID == "" {
		return fmt.Errorf("Telegram 配置不完整")
	}

	bot, err := tgbotapi.NewBotAPI(cfg.BotToken)
	if err != nil {
		return fmt.Errorf("创建 Telegram bot 失败：%w", err)
	}

	// 解析 ChatID
	var chatID int64
	fmt.Sscanf(cfg.ChatID, "%d", &chatID)

	Service = &TelegramService{
		bot:     bot,
		chatID:  chatID,
		enabled: cfg.Enabled,
	}

	log.Printf("Telegram bot 初始化成功: @%s", bot.Self.UserName)
	return nil
}

// SendMessage 发送消息
func SendMessage(text string) error {
	if Service == nil || !Service.enabled {
		return nil
	}

	msg := tgbotapi.NewMessage(Service.chatID, text)
	msg.ParseMode = "Markdown"

	_, err := Service.bot.Send(msg)
	if err != nil {
		return fmt.Errorf("发送 Telegram 消息失败：%w", err)
	}

	return nil
}

// SendNotification 发送通知（带重试）
func SendNotification(title, content string) error {
	if Service == nil || !Service.enabled {
		return nil
	}

	message := fmt.Sprintf("*%s*\n\n%s\n\n_%s_",
		title,
		content,
		time.Now().Format("2006-01-02 15:04:05"))

	// 重试 3 次
	for i := 0; i < 3; i++ {
		if err := SendMessage(message); err != nil {
			log.Printf("发送通知失败 (尝试 %d/3): %v", i+1, err)
			if i < 2 {
				time.Sleep(time.Second * 2)
				continue
			}
			return err
		}
		break
	}

	return nil
}

// SendAlert 发送警报
func SendAlert(alert string) error {
	return SendNotification("⚠️ 警报", alert)
}

// SendInfo 发送信息
func SendInfo(info string) error {
	return SendNotification("ℹ️ 信息", info)
}

// SendSuccess 发送成功消息
func SendSuccess(message string) error {
	return SendNotification("✅ 成功", message)
}

// SendError 发送错误消息
func SendError(errMsg string) error {
	return SendNotification("❌ 错误", errMsg)
}

// sendCommandToRedis 发送命令到 Redis
func sendCommandToRedis(cmd string) error {
	return redis.Set(config.GlobalConfig.System.CommandKey, cmd, 0)
}

// StartCommandHandler 启动 Telegram Bot 命令处理
func StartCommandHandler() {
	if Service == nil || !Service.enabled {
		return
	}

	// 设置命令菜单
	commands := []tgbotapi.BotCommand{
		{
			Command:     "start",
			Description: "开始使用 Janus",
		},
		{
			Command:     "menu",
			Description: "显示命令菜单",
		},
		{
			Command:     "shutdown",
			Description: "关机",
		},
		{
			Command:     "switch",
			Description: "切换系统（Linux ↔ Windows）",
		},
		{
			Command:     "status",
			Description: "查看系统状态",
		},
	}

	_, err := Service.bot.Request(tgbotapi.NewSetMyCommands(commands...))
	if err != nil {
		log.Printf("⚠️ 设置命令菜单失败: %v", err)
	}

	// 配置更新
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := Service.bot.GetUpdatesChan(u)

	log.Println("🤖 Telegram Bot 命令处理器已启动")

	for update := range updates {
		if update.Message == nil {
			continue
		}

		// 检查是否是授权用户
		if !isAuthorizedUser(update.Message.From.ID) {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "❌ 未授权的用户")
			Service.bot.Send(msg)
			continue
		}

		// 处理命令
		command := update.Message.Command()
		text := update.Message.Text

		switch command {
		case "start", "menu":
			handleMenu(update.Message.Chat.ID)
		case "shutdown":
			handleShutdown(update.Message.Chat.ID)
		case "switch":
			handleSwitch(update.Message.Chat.ID)
		case "status":
			handleStatus(update.Message.Chat.ID)
		default:
			if strings.HasPrefix(text, "/") {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "❓ 未知命令，使用 /menu 查看可用命令")
				Service.bot.Send(msg)
			}
		}
	}
}

// isAuthorizedUser 检查用户是否授权
func isAuthorizedUser(userID int64) bool {
	// 检查用户 ID 是否匹配配置的 ChatID
	var configChatID int64
	fmt.Sscanf(config.GlobalConfig.Telegram.ChatID, "%d", &configChatID)
	return userID == configChatID
}

// handleMenu 处理菜单命令
func handleMenu(chatID int64) {
	currentOS := runtime.GOOS
	menuText := fmt.Sprintf(`🖥️ *Janus 控制面板*

*系统信息:*
• 操作系统: %s
• 状态: 运行中

*可用命令:*
/start - 开始使用
/menu - 显示此菜单
/shutdown - 关机
/switch - 切换系统 (Linux ↔ Windows)
/status - 查看系统状态

*使用说明:*
发送命令即可执行相应操作。`, strings.ToUpper(currentOS))

	msg := tgbotapi.NewMessage(chatID, menuText)
	msg.ParseMode = "Markdown"
	Service.bot.Send(msg)
}

// handleShutdown 处理关机命令
func handleShutdown(chatID int64) {
	if err := sendCommandToRedis("shutdown"); err != nil {
		msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("❌ 发送关机命令失败: %v", err))
		Service.bot.Send(msg)
		return
	}

	msg := tgbotapi.NewMessage(chatID, "💤 关机命令已发送，系统即将关机...")
	Service.bot.Send(msg)
}

// handleSwitch 处理切换系统命令
func handleSwitch(chatID int64) {
	if err := sendCommandToRedis("switch"); err != nil {
		msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("❌ 发送切换命令失败: %v", err))
		Service.bot.Send(msg)
		return
	}

	currentOS := runtime.GOOS
	var targetOS string
	if currentOS == "linux" {
		targetOS = "Windows"
	} else {
		targetOS = "Linux"
	}

	msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("🔄 切换系统命令已发送，下次启动将进入 %s...", targetOS))
	Service.bot.Send(msg)
}

// handleStatus 处理状态查询命令
func handleStatus(chatID int64) {
	currentOS := runtime.GOOS
	statusText := fmt.Sprintf(`📊 *系统状态*

*操作系统:* %s
*状态:* 运行中
*时间:* %s

系统正常运行中。`, strings.ToUpper(currentOS), time.Now().Format("2006-01-02 15:04:05"))

	msg := tgbotapi.NewMessage(chatID, statusText)
	msg.ParseMode = "Markdown"
	Service.bot.Send(msg)
}
