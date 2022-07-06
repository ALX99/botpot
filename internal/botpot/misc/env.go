package misc

import (
	"fmt"
	"os"
	"strconv"
)

func GetEnv(key string) string {
	val := os.Getenv(key)
	if val == "" {
		panic(fmt.Sprintf("env variable %s is empty", key))
	}
	return val
}

func GetEnvInt(key string) int {
	val := os.Getenv(key)
	value, err := strconv.Atoi(val)
	if err != nil {
		panic(err)
	}
	return value
}
