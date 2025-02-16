package main

import "os"

func GetenvOrDefault(name string, defaultValue string) string {
	value, present := os.LookupEnv(name)
	if present == false {
		return defaultValue
	}
	return value
}
