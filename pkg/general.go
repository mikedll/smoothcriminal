
package pkg

import (
	"os"
	"log"
	"github.com/joho/godotenv"
)

var Debug = false
var Env string

func fileExists(filename string) bool {
    info, err := os.Stat(filename)
    if os.IsNotExist(err) {
        return false
    }
    return !info.IsDir()
}

func Init() {

	if(fileExists(".env")) {
		loadErr := godotenv.Load()
		if loadErr != nil {
			log.Fatal("Error loading .env file")
		}
	}

	Debug = os.Getenv("DEBUG") == "true"
	Env = os.Getenv("APP_ENV")
	
}
