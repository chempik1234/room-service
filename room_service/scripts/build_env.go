package main

import (
	"flag"
	"fmt"
	"github.com/chempik1234/room-service/internal/config"
	"os"
	"reflect"
	"strings"
)

func createEnvLines(config any) []string {
	typ := reflect.TypeOf(config)
	val := reflect.ValueOf(config)

	var lines []string
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)

		envPrefix := field.Tag.Get("env-prefix")
		if envPrefix == "" {
			envPrefix = field.Tag.Get("env_prefix")
		}
		if envPrefix != "" {
			nestedLines := createEnvLines(val.Field(i).Interface())
			for _, nestedLine := range nestedLines {
				lines = append(lines, fmt.Sprintf("%s%s", envPrefix, nestedLine))
			}
			fmt.Println("created env-prefix:", envPrefix)
			continue
		}

		envName := field.Tag.Get("env")
		if envName == "" {
			fmt.Println("WARNING: no env tags for field", field.Name)
			envName = strings.ToUpper(field.Name)
		}
		envDefault := field.Tag.Get("env-default")
		if envDefault == "" {
			envDefault = field.Tag.Get("envDefault")
		}
		lines = append(lines, fmt.Sprintf("%s=\"%s\"", envName, envDefault))
	}

	return lines
}

func main() {
	envPathFlag := flag.String("filepath", "../config/.env", "env file path to be generated")
	flag.Parse()

	envPath := *envPathFlag

	file, err := os.Create(envPath)
	if err != nil {
		panic(fmt.Errorf("%w. Maybe you didn't create directory for file %s", err, envPath))
	}
	defer func() {
		err = file.Close()
		if err != nil {
			panic(err)
		}
	}()
	for _, line := range createEnvLines(config.Config{}) {
		if _, err := file.WriteString(line + "\n"); err != nil {
			panic(err)
		}
	}
}
