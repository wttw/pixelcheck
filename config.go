package pixelcheck

import (
	"encoding/json"
	"log"
	"os"
)

type Config struct {
	DBConn    string `json:"connstring"`
	URL       string `json:"url"`
	Listen    string `json:"listen"`
	To        string `json:"to"`
	From      string `json:"from"`
	Smarthost string `json:"smarthost"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	Templates string `json:"template_dir"`
	Cert      string `json:"tlscert"`
	Key       string `json:"tlskey"`
	Image     string `json:"image"`
	ImageDir  string `json:"image_dir"`
}

func New(filename string) Config {
	c := Config{
		Listen:    ":8080",
		Templates: ".",
		ImageDir:  ".",
		Image:     "default.png",
	}
	c.Load(filename)
	return c
}

func (c *Config) Load(filename string) {
	f, err := os.Open(filename)
	if err != nil {
		log.Fatalf("Failed to open configuration file %s: %s", filename, err)
	}
	err = json.NewDecoder(f).Decode(c)
	if err != nil {
		log.Fatalf("Failed to parse configuration file %s: %s", filename, err)
	}
}
