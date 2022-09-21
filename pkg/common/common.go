package common

import (
	"encoding/json"
	"io"
	"log"
	"math/rand"
	"net/http"

	"golang.org/x/crypto/argon2"
)

type Msg struct {
	Message string `json:"message"`
}

func WriteMsg(w http.ResponseWriter, msg string, code int) {
	w.WriteHeader(code)
	WriteRespJSON(w, Msg{msg})
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func RandStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func HashPass(plainPassword, salt string) []byte {
	hashedPass := argon2.IDKey([]byte(plainPassword), []byte(salt), 1, 64*1024, 4, 32)
	res := []byte(salt)
	return append(res, hashedPass...)
}

func ParseReqBody(body io.Reader, ptr interface{}) error {
	err := json.NewDecoder(body).Decode(ptr)
	if err != nil {
		return err
	}
	return nil
}

func WriteRespJSON(w http.ResponseWriter, data interface{}) {
	resp, err := json.Marshal(data)
	if err != nil {
		log.Println("common: JSON marshaling failed", err)
		WriteMsg(w, "response failed", http.StatusInternalServerError)
		return
	}

	_, err = w.Write(resp)
	if err != nil {
		log.Println("common: failed writing response", err)
	}
}
