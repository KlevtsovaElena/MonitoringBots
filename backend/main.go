package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type ResponseT struct {
	Ok     bool `json:"ok"`
	Result []struct {
		UpdateID int `json:"update_id"`
		Message  struct {
			MessageID int `json:"message_id"`
			From      struct {
				ID           int    `json:"id"`
				IsBot        bool   `json:"is_bot"`
				FirstName    string `json:"first_name"`
				LastName     string `json:"last_name"`
				Username     string `json:"username"`
				LanguageCode string `json:"language_code"`
			} `json:"from"`
			Chat struct {
				ID        int    `json:"id"`
				FirstName string `json:"first_name"`
				Type      string `json:"type"`
			} `json:"chat"`
			Date int    `json:"date"`
			Text string `json:"text"`
		} `json:"message"`
	} `json:"result"`
}

type UserT struct {
	ID        int
	Username  string
	FirstName string
	LastName  string
	ReqDate   int
}

type MessageT struct {
	UserID  int
	Content string
	Date    int
}

func (m *MessageT) addMessage(text string, messageTime int, chatId int) {
	m.UserID = chatId
	m.Content = text
	m.Date = messageTime
}

var host string = "https://api.telegram.org/bot"

var tokens = []string{
	"6131123688:AAGV7bDvX4aX4_n-ShaiKjXlpUvlnfXsQFY",
	"6266036859:AAGLaQvcjIR8BgkymXNwP0rSfqx2lzQvdmA",
	"6114246715:AAHeEIQBYooYdGG-Dgjqv0jLxPH6zxGJRNY",
	"6089892871:AAHBVa5OpNIg0WYzvIDXj7x8nWqX3n0h6EQ",
	"6025286750:AAHWYyfw1g4-QCP6iopsR5xkMprILA3vdkI",
}

var lastMessage = []int{0, 0, 0, 0, 0}

var importansWord = []string{
	"срочно",
	"помогите",
	"помощь",
	"помочь",
	"важно",
	"конфликт",
	"неприятн",
	"паник",
	"sos",
}

// создали базу данных сообщений
var MessagesDB = []MessageT{}

// создали базу данных юзеров
var UsersDB = make(map[int]UserT)

// создаем соединение с БД
var Db, Err = sql.Open("mysql", "root:nordic123@tcp(mysql:3306)/inordic")

func main() {

	// проверка подключились ли к БД
	if Err != nil {
		fmt.Println("НЕ подключились к БД", Err)
	}

	//получили юзеров из базы в оперативную память
	rows, err := Db.Query("select * from `users`")
	if err != nil {
		fmt.Println("Не удалось получить юзеров", err)
	}

	// запишем юзеров в оперативку
	for rows.Next() {
		u := UserT{}
		err := rows.Scan(&u.ID, &u.Username, &u.FirstName, &u.LastName, &u.ReqDate)
		if err != nil {
			fmt.Println("ошибка при считывании юзера в u", err)
			continue
		}
		UsersDB[u.ID] = u
	}
	fmt.Println(UsersDB)

	for range time.Tick(time.Second * 1) {
		//отправляем запрос к Telegram API на получение сообщений длЯ каждого бота
		for j := 0; j < len(tokens); j++ {
			handleBot(j)
		}
	}

}

// функция отправки сообщения пользователю
func sendMessage(chatId int, text string, token string) {
	http.Get(host + token + "/sendMessage?chat_id=" + strconv.Itoa(chatId) + "&text=" + text)
}

// функция проверки на важные слова
func checkImportant(text string) bool {
	for i := 0; i < len(importansWord); i++ {
		if strings.Contains(strings.ToLower(text), importansWord[i]) {
			return true
		}
	}
	return false
}

// функция обработки обращений
func handleBot(j int) {

	var url string = host + tokens[j] + "/getUpdates?offset=" + strconv.Itoa(lastMessage[j])
	// отправляем запрос на url, получим новые сообщения бота
	response, err := http.Get(url)
	if err != nil {
		fmt.Println("Ошибка отправки запроса на апи телеги", err)
	}
	// считываем тело запроса (получаем Json в виде среза байт)
	data, _ := io.ReadAll(response.Body)

	//парсим данные из json в структуру
	var responseObj ResponseT
	json.Unmarshal(data, &responseObj)

	//считаем количество новых сообщений
	number := len(responseObj.Result)

	//если сообщений нет - то дальше код не выполняем
	if number < 1 {
		return
	}

	fmt.Println("сообщения из ", tokens[j], "всего", number)

	//в цикле достанем инормацию по каждому сообщению
	for i := 0; i < number; i++ {

		text := responseObj.Result[i].Message.Text
		chatId := responseObj.Result[i].Message.From.ID
		messageTime := responseObj.Result[i].Message.Date
		username := responseObj.Result[i].Message.From.Username
		firstName := responseObj.Result[i].Message.From.FirstName
		lastName := responseObj.Result[i].Message.From.LastName

		//определяем зарегистрирован ли пользователь
		_, exist := UsersDB[chatId]

		// если не зарегистрирован
		if exist == false {
			// собираем инфу по юзеру
			user := UserT{}
			user.ID = chatId
			user.Username = username
			user.FirstName = firstName
			user.LastName = lastName
			user.ReqDate = messageTime

			//если не зарегистрирован - добавляем в БД и сохраняем в ОП
			_, err := Db.Query("INSERT INTO `users`(`id`,`username`,`first_name`,`last_name`, `date_req`) VALUES(?,?, ?, ?,?)",
				chatId, username, firstName, lastName, messageTime)
			if err != nil {
				fmt.Println("Ошибка сохранения пользователя ", err)
			} else {
				fmt.Println("пользователь добавлен")
			}

			// записываем в оперативку
			UsersDB[chatId] = user
		}

		//проверим сообщение на пустоту
		if text == "" {
			continue
		}
		fmt.Println("непустое сообщение")

		is_important := 0
		if checkImportant(text) {
			is_important = 1
		}

		//запись сообщений в БД
		_, err := Db.Query("INSERT INTO `messages`(`user_id`,`content`,`c_time`, `bot_id`, `is_important`) VALUES(?,?, ?,?,?)",
			chatId, text, messageTime, j+1, is_important)
		if err != nil {
			fmt.Println("Ошибка сохранения сообщения ", err)
		} else {
			fmt.Println("сообщение " + text + " добавлено")
		}

		//отвечаем пользователю на его сообщение
		go sendMessage(chatId, text, tokens[j])

	}

	//запоминаем update_id  последнего сообщения для бота
	lastMessage[j] = responseObj.Result[number-1].UpdateID + 1
	fmt.Println(MessagesDB)
}
