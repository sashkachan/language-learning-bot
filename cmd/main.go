package main

import (
	"language-learning-bot/cmd/server"
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

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Run langekko as a server",
	Run: func(cmd *cobra.Command, args []string) {
		server.StartServer()
	},
}

func Execute() {
	rootCmd.AddCommand(telegramCmd)
	rootCmd.AddCommand(serverCmd)
	rootCmd.Execute()
}

func main() {
	Execute()
}
