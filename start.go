package main

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"

	_ "github.com/go-sql-driver/mysql"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var db *sql.DB

func saveUser(chatID int64, username string) {
	_, err := db.Exec("INSERT INTO Users (chat_id, username) VALUES (?, ?) ON DUPLICATE KEY UPDATE username=VALUES(username)", chatID, username)
	if err != nil {
		log.Printf("Failed to save user: %v\n", err)
	} else {
		log.Printf("Succesfully added into database (%v | %s)\n", chatID, username)
	}
}

type userDb struct {
	ID       int
	ChatID   int64
	Username string
	Balance  float64
}

func getDbTeleg(chatID int64, username string) (userDb, error) {
	var dbRow userDb

	err := db.QueryRow("SELECT id, chat_id, username, balance FROM Users WHERE chat_id = ? AND username = ?",
		chatID, username).Scan(&dbRow.ID, &dbRow.ChatID, &dbRow.Username, &dbRow.Balance)
	if err != nil {
		if err == sql.ErrNoRows {
			return userDb{}, fmt.Errorf("нет данных для заданных критериев")
		}
		return userDb{}, err
	}

	return dbRow, nil
}

// conf.бот
const (
	botToken   = "6585541253:AAHXh-XKJQo-o_rXgVnt3Z9t51eT8Zfh1kc"
	isBotDebub = true
	botTimeout = 60
)

// conf.база_данных
const (
	usernameDb     = "root"
	passwordDb     = "12345"
	connectionType = "tcp"
	addressDb      = "127.0.0.1:3306"
	nameDb         = "telegrambot"
)

func main() {
	var err error

	// Подключение к базе данных
	dsn := (usernameDb + ":" + passwordDb + "@" + connectionType + "(" + addressDb + ")/" + nameDb)
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer db.Close()

	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = isBotDebub

	log.Printf("Авторизован как %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = botTimeout

	updates := bot.GetUpdatesChan(u)

	// кнопки
	inlineMainMenu := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Магазин 🎮", "shop_menu"),
			tgbotapi.NewInlineKeyboardButtonData("Кабинет 📄", "cabinet"),
		),
		// tgbotapi.NewInlineKeyboardRow(
		// 	tgbotapi.NewInlineKeyboardButtonData("FAQ ❗", "faq"),
		// 	tgbotapi.NewInlineKeyboardButtonData("Гарантии ✔", "guarantees"),
		// ),
		// tgbotapi.NewInlineKeyboardRow(
		// 	tgbotapi.NewInlineKeyboardButtonData("Отзывы 🗣", "reviews"),
		// 	tgbotapi.NewInlineKeyboardButtonData("Поддержка 🧑‍💼", "support"),
		// ),
	)

	inlineShopMenu := tgbotapi.NewInlineKeyboardMarkup(
		// tgbotapi.NewInlineKeyboardRow(
		// 	tgbotapi.NewInlineKeyboardButtonData("Roblox", "roblox_start"),
		// 	tgbotapi.NewInlineKeyboardButtonData("Steam", "steam_start"),
		// ),
		// tgbotapi.NewInlineKeyboardRow(
		// 	tgbotapi.NewInlineKeyboardButtonData("Spotify", "spotify_start"),
		// 	tgbotapi.NewInlineKeyboardButtonData("Brawl Stars", "brawl_start"),
		// ),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Назад", "main_menu"),
		),
	)

	inlineCabinet := tgbotapi.NewInlineKeyboardMarkup(
		// tgbotapi.NewInlineKeyboardRow(
		// 	tgbotapi.NewInlineKeyboardButtonData("Пополнить баланс 💸", "balance"),
		// ),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Назад", "main_menu"),
		),
	)

	for update := range updates {
		if update.Message != nil {
			// обработка команд
			switch update.Message.Text {
			case "/start":
				saveUser(update.Message.Chat.ID, update.Message.From.UserName)

				photoFile := tgbotapi.FilePath("other/title.jpg")
				msg := tgbotapi.NewPhoto(update.Message.Chat.ID, photoFile)
				msg.Caption = "Главное меню."
				msg.ReplyMarkup = inlineMainMenu
				_, err := bot.Send(msg)
				if err != nil {
					log.Panic(err)
				}
			default:
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Не знаю такую команду")
				_, err := bot.Send(msg)
				if err != nil {
					log.Panic(err)
				}
			}

		}

		if update.CallbackQuery != nil {
			callback := tgbotapi.NewCallback(update.CallbackQuery.ID, update.CallbackQuery.Data)
			if _, err := bot.Request(callback); err != nil {
				log.Panic(err)
			}

			if update.CallbackQuery != nil {
				var pagePhoto, pageTitle string
				var pageInline tgbotapi.InlineKeyboardMarkup

				// обработка нажатия на кнопки
				switch update.CallbackQuery.Data {
				case "main_menu":
					pageTitle = "Главное меню."
					pagePhoto = "other/title.jpg"
					pageInline = inlineMainMenu

				case "shop_menu":
					pageTitle = "Магазин."
					pagePhoto = "other/shop.jpg"
					pageInline = inlineShopMenu

				case "cabinet":
					dbData, err := getDbTeleg(update.CallbackQuery.From.ID, update.CallbackQuery.From.UserName)
					if err != nil {
						log.Println("Error retrieving data:", err)
						continue
					}
					balanceStr := strconv.FormatFloat(dbData.Balance, 'f', 2, 64)
					chatId := strconv.FormatInt(dbData.ChatID, 10)
					pageTitle = ("Привет,  " + update.CallbackQuery.From.UserName + "!\n" +
						"Ваш ID: " + chatId + "\n\n" +
						"Ваш Баланс: " + balanceStr + "\n")
					pagePhoto = "other/cabinet.webp"
					pageInline = inlineCabinet
				}
				photoFile := tgbotapi.FilePath(pagePhoto)

				m := tgbotapi.EditMessageMediaConfig{
					BaseEdit: tgbotapi.BaseEdit{
						ChatID:          update.CallbackQuery.Message.Chat.ID,
						MessageID:       update.CallbackQuery.Message.MessageID,
						InlineMessageID: update.CallbackQuery.InlineMessageID,
						ReplyMarkup:     &pageInline,
					},
					Media: tgbotapi.InputMediaPhoto{
						BaseInputMedia: tgbotapi.BaseInputMedia{
							Type:    "photo",
							Media:   photoFile,
							Caption: pageTitle,
						},
					},
				}

				_, err := bot.Request(m)
				if err != nil {
					log.Panic(err)
				}
			}
		}
	}
}
