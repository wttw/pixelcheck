package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net"
	"net/smtp"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	flag "github.com/spf13/pflag"
	"github.com/wttw/pixelcheck"
)

const appName = "pixelsend"

const alphabet = "abcdefghijklmnopqrstuvwxyz"

func main() {
	var configFile string
	var recipient string
	var from string
	var tplName string
	var imageID int
	var imageFile string
	var note string
	helpRequest := false

	flag.StringVar(&configFile, "config", "pixelcheck.yaml", "Alternate configuration file")
	flag.StringVar(&recipient, "to", "", "Recipient")
	flag.StringVar(&from, "from", "", "822.From")
	flag.StringVar(&tplName, "template", "default", "Template for message")
	flag.IntVar(&imageID, "image", 0, "Reuse previous image if set")
	flag.StringVar(&imageFile, "imagefile", "", "Serve this image file")
	flag.StringVar(&note, "note", "", "Mark the mail sent with this note")
	flag.BoolVarP(&helpRequest, "help", "h", false, "Display brief help")

	flag.Parse()
	if helpRequest {
		_, _ = fmt.Fprintf(os.Stderr, "Usage of %s:\n", appName)
		flag.PrintDefaults()
		os.Exit(0)
	}

	c := pixelcheck.New(configFile)

	// Use some defaults from our config file
	if recipient == "" {
		recipient = c.To
	}
	if from == "" {
		from = c.From
	}
	if imageFile == "" {
		imageFile = c.Image
	}

	// We need a database
	db, err := pgxpool.Connect(context.Background(), c.DBConn)
	if err != nil {
		log.Fatal(err)
	}

	// And we need a message template
	tplPath := filepath.Join(c.Templates, strings.TrimSuffix(tplName, ".tpl")+".tpl")
	emailTpl, err := template.ParseFiles(tplPath)
	if err != nil {
		log.Fatalf("Failed to open message template: %s", err)
	}

	var imageURL string
	if imageID == 0 {
		// Create a new tracking image
		imageURL, imageID = newImage(c, db, imageFile)
	} else {
		// Get existing image
		var u *string
		err = db.QueryRow(context.Background(), `select url from image where id=$s`, imageID).Scan(&u)
		if err != nil {
			log.Fatalf("Failed to retrieve image %d: %s", imageID, err)
		}
		if u == nil {
			log.Fatalf("Image %d isn't valid", imageID)
		}
		imageURL = *u
	}

	var payload bytes.Buffer
	err = emailTpl.Execute(&payload, struct {
		Date     string
		To       string
		From     string
		Note     string
		Image    string
		Filename string
	}{
		Date:     time.Now().Format(time.RFC822Z),
		To:       recipient,
		From:     from,
		Note:     note,
		Image:    imageURL,
		Filename: imageFile,
	})
	if err != nil {
		log.Fatalf("Failed to build message: %s", err)
	}

	var mailID int
	// language=SQL
	err = db.QueryRow(context.Background(), `insert into mail (recipient, sender, body, template, image, note) VALUES ($1, $2, $3, $4, $5, $6) returning id`, recipient, from, payload.String(), tplName, imageID, note).Scan(&mailID) //nolint:lll
	if err != nil {
		log.Fatalf("Failed to insert message into DB: %s", err)
	}

	// Convert to CRLF line endings, because we've read RFC 821
	var enc []byte
	for _, c := range payload.Bytes() {
		switch c {
		case '\n':
			enc = append(enc, '\r', '\n')
		case '\r':
		default:
			enc = append(enc, c)
		}
	}
	host, _, err := net.SplitHostPort(c.Smarthost)
	if err != nil {
		log.Fatalf("Malformed smarthost string: %s", err)
	}

	// fmt.Printf("Sending mail...\n%s\n", string(enc))
	auth := smtp.PlainAuth("", c.Username, c.Password, host)
	err = smtp.SendMail(c.Smarthost, auth, from, []string{recipient}, enc)
	if err != nil {
		log.Fatalf("Failed to send email: %s", err)
	}

	log.Printf("Mail %d containing image %d (%s) sent", mailID, imageID, imageURL)
}

func newImage(c pixelcheck.Config, db *pgxpool.Pool, imageFile string) (string, int) {
	// Create a new tracking image
	urlTemplate, err := template.New("path").Parse(c.URL)
	if err != nil {
		log.Fatalf("Failed to parse urlString template: %s", err)
	}
	if _, err := os.Stat(filepath.Join(c.ImageDir, imageFile)); os.IsNotExist(err) {
		log.Fatalf("Image file %s doesn't exist", imageFile)
	}
	var imageID int
	// language=SQL
	err = db.QueryRow(context.Background(), `insert into image (file) values ($1) returning id`, imageFile).Scan(&imageID)
	if err != nil {
		log.Fatalf("Failed to create image: %s", err)
	}

	// let's make some base26
	encoded := ""
	n := imageID
	for n > 0 {
		remainder := n % len(alphabet)
		encoded = alphabet[remainder:remainder+1] + encoded
		n /= len(alphabet)
	}
	var urlString bytes.Buffer
	err = urlTemplate.Execute(&urlString,
		struct {
			ImageID     int
			ImageCookie string
			ImageFile   string
		}{
			ImageID:     imageID,
			ImageCookie: encoded,
			ImageFile:   imageFile,
		})
	if err != nil {
		log.Fatalf("Failed to execute url template: %s", err)
	}
	u, err := url.Parse(urlString.String())
	if err != nil {
		log.Fatalf("Generated an invalid URL: %s", err)
	}
	imageURL := u.String()
	// language=SQL
	_, err = db.Exec(context.Background(), `update image set url=$1, path=$2 where id=$3`, imageURL, u.Path, imageID)
	if err != nil {
		log.Fatalf("Failed to update image: %s", err)
	}
	return imageURL, imageID
}