package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	// "os"
	"strconv"

	_ "github.com/go-sql-driver/mysql"
)

type UserT struct {
	ID        int    `json:"id"`
	Username  string `json:"username"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	ReqDate   int    `json:"date_req"`
}

type ApiBotsT struct {
	UsersCount    int
	MessagesCount int
	BotsContents  []BotsContentT
}

type BotsContentT struct {
	Name     string
	Messages []MessagesBotT
}

type MessagesBotT struct {
	UserId      int
	Username    string
	FirstName   string
	LastName    string
	Content     string
	DateTime    int
	IsImportant int8
}

type BotsMySqlT struct {
	Id        int
	Bot_id    string
	Name      string
	Is_active int
}

type countMessagesT struct {
	CountMessage int
	CountUser    int
}

var UsersDB = make(map[int]UserT)

func main() {

	// начальное количество сообщений и юзеров устанавливаем в ноль
	CountMessages := countMessagesT{CountMessage: 0, CountUser: 0}

	ApiBots := ApiBotsT{}

	//подключение к БД
	db, err := sql.Open("mysql", "root:nordic123@tcp(mysql:3306)/inordic")
	if err != nil {
		fmt.Println("НЕ подключились к БД", err)
	}
	fmt.Println("подключились к БД")

	//делаем запросы к базе, чтобы получить данные и построить API
	// **** ДАННЫЕ ЮЗЕРОВ **** //
	// записываем общее кол-во юзеров
	countRow := db.QueryRow("select count(id) from `users`")
	fmt.Println("получили кол-во юзеров")

	err = countRow.Scan(&CountMessages.CountUser)
	if err != nil {
		fmt.Println("ошибка записи количества юзеров", err)
	}
	fmt.Println(CountMessages.CountUser)

	// **** ДАННЫЕ БОТОВ **** //
	// получим данные БОТОВ
	rows, err := db.Query("select * from bots")
	if err != nil {
		fmt.Println("Не смогли получить ботов из БД", err)
	}
	fmt.Println("получили ботов")
	defer rows.Close()

	bots := []BotsMySqlT{}

	// разнесем по полям и соберём их всех в переменную bots
	for rows.Next() {
		b := BotsMySqlT{}
		err := rows.Scan(&b.Id, &b.Bot_id, &b.Name, &b.Is_active)
		if err != nil {
			fmt.Println("ошибка сканирования в BotsMySql данных одного бота", err)
			continue
		}
		bots = append(bots, b)
	}
	fmt.Println("собрали bots", bots)

	// выводим нашу апишку при запросе с указанного адреса
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		ApiBots.UsersCount = CountMessages.CountUser
		ApiBots.MessagesCount = 0

		//переменная, куда будем собирать апи
		botsAPI := []BotsContentT{}

		//теперь для каждого бота соберём его сообщения
		for i := 0; i < len(bots); i++ {

			//сначала возьмём имя бота
			bot := BotsContentT{}
			bot.Name = bots[i].Name

			//сделаем запрос в базу по bot_id
			rows, err := db.Query("select messages.user_id, users.username, users.first_name, users.last_name, messages.content, messages.c_time, messages.is_important from messages, users where users.id=messages.user_id and bot_id=?", bots[i].Id)
			if err != nil {
				fmt.Println("Не смогли получить сообщения бота", err)
			}
			defer rows.Close()

			//соберём сообщения этого бота в апишку
			for rows.Next() {
				m := MessagesBotT{}
				err := rows.Scan(&m.UserId, &m.Username, &m.FirstName, &m.LastName, &m.Content, &m.DateTime, &m.IsImportant)
				if err != nil {
					fmt.Println(err)
					continue
				}

				//если юзернейм пустой, то берём данные из другого поля и дублируем их в юзернейм
				if m.Username == "" {
					if m.FirstName != "" {
						m.Username = m.FirstName
					} else if m.LastName != "" {
						m.Username = m.LastName
					} else {
						m.Username = strconv.Itoa(m.UserId)
					}
				}
				bot.Messages = append(bot.Messages, m)
			}
			ApiBots.MessagesCount += len(bot.Messages)
			botsAPI = append(botsAPI, bot)
		}
		ApiBots.BotsContents = botsAPI

		// сверим изменилось ли кол-во сообщений
		// если изменилось, то посмотрим изменилось ли кол-во юзеров
		if CountMessages.CountMessage != ApiBots.MessagesCount {
			fmt.Println("кол-во сообщений не совпадает")
			countRow = db.QueryRow("select count(id) from `users`")
			err = countRow.Scan(&CountMessages.CountUser)
			if err != nil {
				fmt.Println("ошибка записи количества юзеров", err)
			}
			ApiBots.UsersCount = CountMessages.CountUser
			CountMessages.CountMessage = ApiBots.MessagesCount
		}

		//закодируем в json данные, чтобы выдавать в апи
		JsonBotsAPI, _ := json.Marshal(ApiBots)

		//разрешим подключаться из браузера
		w.Header().Set("Access-Control-Allow-Origin", "*")

		//выдаём апишку
		fmt.Fprintf(w, string(JsonBotsAPI))

	})
	http.ListenAndServe(":80", nil)

}
