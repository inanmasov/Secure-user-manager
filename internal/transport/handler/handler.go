package handler

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"example.com/Go/internal/database"
	_ "github.com/lib/pq"
	"gopkg.in/gomail.v2"
)

type Person struct {
	Id       int    `json:"id"`
	Mail     string `json:"mail"`
	Password string `json:"password"`
	Code     int    `json:"second_factor"`
}

func generateCode() int {
	rand.Seed(time.Now().UnixNano())

	return rand.Intn(900000) + 100000
}

func getPeople(w http.ResponseWriter, r *http.Request) {
	var person Person
	err := json.NewDecoder(r.Body).Decode(&person)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	db, err := database.Initialize()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	result, err := db.DB.Query("select id from Person where mail = $1", person.Mail)
	if err != nil {
		panic(err)
	}
	defer result.Close()

	if result.Next() {
		var id int
		if err := result.Scan(&id); err != nil {
			http.Error(w, "Error scanning result", http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(id)
	} else {
		fmt.Fprintf(w, "Person not found")
	}
}

func postPeople(w http.ResponseWriter, r *http.Request) {
	var person Person
	err := json.NewDecoder(r.Body).Decode(&person)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if person.Code != 0 {
		db, err := database.Initialize()
		if err != nil {
			log.Fatal(err)
		}
		defer db.Close()

		result, err := db.DB.Query("select * from PersonTmp where second_factor = $1", person.Code)
		if err != nil {
			panic(err)
		}
		defer result.Close()

		var personTmp Person

		if result.Next() {
			if err := result.Scan(&personTmp.Mail, &personTmp.Password, &personTmp.Code); err != nil {
				http.Error(w, "Error scanning result", http.StatusInternalServerError)
				return
			}
		} else {
			fmt.Fprintf(w, "Person not found")
			return
		}

		hasher := sha256.New()
		hasher.Write([]byte(person.Password))
		hashInBytes := hasher.Sum(nil)
		hashPsw := hex.EncodeToString(hashInBytes)

		if person.Mail == personTmp.Mail && hashPsw == personTmp.Password && person.Code == personTmp.Code {
			db, err := database.Initialize()
			if err != nil {
				log.Fatal(err)
			}
			defer db.Close()

			result, err := db.DB.Exec("insert into Person (mail, password) values ($1, $2)", person.Mail, hashPsw)
			if err != nil {
				fmt.Fprintf(w, "Error create person: "+fmt.Sprint(err))
				return
			}

			num, err := result.RowsAffected()
			if err != nil {
				fmt.Fprintf(w, "Error adding person")
				return
			}
			fmt.Fprintf(w, "User %d added!", num)

			_, err = db.DB.Exec("delete from PersonTmp where second_factor = $1", person.Code)
			if err != nil {
				log.Printf("Failed to remove user %s from temporary table", person.Mail)
				return
			}
		} else {
			fmt.Fprintf(w, "Error adding person")
			return
		}
		return
	}

	code := generateCode()
	mailer := gomail.NewMessage()
	mailer.SetHeader("From", person.Mail)
	mailer.SetHeader("To", person.Mail)
	mailer.SetHeader("Subject", "Second factor")
	mailer.SetBody("text/plain", strconv.Itoa(code))

	dialer := gomail.NewDialer(
		"smtp.gmail.com",
		587,
		person.Mail,
		"wzbowvynfkkskctz",
	)

	err = dialer.DialAndSend(mailer)
	if err != nil {
		fmt.Fprintf(w, "Error send second factor: "+fmt.Sprint(err))
		return
	}

	hasher := sha256.New()
	hasher.Write([]byte(person.Password))
	hashInBytes := hasher.Sum(nil)
	hashPsw := hex.EncodeToString(hashInBytes)

	db, err := database.Initialize()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	result, err := db.DB.Exec("insert into PersonTmp (mail, password, second_factor) values ($1, $2, $3)", person.Mail, hashPsw, code)
	if err != nil {
		fmt.Fprintf(w, "Error create person: "+fmt.Sprint(err))
		return
	}

	_, err = result.RowsAffected()
	if err != nil {
		fmt.Fprintf(w, "Error adding person")
		return
	}
	fmt.Fprintf(w, "Enter the code sent to the specified email")
}

func updatePeople(w http.ResponseWriter, r *http.Request) {
	var person Person
	err := json.NewDecoder(r.Body).Decode(&person)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if person.Mail != "" && person.Password != "" {
		hasher := sha256.New()
		hasher.Write([]byte(person.Password))
		hashInBytes := hasher.Sum(nil)
		hashPsw := hex.EncodeToString(hashInBytes)

		db, err := database.Initialize()
		if err != nil {
			log.Fatal(err)
		}
		defer db.Close()

		result, err := db.DB.Exec("update Person set mail = $1, password = $2 where id = $3", person.Mail, hashPsw, person.Id)
		if err != nil {
			fmt.Fprintf(w, "Error update person: "+fmt.Sprint(err))
			return
		}
		num, err := result.RowsAffected()
		if err != nil {
			fmt.Fprintf(w, "Error update person")
			return
		}

		fmt.Fprintf(w, fmt.Sprint(num)+" line updated")
	} else if person.Mail != "" {
		db, err := database.Initialize()
		if err != nil {
			log.Fatal(err)
		}
		defer db.Close()

		result, err := db.DB.Exec("update Person set mail = $1 where id = $2", person.Mail, person.Id)
		if err != nil {
			fmt.Fprintf(w, "Error update person: "+fmt.Sprint(err))
			return
		}
		num, err := result.RowsAffected()
		if err != nil {
			fmt.Fprintf(w, "Error update person")
			return
		}

		fmt.Fprintf(w, fmt.Sprint(num)+" line updated")
	} else if person.Password != "" {
		hasher := sha256.New()
		hasher.Write([]byte(person.Password))
		hashInBytes := hasher.Sum(nil)
		hashPsw := hex.EncodeToString(hashInBytes)

		db, err := database.Initialize()
		if err != nil {
			log.Fatal(err)
		}
		defer db.Close()

		result, err := db.DB.Exec("update Person set password = $1 where id = $2", hashPsw, person.Id)
		if err != nil {
			fmt.Fprintf(w, "Error update person: "+fmt.Sprint(err))
			return
		}
		num, err := result.RowsAffected()
		if err != nil {
			fmt.Fprintf(w, "Error update person")
			return
		}

		fmt.Fprintf(w, fmt.Sprint(num)+" line updated")
	} else {
		fmt.Fprintf(w, "Error input")
	}
}

func deletePeople(w http.ResponseWriter, r *http.Request) {
	var person Person
	err := json.NewDecoder(r.Body).Decode(&person)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	db, err := database.Initialize()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	result, err := db.DB.Exec("delete from Person where mail = $1", person.Mail)
	if err != nil {
		fmt.Fprintf(w, "Error delete person: "+fmt.Sprint(err))
		return
	}

	num, err := result.RowsAffected()
	if err != nil {
		fmt.Fprintf(w, "Error deleted person")
	}

	fmt.Fprintf(w, fmt.Sprint(num)+" line deleted")
}

func PeopleHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		getPeople(w, r)
	case http.MethodPost:
		postPeople(w, r)
	case http.MethodPatch:
		updatePeople(w, r)
	case http.MethodDelete:
		deletePeople(w, r)
	default:
		http.Error(w, "Invalid http method", http.StatusMethodNotAllowed)
	}
}

func HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "http web-server works correctly")
}
