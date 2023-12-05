package main

import (
	"language-learning-bot/cmd/telegram"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "langekko",
	Short: "langekko is a language learning bot",
	Run: func(cmd *cobra.Command, args []string) {
		telegram.StartTelegramBot()
	},
}

var telegramCmd = &cobra.Command{
	Use:   "telegram",
	Short: "Run langekko as a Telegram bot",
	Run: func(cmd *cobra.Command, args []string) {
		telegram.StartTelegramBot()
	},
}

func Execute() {
	rootCmd.AddCommand(telegramCmd)
	rootCmd.Execute()
}

func main() {
	Execute()
}
