package config

import (
	"fmt"
	"log"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Redis    RedisConfig    `mapstructure:"redis"`
	Telegram TelegramConfig `mapstructure:"telegram"`
	HTTP     HTTPConfig     `mapstructure:"http"`
	System   SystemConfig   `mapstructure:"system"`
	Log      LogConfig      `mapstructure:"log"`
}

type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type TelegramConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	BotToken string `mapstructure:"bot_token"`
	ChatID   string `mapstructure:"chat_id"`
}

type HTTPConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
}

type SystemConfig struct {
	CommandKey    string `mapstructure:"command_key"`
	CheckInterval int    `mapstructure:"check_interval"`
	Linux         struct {
		GrubWinEntry string `mapstructure:"grub_win_entry"`
		ShutdownCmd  string `mapstructure:"shutdown_cmd"`
		RebootCmd    string `mapstructure:"reboot_cmd"`
	} `mapstructure:"linux"`
	Windows struct {
		ShutdownCmd string `mapstructure:"shutdown_cmd"`
		RebootCmd   string `mapstructure:"reboot_cmd"`
	} `mapstructure:"windows"`
}

type LogConfig struct {
	Level      string `mapstructure:"level"`
	File       string `mapstructure:"file"`
	MaxSize    int    `mapstructure:"max_size"`
	MaxBackups int    `mapstructure:"max_backups"`
	MaxAge     int    `mapstructure:"max_age"`
	Compress   bool   `mapstructure:"compress"`
}

var (
	GlobalConfig = &Config{}
	v            *viper.Viper
)

func InitConfig(configPath string) error {
	v = viper.New()
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.AddConfigPath(".")
		v.AddConfigPath("./config")
		v.AddConfigPath("../config")
		v.SetConfigName("config")
		v.SetConfigType("yaml")
	}
	v.SetEnvPrefix("APP")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// 读取配置文件
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Println("配置文件未找到，运行失败")
		} else {
			return fmt.Errorf("读取配置文件失败：%w", err)
		}
	}

	// 解析配置到结构体
	GlobalConfig = &Config{}
	if err := v.Unmarshal(GlobalConfig); err != nil {
		return fmt.Errorf("解析配置失败：%w", err)
	}

	log.Printf("配置文件加载成功: %s", v.ConfigFileUsed())
	return nil
}

func Get(key string) interface{} {
	return v.Get(key)
}

// GetString 获取字符串配置
func GetString(key string) string {
	return v.GetString(key)
}

// GetInt 获取整数配置
func GetInt(key string) int {
	return v.GetInt(key)
}

// GetBool 获取布尔配置
func GetBool(key string) bool {
	return v.GetBool(key)
}

// IsSet 检查配置是否存在
func IsSet(key string) bool {
	return v.IsSet(key)
}
