# Language Learning Telegram Bot

This Telegram bot is designed to assist users in learning new languages, with initial support for Dutch and Russian. It leverages the OpenAI API to provide examples, translations, and pronunciation for given words. The bot also helps with understanding various grammar aspects of words.

## Features

- **Language Selection:** Users can choose a language to start learning.
- **Word Usage Exploration:** Offers examples, translations, and pronunciation of a given word.
- **Grammar Assistance:** Provides insights into grammar aspects of words, such as verb conjugations.
- **User Interaction Recording:** Records words and selections in a SQLite database to minimize repeated API requests.

## Configuration

The bot's settings are managed through the `.env` file, which includes configurations like the OpenAI API prompt template.

## Database

User interactions are stored in a SQLite database, allowing for efficient retrieval and minimizing redundant API calls.

## Getting Started

To run the bot:

1. Ensure Go is installed on your system.
2. Set up a SQLite database with the necessary schema.
3. Copy `.env.example` to `.env` and modify the required variables.
4. Run the bot using `go run main.go`.

## License

This project is licensed under the MIT License - see the [LICENSE.md](LICENSE.md) file for details.