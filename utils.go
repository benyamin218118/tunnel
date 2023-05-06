package main

import (
	"encoding/base64"
	"math/rand"
	"time"
)

var letterRunes = []rune("1234567890abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func RandString(n int) string {
	b := make([]rune, n)
	rand.Seed(time.Now().UnixNano())
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func base64Encode(str string) string {
	return base64.StdEncoding.EncodeToString([]byte(str))
}

func panicIfErr(err error) {
	if err != nil {
		panic(err)
	}
}
