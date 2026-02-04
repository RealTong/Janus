package telegram

import (
	"fmt"
	"log"
	"runtime"
	"strings"
	"time"

	"janus/config"
	"janus/pkg/osinfo"
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

// SendStartMenu 发送带键盘菜单的上线通知
func SendStartMenu() error {
	if Service == nil || !Service.enabled {
		return nil
	}

	osInfo := osinfo.GetCurrentOSInfo()
	targetOS := "Windows"
	if osInfo.OS != "linux" {
		targetOS = "Linux"
	}

	text := fmt.Sprintf("🟢 *Janus Online*\n\n*Current System Info:*\n• OS: %s\n• Status: 🟢 Running\n• Private IP: %s\n• User: %s\n• Time: %s",
		strings.ToUpper(osInfo.OS),
		osInfo.PrivateIP,
		osInfo.UserInfo,
		time.Now().Format("2006-01-02 15:04:05"))

	msg := tgbotapi.NewMessage(Service.chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("🛑 关机(%s)", strings.ToUpper(osInfo.OS)), "shutdown"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("🔄 切换到(%s)", targetOS), "switch"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📊 查看系统状态", "status"),
		),
	)

	_, err := Service.bot.Send(msg)
	if err != nil {
		return fmt.Errorf("发送上线通知失败：%w", err)
	}

	log.Println("✅ 上线通知已发送")
	return nil
}

// StartInlineKeyBoard 启动内联键盘
func StartInlineKeyBoard() error {
	if Service == nil || !Service.enabled {
		return nil
	}
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := Service.bot.GetUpdatesChan(u)

	commands := []tgbotapi.BotCommand{
		{
			Command:     "start",
			Description: "开始使用 Janus，显示交互式命令菜单，使用 /help 查看帮助",
		},
		{
			Command:     "help",
			Description: "显示帮助信息",
		},
	}

	_, err := Service.bot.Request(tgbotapi.NewSetMyCommands(commands...))
	if err != nil {
		log.Printf("⚠️ Bot 命令菜单设置失败: %v", err)
	}

	for update := range updates {
		if update.Message != nil && !isAuthorizedUser(update.Message.From.ID) {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "❌ 未授权的用户")
			Service.bot.Send(msg)
			continue
		}
		if update.Message != nil {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, update.Message.Text)
			switch update.Message.Command() {
			case "start", "menu":
				osInfo := osinfo.GetCurrentOSInfo()
				targetOS := ""
				if osInfo.OS == "linux" {
					targetOS = "Windows"
				} else {
					targetOS = "Linux"
				}
				msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("🛑 关机(%s)", strings.ToUpper(osInfo.OS)), "shutdown"),
					),
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("🔄 切换到(%s)", targetOS), "switch"),
					),
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData("📊 查看系统状态", "status"),
					),
				)

				msg.Text = fmt.Sprintf("🖥️ *Welcome to Janus Control Panel*\n\n*Current System Info:*\n• OS: %s\n• Status: %s\n• Private IP: %s\n• User: %s", strings.ToUpper(osInfo.OS), "🟢 Running", osInfo.PrivateIP, osInfo.UserInfo)
				msg.ParseMode = "Markdown"
				if _, err := Service.bot.Send(msg); err != nil {
					panic(err)
				}
			case "help":
				msg.Text = "🤖 *Janus 帮助*\n\n*命令:*\n• /start - 开始使用 Janus，显示交互式命令菜单\n• /help - 显示帮助信息"
				if _, err := Service.bot.Send(msg); err != nil {
					panic(err)
				}
			default:
				msg.Text = "❓ 未知命令，使用 /help 查看帮助"
				if _, err := Service.bot.Send(msg); err != nil {
					panic(err)
				}
			}

		} else if update.CallbackQuery != nil {
			switch update.CallbackQuery.Data {
			case "shutdown":
				handleShutdown(update.CallbackQuery.Message.Chat.ID)
			case "switch":
				handleSwitch(update.CallbackQuery.Message.Chat.ID)
			case "status":
				handleStatus(update.CallbackQuery.Message.Chat.ID)
			}
		}
	}
	return nil
}

// isAuthorizedUser 检查用户是否授权
func isAuthorizedUser(userID int64) bool {
	// 检查用户 ID 是否匹配配置的 ChatID
	var configChatID int64
	fmt.Sscanf(config.GlobalConfig.Telegram.ChatID, "%d", &configChatID)
	return userID == configChatID
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
