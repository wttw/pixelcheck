package pixelcheck

import (
	"log"
	"os"

	"gopkg.in/yaml.v2"
)

type Config struct {
	DBConn    string `yaml:"connstring"`
	URL       string `yaml:"url"`
	Listen    string `yaml:"listen"`
	To        string `yaml:"to"`
	From      string `yaml:"from"`
	Smarthost string `yaml:"smarthost"`
	Username  string `yaml:"username"`
	Password  string `yaml:"password"`
	Templates string `yaml:"template_dir"`
	Cert      string `yaml:"tlscert"`
	Key       string `yaml:"tlskey"`
	Image     string `yaml:"image"`
	ImageDir  string `yaml:"image_dir"`
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
	err = yaml.NewDecoder(f).Decode(c)
	if err != nil {
		log.Fatalf("Failed to parse configuration file %s: %s", filename, err)
	}
}
